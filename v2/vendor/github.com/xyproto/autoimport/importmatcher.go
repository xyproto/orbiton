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
	JARPaths []string          // list of paths to examine for .jar files
	onlyJava bool              // only Java, or Kotlin too?
	mut      sync.RWMutex      // mutex for protecting the map
	classMap map[string]string // map from class name to class path. Shortest class path "wins".
}

// New creates a new ImportMatcher. If onlyJava is false, /usr/share/kotlin/lib will be added to the .jar file search path.
func New(onlyJava bool) (*ImportMatcher, error) {
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
	return NewCustom(JARSearchPaths, onlyJava)
}

// addDir adds a directory to the current slice of paths to search for .jar files
func (impM *ImportMatcher) addDir(path string) {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	impM.JARPaths = append(impM.JARPaths, path)
}

// NewCustom creates a new ImportMatcher, given a slice of paths to search for .jar files
func NewCustom(JARPaths []string, onlyJava bool) (*ImportMatcher, error) {
	var impM ImportMatcher
	impM.onlyJava = onlyJava

	impM.JARPaths = make([]string, 0)
	for _, path := range JARPaths {
		if isSymlink(path) {
			// follow the symlink, once
			path = followSymlink(path)
			// if the path is a directory, collect it
			if isDir(path) {
				impM.addDir(path)
				continue
			}
			// follow the symlink, repeatedly
			for isSymlink(path) {
				path = followSymlink(path)
			}
			// if the path is a directory, collect it
			if isDir(path) {
				impM.addDir(path)
			}
		} else if isDir(path) {
			impM.addDir(path)
		}
	}

	if len(impM.JARPaths) == 0 {
		return nil, errors.New("no paths to search for JAR files")
	}

	impM.classMap = make(map[string]string)

	found := make(chan string)
	done := make(chan bool)

	go impM.produceClasses(found)
	go impM.consumeClasses(found, done)
	<-done

	return &impM, nil
}

// ClassMap returns the mapping from class names to class paths
func (impM *ImportMatcher) ClassMap() map[string]string {
	return impM.classMap
}

// readSOURCE returns a list of classes within the given src.zip file,
// for instance "some.package.name.SomeClass"
func (impM *ImportMatcher) readSOURCE(filePath string, found chan string) {
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
func (impM *ImportMatcher) readJAR(filePath string, found chan string) {
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
func (impM *ImportMatcher) findClassesInJarOrSrc(JARPath string, found chan string) {
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
				impM.readJAR(filePath, found)
				wg.Done()
			}(filePath)
			return err
		} else if filepath.Base(filePath) == "src.zip" && filepath.Base(filepath.Dir(filePath)) == "lib" {
			wg.Add(1)
			go func(filePath string) {
				impM.readSOURCE(filePath, found)
				wg.Done()
			}(filePath)
			return err
		}

		return nil
	})
	wg.Wait()
}

func (impM *ImportMatcher) produceClasses(found chan string) {
	var wg sync.WaitGroup
	for _, JARPath := range impM.JARPaths {
		// fmt.Printf("About to search for .jar files in %s...\n", JARPath)
		wg.Add(1)
		go func(path string) {
			impM.findClassesInJarOrSrc(path, found)
			wg.Done()
		}(JARPath)
	}
	wg.Wait()
	close(found)
}

func (impM *ImportMatcher) consumeClasses(found <-chan string, done chan<- bool) {
	for classPath := range found {

		// Let className be classPath by default, in case the replacements doesn't go through
		className := classPath
		if strings.Contains(classPath, ".") {
			fields := strings.Split(classPath, ".")
			lastField := fields[len(fields)-1]
			className = lastField
		}

		// Check if the same or a shorter class name name already exists. Also prioritize class paths that does not start with "sun.".
		impM.mut.RLock()
		if existingClassPath, ok := impM.classMap[className]; ok && existingClassPath != "" && ((len(existingClassPath) <= len(classPath)) || (!strings.HasPrefix(existingClassPath, "sun.") && strings.HasPrefix(classPath, "sun."))) {
			impM.mut.RUnlock()
			continue
		}
		impM.mut.RUnlock()

		// fmt.Println("classPath", classPath)

		// Store the new class name and class path
		impM.mut.Lock()
		impM.classMap[className] = classPath
		impM.mut.Unlock()
	}
	done <- true
}

func (impM *ImportMatcher) String() string {
	var sb strings.Builder

	impM.mut.RLock()
	for className, classPath := range impM.classMap {
		sb.WriteString(className + ": " + classPath + "\n")
	}
	impM.mut.RUnlock()

	return sb.String()
}

// StarPath takes the start of the class name and tries to return the shortest
// found class name, and also the import path like "java.io.*"
// Returns empty strings if there are no matches.
func (impM *ImportMatcher) StarPath(startOfClassName string) (string, string) {
	shortestClassName := ""
	shortestImportPath := ""
	for className, classPath := range impM.classMap {
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
func (impM *ImportMatcher) StarPathExact(exactClassName string) string {
	shortestClassName := ""
	shortestImportPath := ""
	for className, classPath := range impM.classMap {
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
func (impM *ImportMatcher) ImportPathExact(exactClassName string) string {
	shortestClassName := ""
	shortestImportPath := ""
	for className, classPath := range impM.classMap {
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
func (impM *ImportMatcher) StarPathAll(startOfClassName string) ([]string, []string) {
	allClassNames := make([]string, 0)
	allImportPaths := make([]string, 0)
	for className, classPath := range impM.classMap {
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
func (impM *ImportMatcher) StarPathAllExact(exactClassName string) ([]string, []string) {
	allClassNames := make([]string, 0)
	allImportPaths := make([]string, 0)
	for className, classPath := range impM.classMap {
		if className == exactClassName {
			allClassNames = append(allClassNames, className)
			allImportPaths = append(allImportPaths, strings.Replace(classPath, className, "*", 1))
		}
	}
	return allClassNames, allImportPaths
}
