// Package autoimport tries to find which import should be used, given the start of a class name
package autoimport

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

// ImportMatcher is a struct that contains a list of JAR file paths,
// and a lookup map from class names to class paths, which is populated
// when New or NewCustom is called.
type ImportMatcher struct {
	classMap              map[string]string // map from class name to class path. Shortest class path "wins".
	JARPaths              []string          // list of paths to examine for .jar files
	mut                   sync.RWMutex      // mutex for protecting the map
	onlyJava              bool              // only Java, or Kotlin too?
	removeExistingImports bool              // keep existing imports (but also avoid duplicates)
}

// New creates a new ImportMatcher. If onlyJava is false, /usr/share/kotlin/lib will be added to the .jar file search path.
// The first (optional) bool should be set to true if only Java should be considered, and not Kotlin.
// The second (optional) bool should be set to true if the import organizer should always start out with removing existing imports.
func New(settings ...bool) (*ImportMatcher, error) {

	var onlyJava bool
	if len(settings) > 0 {
		onlyJava = settings[0]
	}

	javaHomePath, err := FindJava()
	if err != nil {
		return nil, err
	}
	JARSearchPaths := []string{javaHomePath}
	if !onlyJava {
		kotlinPath, err := FindKotlin()
		if err != nil {
			return nil, err
		}
		JARSearchPaths = append(JARSearchPaths, kotlinPath)
	}

	return NewCustom(JARSearchPaths, settings...)
}

// addDir adds a directory to the current slice of paths to search for .jar files
func (ima *ImportMatcher) addDir(path string) {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	ima.JARPaths = append(ima.JARPaths, path)
}

// NewCustom creates a new ImportMatcher, given a slice of paths to search for .jar files
// The first (optional) bool should be set to true if only Java should be considered, and not Kotlin.
// The second (optional) bool should be set to true if the import organizer should always start out with removing existing imports.
func NewCustom(JARPaths []string, settings ...bool) (*ImportMatcher, error) {
	var ima ImportMatcher

	if len(settings) > 0 {
		ima.onlyJava = settings[0]
	}

	if len(settings) > 1 {
		ima.removeExistingImports = settings[1]
	}

	ima.JARPaths = make([]string, 0)
	for _, path := range JARPaths {
		if isSymlink(path) {
			// follow the symlink, once
			path = followSymlink(path)
			// if the path is a directory, collect it
			if isDir(path) {
				ima.addDir(path)
				continue
			}
			// follow the symlink, repeatedly
			for isSymlink(path) {
				path = followSymlink(path)
			}
			// if the path is a directory, collect it
			if isDir(path) {
				ima.addDir(path)
			}
		} else if isDir(path) {
			ima.addDir(path)
		}
	}

	if len(ima.JARPaths) == 0 {
		return nil, errors.New("no paths to search for JAR files")
	}

	ima.classMap = make(map[string]string)

	found := make(chan string)
	done := make(chan bool)

	go ima.produceClasses(found)
	go ima.consumeClasses(found, done)
	<-done

	return &ima, nil
}

// ClassMap returns the mapping from class names to class paths
func (ima *ImportMatcher) ClassMap() map[string]string {
	return ima.classMap
}

// readSOURCE returns a list of classes within the given src.zip file,
// for instance "some.package.name.SomeClass"
func (ima *ImportMatcher) readSOURCE(filePath string, found chan string) {
	readCloser, err := zip.OpenReader(filePath)
	if err != nil {
		return
	}
	defer readCloser.Close()

	for _, f := range readCloser.File {
		fileName := f.Name
		if strings.HasSuffix(fileName, ".java") || strings.HasSuffix(fileName, ".JAVA") {

			// The class name is derived from the .java path within the src.zip file

			className := strings.TrimSuffix(strings.TrimSuffix(fileName, ".java"), ".JAVA")
			className = strings.ReplaceAll(className, "/", ".")
			className = strings.TrimPrefix(className, "java.base.")
			className = strings.TrimPrefix(className, "jdk.internal.")
			if className == "" {
				continue
			}

			// Filter out class names that are only lowercase (and '.')
			allLower := true
			for _, r := range className {
				if !unicode.IsLower(r) && r != '.' {
					allLower = false
				}
			}
			if allLower {
				continue
			}

			found <- className
		}
	}
}

// readJAR returns a list of classes within the given .jar file,
// for instance "some.package.name.SomeClass"
func (ima *ImportMatcher) readJAR(filePath string, found chan string) {
	readCloser, err := zip.OpenReader(filePath)
	if err != nil {
		return
	}
	defer readCloser.Close()

	for _, f := range readCloser.File {
		fileName := f.Name
		if strings.HasSuffix(fileName, ".class") || strings.HasSuffix(fileName, ".CLASS") {

			// The class name is derived from the .class path within the jar file

			className := strings.TrimSuffix(strings.TrimSuffix(fileName, ".class"), ".CLASS")
			className = strings.ReplaceAll(className, "/", ".")
			className = strings.TrimSuffix(className, "$1")
			className = strings.TrimSuffix(className, "$1")
			if pos := strings.Index(className, "$"); pos >= 0 {
				className = className[:pos]
			}

			if className == "" {
				continue
			}

			// Filter out class names that are only lowercase (and '.')
			allLower := true
			for _, r := range className {
				if !unicode.IsLower(r) && r != '.' {
					allLower = false
				}
			}
			if allLower {
				continue
			}

			found <- className
		}
	}
}

// findClassesInJarOrSrc will search the given JAR path for JAR files,
// and then search each JAR file for for classes.
// Found classes will be sent to the found chan.
// Will also search "*/lib/src.zip" files.
func (ima *ImportMatcher) findClassesInJarOrSrc(JARPath string, found chan string) {
	var wg sync.WaitGroup
	filepath.Walk(JARPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(path, "/demo/") {
			return nil
		}

		fileName := info.Name()
		filePath := path

		if filepath.Ext(fileName) == ".jar" || filepath.Ext(fileName) == ".JAR" {
			wg.Add(1)
			go func(filePath string) {
				ima.readJAR(filePath, found)
				wg.Done()
			}(filePath)
			return err
		} else if filepath.Base(filePath) == "src.zip" && filepath.Base(filepath.Dir(filePath)) == "lib" {
			wg.Add(1)
			go func(filePath string) {
				ima.readSOURCE(filePath, found)
				wg.Done()
			}(filePath)
			return err
		}

		return nil
	})
	wg.Wait()
}

func (ima *ImportMatcher) produceClasses(found chan string) {
	var wg sync.WaitGroup
	for _, JARPath := range ima.JARPaths {
		// fmt.Printf("About to search for .jar files in %s...\n", JARPath)
		wg.Add(1)
		go func(path string) {
			ima.findClassesInJarOrSrc(path, found)
			wg.Done()
		}(JARPath)
	}
	wg.Wait()
	close(found)
}

func (ima *ImportMatcher) consumeClasses(found <-chan string, done chan<- bool) {
	for classPath := range found {

		// Let className be classPath by default, in case the replacements doesn't go through
		className := classPath
		if strings.Contains(classPath, ".") {
			fields := strings.Split(classPath, ".")
			lastField := fields[len(fields)-1]
			className = lastField
		}

		// Check if the same or a shorter class name name already exists. Also prioritize class paths that does not start with "sun.".
		ima.mut.RLock()
		if existingClassPath, ok := ima.classMap[className]; ok && existingClassPath != "" && ((len(existingClassPath) <= len(classPath)) || (!strings.HasPrefix(existingClassPath, "sun.") && strings.HasPrefix(classPath, "sun."))) {
			ima.mut.RUnlock()
			continue
		}
		ima.mut.RUnlock()

		// fmt.Println("classPath", classPath)

		// Store the new class name and class path
		ima.mut.Lock()
		ima.classMap[className] = classPath
		ima.mut.Unlock()
	}
	done <- true
}

func (ima *ImportMatcher) String() string {
	var sb strings.Builder

	ima.mut.RLock()
	for className, classPath := range ima.classMap {
		sb.WriteString(className + ": " + classPath + "\n")
	}
	ima.mut.RUnlock()

	return sb.String()
}

// StarPath takes the start of the class name and tries to return the shortest
// found class name, and also the import path like "java.io.*"
// Returns empty strings if there are no matches.
func (ima *ImportMatcher) StarPath(startOfClassName string) (string, string) {
	shortestClassName := ""
	shortestImportPath := ""
	for className, classPath := range ima.classMap {
		if strings.HasPrefix(className, startOfClassName) {
			if shortestClassName == "" || len(className) < len(shortestClassName) {
				shortestClassName = className
				shortestImportPath = strings.Replace(classPath, className, "*", 1)
			} else if len(className) == len(shortestClassName) {
				importPath := strings.Replace(classPath, className, "*", 1)
				if shortestImportPath == "" || len(importPath) < len(shortestImportPath) {
					shortestClassName = className
					shortestImportPath = importPath
				}
			}
		}
	}
	return shortestClassName, shortestImportPath
}

// StarPathExact takes the exact class name and tries to return the shortest
// import path for the matching class, if found, like "java.io.*".
// Returns empty string if there are no matches.
func (ima *ImportMatcher) StarPathExact(exactClassName string) string {
	shortestClassName := ""
	shortestImportPath := ""
	for className, classPath := range ima.classMap {
		if className == exactClassName {
			if shortestClassName == "" {
				shortestClassName = className
				shortestImportPath = strings.Replace(classPath, className, "*", 1)
			} else if len(className) == len(shortestClassName) {
				importPath := strings.Replace(classPath, className, "*", 1)
				if shortestImportPath == "" || len(importPath) < len(shortestImportPath) {
					shortestClassName = className
					shortestImportPath = importPath
				}
			}
		}
	}
	return shortestImportPath
}

// ImportPathExact takes the exact class name and tries to return the shortest
// specific import path for the matching class. For example, "File" could result
// in "java.io.File". The function returns an empty string if there are no matches.
func (ima *ImportMatcher) ImportPathExact(exactClassName string) string {
	shortestClassName := ""
	shortestImportPath := ""
	for className, classPath := range ima.classMap {
		if className == exactClassName {
			if shortestClassName == "" {
				shortestClassName = className
				shortestImportPath = classPath
			} else if len(className) == len(shortestClassName) {
				importPath := classPath
				if shortestImportPath == "" || len(importPath) < len(shortestImportPath) {
					shortestClassName = className
					shortestImportPath = importPath
				}
			}
		}
	}
	return shortestImportPath
}

// StarPathAll takes the start of the class name and tries to return all
// found class names, and also the import paths, like "java.io.*".
// Returns empty strings if there are no matches.
func (ima *ImportMatcher) StarPathAll(startOfClassName string) ([]string, []string) {
	allClassNames := make([]string, 0)
	allImportPaths := make([]string, 0)
	for className, classPath := range ima.classMap {
		if strings.HasPrefix(className, startOfClassName) {
			allClassNames = append(allClassNames, className)
			allImportPaths = append(allImportPaths, strings.Replace(classPath, className, "*", 1))
		}
	}
	return allClassNames, allImportPaths
}

// StarPathAllExact takes the exact class name and tries to return all
// matching class names, and also the import paths, like "java.io.*".
// Returns empty strings if there are no matches.
func (ima *ImportMatcher) StarPathAllExact(exactClassName string) ([]string, []string) {
	allClassNames := make([]string, 0)
	allImportPaths := make([]string, 0)
	for className, classPath := range ima.classMap {
		if className == exactClassName {
			allClassNames = append(allClassNames, className)
			allImportPaths = append(allImportPaths, strings.Replace(classPath, className, "*", 1))
		}
	}
	return allClassNames, allImportPaths
}
