package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xyproto/files"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

var (
	errNoSuitableBuildCommand = errors.New("no suitable build command")
	pandocMutex               sync.RWMutex
)

// exeName tries to find a suitable name for the executable, given a source filename
// For instance, "main" or the name of the directory holding the source filename.
// If shouldExist is true, the function will try to select either "main" or the parent
// directory name, depending on which one is there.
func (e *Editor) exeName(sourceFilename string, shouldExist bool) string {
	exeFirstName := "main" // The default name
	sourceDir := filepath.Dir(sourceFilename)

	// NOTE: Abs is used to prevent sourceDirectoryName from becoming just "."
	absDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return exeFirstName
	}

	sourceDirectoryName := filepath.Base(absDir)

	if shouldExist {
		// If "main" exists, use that
		if files.IsFile(filepath.Join(sourceDir, exeFirstName)) {
			return exeFirstName
		}
		// If the name of the source file, without the extension, exists, use that
		exeFirstName = strings.TrimSuffix(filepath.Base(sourceFilename), filepath.Ext(sourceFilename))
		if files.IsFile(filepath.Join(sourceDir, exeFirstName)) {
			return exeFirstName
		}
		// Use the name of the source directory as the default executable filename instead
		if files.IsFile(filepath.Join(sourceDir, sourceDirectoryName)) {
			// exeFirstName = sourceDirectoryName
			return sourceDirectoryName
		}
	}

	// Find a suitable default executable first name
	switch e.mode {
	case mode.Assembly, mode.Kotlin, mode.Lua, mode.OCaml, mode.Rust, mode.Terra, mode.Zig:
		if sourceDirectoryName == "build" {
			parentDirName := filepath.Base(filepath.Clean(filepath.Join(sourceDir, "..")))
			if shouldExist && files.IsFile(filepath.Join(sourceDir, parentDirName)) {
				return parentDirName
			}
		}
		// Default to the source directory base name, for these programming languages
		return sourceDirectoryName
	case mode.Odin:
		if shouldExist && files.IsFile(filepath.Join(sourceDir, sourceDirectoryName+".bin")) {
			return sourceDirectoryName + ".bin"
		}
		// Default to just the source directory base name
		return sourceDirectoryName
	}

	// Use the name of the current directory, if a file with that name exists
	if shouldExist && files.IsFile(filepath.Join(sourceDir, sourceDirectoryName)) {
		return sourceDirectoryName
	}

	// Default to "main"
	return exeFirstName
}

func has(executableInPath string) bool {
	return files.WhichCached(executableInPath) != ""
}

// GenerateBuildCommand will generate a command for building the given filename (or for displaying HTML)
// If there are no errors, a exec.Cmd is returned together with a function that can tell if the build
// produced an executable, together with the executable name,
func (e *Editor) GenerateBuildCommand(c *vt100.Canvas, tty *vt100.TTY, filename string) (*exec.Cmd, func() (bool, string), error) {
	var cmd *exec.Cmd

	// A function that signals that everything is fine, regardless of if an executable is produced or not, after building
	everythingIsFine := func() (bool, string) {
		return true, "everything"
	}

	// A function that signals that something is wrong, regardless of if an executable is produced or not, after building
	nothingIsFine := func() (bool, string) {
		return false, "nothing"
	}

	// Find the absolute path to the source file
	sourceFilename, err := filepath.Abs(filename)
	if err != nil {
		return cmd, nothingIsFine, err
	}

	// Set up a few basic variables about the given source file
	var (
		sourceDir      = filepath.Dir(sourceFilename)
		parentDir      = filepath.Clean(filepath.Join(sourceDir, ".."))
		grandParentDir = filepath.Clean(filepath.Join(sourceDir, "..", ".."))
		exeFirstName   = e.exeName(sourceFilename, false)
		exeFilename    = filepath.Join(sourceDir, exeFirstName)
		jarFilename    = exeFirstName + ".jar"
		kokaBuildDir   = filepath.Join(userCacheDir, "o", "koka")
		pyCacheDir     = filepath.Join(userCacheDir, "o", "python")
		zigCacheDir    = filepath.Join(userCacheDir, "o", "zig")
	)

	if noWriteToCache {
		kokaBuildDir = filepath.Join(sourceDir, "o", "koka")
		pyCacheDir = filepath.Join(sourceDir, "o", "python")
		zigCacheDir = filepath.Join(sourceDir, "o", "zig")
	}

	exeExists := func() (bool, string) {
		// Check if exeFirstName exists
		return files.IsFile(filepath.Join(sourceDir, exeFirstName)), exeFirstName
	}

	exeOrMainExists := func() (bool, string) {
		// First check if exeFirstName exists
		if files.IsFile(filepath.Join(sourceDir, exeFirstName)) {
			return true, exeFirstName
		}
		// Then try with just "main"
		return files.IsFile(filepath.Join(sourceDir, "main")), "main"
	}

	exeBaseNameOrMainExists := func() (bool, string) {
		// First check if exeFirstName exists
		if files.IsFile(filepath.Join(sourceDir, exeFirstName)) {
			return true, exeFirstName
		}
		// Then try with the current directory name
		baseDirName := filepath.Base(sourceDir)
		if files.IsFile(filepath.Join(sourceDir, baseDirName)) {
			return true, baseDirName
		}
		// The try with just "main"
		if files.IsFile(filepath.Join(sourceDir, "main")) {
			return true, "main"
		}
		return false, ""
	}

	switch filepath.Base(sourceFilename) {
	case "CMakeLists.txt":
		var s string
		if has("cmake") {
			s = "cmake -B build -D CMAKE_BUILD_TYPE=Debug -G Ninja -S . -W no-dev || (rm -rv build; cmake -B build -D CMAKE_BUILD_TYPE=Debug -G Ninja -S . -W no-dev) && ninja -C build"
			if !has("ninja") {
				s = strings.ReplaceAll(s, " -G Ninja", "")
				s = strings.ReplaceAll(s, "ninja -C ", "make -C ")
			}
			if isBSD && has("gmake") {
				s = strings.ReplaceAll(s, "make -C ", "gmake -C ")
			}
		}
		lastCommand, err := readLastCommand()
		if releaseBuildFlag {
			if err == nil && strings.Contains(lastCommand, "CMAKE_BUILD_TYPE=Debug ") {
				s = "rm -r build; " + s
			}
			s = strings.ReplaceAll(s, "CMAKE_BUILD_TYPE=Debug ", "CMAKE_BUILD_TYPE=Release ")
		} else {
			if err == nil && strings.Contains(lastCommand, "CMAKE_BUILD_TYPE=Release ") {
				s = "rm -r build; " + s
			}
		}
		if s != "" {
			// Save and exec / replace the process with syscall.Exec
			if e.Save(c, tty) == nil { // success
				// Unlock and save the lock file
				if absFilename, err := filepath.Abs(e.filename); fileLock != nil && err == nil { // success
					fileLock.Unlock(absFilename)
					fileLock.Save()
				}
				quitExecShellCommand(tty, sourceDir, s) // The program ends here
			}
			// Could not save the file, execute the command in a separate process
			args := strings.Split(s, " ")
			cmd = exec.Command(args[0], args[1:]...)
			cmd.Dir = sourceDir
			return cmd, everythingIsFine, nil
		}
	case "PKGBUILD":
		var s string
		if has("tinyionice") {
			s += "tinyionice "
		} else if has("ionice") {
			s += "ionice"
		}
		foundCommand := false
		if has("pkgctl") { // extrabuild
			s += "pkgctl build"
			foundCommand = true
		} else if has("makepkg") {
			s += "makepkg"
			foundCommand = true
		}
		if foundCommand {
			// Save and exec / replace the process with syscall.Exec
			if e.Save(c, tty) == nil { // success
				// Unlock and save the lock file
				if absFilename, err := filepath.Abs(e.filename); fileLock != nil && err == nil { // success
					fileLock.Unlock(absFilename)
					fileLock.Save()
				}
				quitExecShellCommand(tty, sourceDir, s) // The program ends here
			}
			// Could not save the file, execute the command in a separate process
			args := strings.Split(s, " ")
			cmd = exec.Command(args[0], args[1:]...)
			cmd.Dir = sourceDir
			return cmd, everythingIsFine, nil
		}
	}

	switch e.mode {
	case mode.ABC:
		cmd = exec.Command("abc2midi", e.filename, "-o", filepath.Join(tempDir, "o.mid"))
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Make:
		cmd = exec.Command("make")
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Java: // build a .jar file
		javaShellCommand := "javaFiles=$(find . -type f -name '*.java'); " +
			"for f in $javaFiles; do grep -q 'static void main' \"$f\" && mainJavaFile=\"$f\"; done; " +
			"if command -v grep >/dev/null 2>&1 && echo 'test' | grep -P 'test' >/dev/null 2>&1; then " +
			"className=$(grep -oP '(?<=class )[A-Z][a-zA-Z0-9]*' \"$mainJavaFile\" | head -1); " +
			"packageName=$(grep -oP '(?<=package )[a-zA-Z0-9.]*' \"$mainJavaFile\" | head -1); " +
			"else " +
			"className=$(grep -E 'class [A-Z][a-zA-Z0-9]*' \"$mainJavaFile\" | sed -E 's/.*class ([A-Z][a-zA-Z0-9]*).*/\\1/' | head -1); " +
			"packageName=$(grep -E 'package [a-zA-Z0-9.]*' \"$mainJavaFile\" | sed -E 's/.*package ([a-zA-Z0-9.]*).*/\\1/' | head -1); " +
			"fi; " +
			"if [[ $packageName != \"\" ]]; then packageName=\"$packageName.\"; fi; " +
			"mkdir -p _o_build/META-INF; " +
			"javac -d _o_build $javaFiles; " +
			"cd _o_build; " +
			"echo \"Main-Class: $packageName$className\" > META-INF/MANIFEST.MF; " +
			"classFiles=$(find . -type f -name '*.class'); " +
			"jar cmf META-INF/MANIFEST.MF ../" + jarFilename + " $classFiles; " +
			"cd ..; " +
			"rm -rf _o_build"
		cmd = exec.Command("sh", "-c", javaShellCommand)
		cmd.Dir = sourceDir
		return cmd, func() (bool, string) {
			return files.IsFile(filepath.Join(sourceDir, jarFilename)), jarFilename
		}, nil

	case mode.Scala:
		if files.IsFile(filepath.Join(sourceDir, "build.sbt")) && files.WhichCached("sbt") != "" && files.FileHas(filepath.Join(sourceDir, "build.sbt"), "ScalaNative") {
			cmd = exec.Command("sbt", "nativeLink")
			cmd.Dir = sourceDir
			return cmd, func() (bool, string) {
				// TODO: Check for /scala-*/scalanative-out and not scala-3.3.0 specifically
				return files.Exists(filepath.Join(sourceDir, "target", "scala-3.3.0", "scalanative-out")), "target/scala-3.3.0/scalanative-out"
			}, nil
		}
		// For building a .jar file that can not be run with "java -jar main.jar" but with "scala main.jar": scalac -jar main.jar Hello.scala
		scalaShellCommand := "scalaFiles=$(find . -type f -name '*.scala'); for f in $scalaFiles; do grep -q 'def main' \"$f\" && mainScalaFile=\"$f\"; grep -q ' extends App ' \"$f\" && mainScalaFile=\"$f\"; done; objectName=$(grep -oP '(?<=object )[A-Z]+[a-z,A-Z,0-9]*' \"$mainScalaFile\" | head -1); packageName=$(grep -oP '(?<=package )[a-z,A-Z,0-9,.]*' \"$mainScalaFile\" | head -1); if [[ $packageName != \"\" ]]; then packageName=\"$packageName.\"; fi; mkdir -p _o_build/META-INF; scalac -d _o_build $scalaFiles; cd _o_build; echo -e \"Main-Class: $packageName$objectName\\nClass-Path: /usr/share/scala/lib/scala-library.jar\" > META-INF/MANIFEST.MF; classFiles=$(find . -type f -name '*.class'); jar cmf META-INF/MANIFEST.MF ../" + jarFilename + " $classFiles; cd ..; rm -rf _o_build"
		// Compile directly to jar with scalac if /usr/share/scala/lib/scala-library.jar is not found
		if !files.IsFile("/usr/share/scala/lib/scala-library.jar") {
			scalaShellCommand = "scalac -d run_with_scala.jar $(find . -type f -name '*.scala')"
		}
		cmd = exec.Command("sh", "-c", scalaShellCommand)
		cmd.Dir = sourceDir
		return cmd, func() (bool, string) {
			return files.IsFile(filepath.Join(sourceDir, jarFilename)), jarFilename
		}, nil
	case mode.Kotlin:
		if files.WhichCached("kotlinc-native") != "" && strings.Contains(e.String(), "import kotlinx.cinterop.") {
			cmd = exec.Command("kotlinc-native", "-nowarn", "-opt", "-Xallocator=mimalloc", "-produce", "program", "-linker-option", "--as-needed", sourceFilename, "-o", exeFirstName)
			cmd.Dir = sourceDir
			return cmd, func() (bool, string) {
				if files.IsFile(filepath.Join(sourceDir, exeFirstName+".kexe")) {
					return true, exeFirstName + ".kexe"
				}
				return files.IsFile(filepath.Join(sourceDir, exeFirstName)), exeFirstName
			}, nil
		}
		cmd = exec.Command("kotlinc", sourceFilename, "-include-runtime", "-d", jarFilename)
		cmd.Dir = sourceDir
		return cmd, func() (bool, string) {
			return files.IsFile(filepath.Join(sourceDir, jarFilename)), jarFilename
		}, nil
	case mode.Inko:
		cmd := exec.Command("inko", "build", "-o", exeFirstName, sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Go:
		// TODO: Make this code more elegant, and consider searching all parent directories
		hasGoMod := files.IsFile(filepath.Join(sourceDir, "go.mod")) ||
			files.IsFile(filepath.Join(sourceDir, "..", "go.mod")) ||
			files.IsFile(filepath.Join(sourceDir, "..", "..", "go.mod"))
		if hasGoMod {
			cmd = exec.Command("go", "build")
		} else {
			cmd = exec.Command("go", "build", sourceFilename)
		}
		if strings.HasSuffix(sourceFilename, "_test.go") {
			// go test run a test that does not exist in order to build just the tests
			// thanks @cespare at github https://github.com/golang/go/issues/15513#issuecomment-216410016
			cmd = exec.Command("go", "test", "-run", "xxxxxxx")
		}
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Hare:
		cmd := exec.Command("hare", "build")
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Shader:
		sourceFilenameWithoutExt := strings.TrimSuffix(sourceFilename, filepath.Ext(sourceFilename))
		shaderCode := e.String()
		shaderType, err := detectShaderType(shaderCode)
		if err != nil {
			shaderType = "frag" // default to compiling fragment shaders to spirv, the alternative is "vert"
		}
		cmd := exec.Command("glslangValidator", "-V", "-S", shaderType, "-o", sourceFilenameWithoutExt+".spv", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Algol68:
		sourceFilenameWithoutExt := strings.TrimSuffix(sourceFilename, filepath.Ext(sourceFilename))
		sourceFilenameWithoutPath := filepath.Base(sourceFilename)
		cmd := exec.Command("a68g", "--compile", sourceFilenameWithoutPath)
		cmd.Dir = sourceDir
		return cmd, func() (bool, string) {
			executableFirstName := filepath.Base(sourceFilenameWithoutExt)
			return files.IsFile(sourceFilenameWithoutPath), executableFirstName
		}, nil
	case mode.C:
		if files.WhichCached("cxx") != "" {
			cmd = exec.Command("cxx")
			cmd.Dir = sourceDir
			if e.debugMode {
				cmd.Args = append(cmd.Args, "debugnosan")
			}
			return cmd, exeBaseNameOrMainExists, nil
		}
		if files.IsDir(exeFilename) {
			exeFilename = "main"
		}
		// Use gcc directly
		if e.debugMode {
			cmd = exec.Command("gcc", "-o", exeFilename, "-Og", "-g", "-pipe", "-D_BSD_SOURCE", sourceFilename)
			cmd.Dir = sourceDir
			return cmd, exeOrMainExists, nil
		}
		cmd = exec.Command("gcc", "-o", exeFilename, "-O2", "-pipe", "-fPIC", "-fno-plt", "-fstack-protector-strong", "-D_BSD_SOURCE", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, exeOrMainExists, nil
	case mode.Cpp:
		if files.IsFile("BUILD.bazel") && files.WhichCached("bazel") != "" { // Google-style C++ + Bazel projects if
			return exec.Command("bazel", "build"), everythingIsFine, nil
		}
		if files.WhichCached("cxx") != "" {
			cmd = exec.Command("cxx")
			cmd.Dir = sourceDir
			if e.debugMode {
				cmd.Args = append(cmd.Args, "debugnosan")
			}
			return cmd, exeBaseNameOrMainExists, nil
		}
		if files.IsDir(exeFilename) {
			exeFilename = "main"
		}
		// Use g++ directly
		if e.debugMode {
			cmd = exec.Command("g++", "-o", exeFilename, "-Og", "-g", "-pipe", "-Wall", "-Wshadow", "-Wpedantic", "-Wno-parentheses", "-Wfatal-errors", "-Wvla", "-Wignored-qualifiers", sourceFilename)
			cmd.Dir = sourceDir
			return cmd, exeOrMainExists, nil
		}
		cmd = exec.Command("g++", "-o", exeFilename, "-O2", "-pipe", "-fPIC", "-fno-plt", "-fstack-protector-strong", "-Wall", "-Wshadow", "-Wpedantic", "-Wno-parentheses", "-Wfatal-errors", "-Wvla", "-Wignored-qualifiers", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, exeOrMainExists, nil
	case mode.Zig:
		if files.WhichCached("zig") != "" {
			if files.IsFile("build.zig") {
				cmd = exec.Command("zig", "build")
				cmd.Dir = sourceDir
				return cmd, everythingIsFine, nil
			}
			// Just build the current file
			sourceCode := ""
			sourceData, err := os.ReadFile(sourceFilename)
			if err == nil { // success
				sourceCode = string(sourceData)
			}

			cmd = exec.Command("zig", "build-exe", "-lc", sourceFilename, "--name", exeFirstName, "--cache-dir", zigCacheDir)
			cmd.Dir = sourceDir
			// TODO: Find a better way than this
			if strings.Contains(sourceCode, "SDL2/SDL.h") {
				cmd.Args = append(cmd.Args, "-lSDL2")
			}
			if strings.Contains(sourceCode, "gmp.h") {
				cmd.Args = append(cmd.Args, "-lgmp")
			}
			if strings.Contains(sourceCode, "glfw") {
				cmd.Args = append(cmd.Args, "-lglfw")
			}
			return cmd, exeExists, nil
		}
		// No result
	case mode.V:
		cmd = exec.Command("v", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, exeOrMainExists, nil
	case mode.Garnet:
		cmd = exec.Command("garnetc", "-o", exeFirstName, sourceFilename)
		cmd.Dir = sourceDir
		return cmd, exeExists, nil
	case mode.Rust:
		if e.debugMode {
			cmd = exec.Command("cargo", "build", "--profile", "dev")
		} else {
			cmd = exec.Command("cargo", "build", "--profile", "release")
		}
		if files.IsFile("Cargo.toml") {
			cmd.Dir = sourceDir
			return cmd, everythingIsFine, nil
		}
		if files.IsFile(filepath.Join(parentDir, "Cargo.toml")) {
			cmd.Dir = parentDir
			return cmd, everythingIsFine, nil
		}
		// Use rustc instead of cargo if Cargo.toml is missing
		if rustcExecutable := files.WhichCached("rustc"); rustcExecutable != "" {
			if e.debugMode {
				cmd = exec.Command(rustcExecutable, sourceFilename, "-g", "-o", exeFilename)
			} else {
				cmd = exec.Command(rustcExecutable, sourceFilename, "-o", exeFilename)
			}
			cmd.Dir = sourceDir
			return cmd, exeExists, nil
		}
		// No result
	case mode.C3:
		if e.debugMode {
			cmd = exec.Command("c3c", "compile", "-o", exeFilename, ".")
		} else {
			cmd = exec.Command("c3c", "compile", "-g", "-o", exeFilename, ".")
		}
		cmd.Dir = sourceDir
		return cmd, exeExists, nil
	case mode.Clojure:
		cmd = exec.Command("lein", "uberjar")
		projectFileExists := files.IsFile("project.clj")
		parentProjectFileExists := files.IsFile("../project.clj")
		grandParentProjectFileExists := files.IsFile("../../project.clj")
		cmd.Dir = sourceDir
		if !projectFileExists && parentProjectFileExists {
			cmd.Dir = parentDir
		} else if !projectFileExists && !parentProjectFileExists && grandParentProjectFileExists {
			cmd.Dir = grandParentDir
		} else if !projectFileExists && !parentProjectFileExists && !grandParentProjectFileExists {
			cmd = exec.Command("clojure", "-e", `(try (clojure.core/read-string (slurp "`+sourceFilename+`")) (println "Syntax OK") (catch Exception e (println "Syntax error:" (.getMessage e))))`)
			cmd.Dir = sourceDir
		}
		return cmd, everythingIsFine, nil
	case mode.Haskell:
		cmd = exec.Command("ghc", "-dynamic", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Python:
		if isDarwin {
			cmd = exec.Command("python3", "-m", "py_compile", sourceFilename)
		} else {
			cmd = exec.Command("python", "-m", "py_compile", sourceFilename)
		}
		cmd.Env = append(cmd.Env, "PYTHONUTF8=1")
		if !files.Exists(pyCacheDir) {
			os.MkdirAll(pyCacheDir, 0o700)
		}
		cmd.Env = append(cmd.Env, "PYTHONPYCACHEPREFIX="+pyCacheDir)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.OCaml:
		cmd = exec.Command("ocamlopt", "-o", exeFirstName, sourceFilename)
		cmd.Dir = sourceDir
		return cmd, exeExists, nil
	case mode.Crystal:
		cmd = exec.Command("crystal", "build", "--no-color", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Dart:
		cmd = exec.Command("dart", "compile", "exe", "--verbosity", "error", "-o", exeFirstName, sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Erlang:
		cmd = exec.Command("erlc", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Fortran77, mode.Fortran90:
		cmd = exec.Command("gfortran", "-o", exeFirstName, sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Lua:
		cmd = exec.Command("luac", "-o", exeFirstName+".out", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Nim:
		cmd = exec.Command("nim", "c", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.ObjectPascal:
		cmd = exec.Command("fpc", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.D:
		if e.debugMode {
			cmd = exec.Command("gdc", "-Og", "-g", "-o", exeFirstName, sourceFilename)
		} else {
			cmd = exec.Command("gdc", "-o", exeFirstName, sourceFilename)
		}
		cmd.Dir = sourceDir
		return cmd, exeExists, nil
	case mode.HTML:
		if isDarwin {
			cmd = exec.Command("open", sourceFilename)
		} else {
			cmd = exec.Command("xdg-open", sourceFilename)
		}
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Koka:
		cmd = exec.Command("koka", "--builddir", kokaBuildDir, "-o", exeFirstName, sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Odin:
		pattern := filepath.Join(sourceDir, "*.odin")
		if matches, err := filepath.Glob(pattern); err == nil && len(matches) != 1 {
			cmd = exec.Command("odin", "build", ".", "-max-error-count:1")
		} else {
			cmd = exec.Command("odin", "build", sourceFilename, "-file", "-max-error-count:1")
		}
		if e.debugMode {
			cmd.Args = append(cmd.Args, "-strict-style", "-vet-unused", "-vet-using-stmt", "-vet-using-param", "-vet-style", "-vet-semicolon", "-debug")
		}
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.CS:
		cmd = exec.Command("csc", "-nologo", "-unsafe", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.StandardML:
		cmd = exec.Command("mlton", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Agda:
		cmd = exec.Command("agda", "-c", sourceFilename)
		cmd.Dir = sourceDir
		return cmd, everythingIsFine, nil
	case mode.Assembly:
		objFullFilename := exeFilename + ".o"
		objCheckFunc := func() (bool, string) {
			// Note that returning the full path as the second argument instead of only the base name
			// is only done for mode.Assembly. It's treated differently further down when linking.
			return files.IsFile(objFullFilename), objFullFilename
		}
		// try to use yasm
		if files.WhichCached("yasm") != "" {
			cmd = exec.Command("yasm", "-f", "elf64", "-o", objFullFilename, sourceFilename)
			if e.debugMode {
				cmd.Args = append(cmd.Args, "-g", "dwarf2")
			}
			return cmd, objCheckFunc, nil
		}
		// then try to use nasm
		if files.WhichCached("nasm") != "" { // use nasm
			cmd = exec.Command("nasm", "-f", "elf64", "-o", objFullFilename, sourceFilename)
			if e.debugMode {
				cmd.Args = append(cmd.Args, "-g")
			}
			return cmd, objCheckFunc, nil
		}
		// No result
	}

	return nil, nothingIsFine, errNoSuitableBuildCommand // errors.New("No build command for " + e.mode.String() + " files")
}

// BuildOrExport will try to build the source code or export the document.
// Returns a status message and then true if an action was performed and another true if compilation/testing worked out.
// Will also return the executable output file, if available after compilation.
func (e *Editor) BuildOrExport(tty *vt100.TTY, c *vt100.Canvas, status *StatusBar) (string, error) {
	// Clear the status messages, if we have a status bar
	if status != nil && c != nil {
		status.ClearAll(c, false)
	}

	// Find the absolute path to the source file
	sourceFilename, err := filepath.Abs(e.filename)
	if err != nil {
		return "", err
	}

	// Set up a few basic variables about the given source file
	var (
		baseFilename = filepath.Base(sourceFilename)
		sourceDir    = filepath.Dir(sourceFilename)
		exeFirstName = e.exeName(sourceFilename, false)
		exeFilename  = filepath.Join(sourceDir, exeFirstName)
		ext          = filepath.Ext(sourceFilename)
	)

	// Get a few simple cases out of the way first, by filename extension
	switch e.mode {
	case mode.SCDoc: //
		const manFilename = "out.1"
		status.SetMessage("Exporting SCDoc to PDF")
		status.Show(c, e)
		if err := e.exportScdoc(manFilename); err != nil {
			return "", err
		}
		if status != nil {
			status.SetMessage("Saved " + manFilename)
		}
		return manFilename, nil
	case mode.ASCIIDoc: // asciidoctor
		const manFilename = "out.1"
		status.SetMessage("Exporting ASCIIDoc to PDF")
		status.Show(c, e)
		if err := e.exportAdoc(c, tty, manFilename); err != nil {
			return "", err
		}
		if status != nil {
			status.SetMessage("Saved " + manFilename)
		}
		return manFilename, nil
	case mode.Lilypond:
		ext := filepath.Ext(e.filename)
		firstName := strings.TrimSuffix(filepath.Base(e.filename), ext)
		outputFilename := firstName + ".pdf" // lilypond may output .midi and/or .pdf by default. --svg is also possible.
		status.SetMessage("Exporting Lilypond to PDF")
		status.Show(c, e)
		cmd := exec.Command("lilypond", "-o", firstName, e.filename)
		saveCommand(cmd)
		return outputFilename, cmd.Run()
	case mode.Markdown:
		htmlFilename := strings.ReplaceAll(filepath.Base(sourceFilename), ".", "_") + ".html"
		go func() {
			_ = e.exportMarkdownHTML(c, status, htmlFilename)
		}()
		return htmlFilename, nil
	case mode.Lua:
		if e.LuaLoveOrLovr() {
			return "", nil
		}
	}

	// The immediate builds are done, time to build a exec.Cmd, run it and analyze the output

	cmd, compilationProducedSomething, err := e.GenerateBuildCommand(c, tty, sourceFilename)
	if err != nil {
		return "", err
	}

	// Check that the resulting cmd.Path executable exists
	if files.WhichCached(cmd.Path) == "" {
		return "", fmt.Errorf("%s (%s %s)", errNoSuitableBuildCommand.Error(), "could not find", cmd.Path)
	}

	// Display a status message with no timeout, about what is currently being done
	if status != nil {
		var progressStatusMessage string
		if e.mode == mode.HTML || e.mode == mode.XML {
			progressStatusMessage = "Displaying"
		} else if !e.debugMode {
			progressStatusMessage = "Building"
		}
		status.SetMessage(progressStatusMessage)
		status.ShowNoTimeout(c, e)
	}

	// Save the command in a temporary file
	saveCommand(cmd)

	// --- Compilation ---

	// Run the command and fetch the combined output from stderr and stdout.
	// Ignore the status code / error, only look at the output.
	output, err := cmd.CombinedOutput()

	// Done building, clear the "Building" message
	if status != nil {
		status.ClearAll(c, false)
	}

	// Get the exit code and combined output of the build command
	exitCode := 0
	if exitError, ok := err.(*exec.ExitError); ok {
		exitCode = exitError.ExitCode()
	}
	outputString := string(bytes.TrimSpace(output))

	// Remove .Random.seed if a68g was just used
	if e.mode == mode.Algol68 {
		if files.IsFile(".Random.seed") {
			os.Remove(".Random.seed")
		}
	}

	// Check if there was a non-zero exit code together with no output
	if exitCode != 0 && outputString == "" {
		return "", errors.New("non-zero exit code and no error message")
	}

	// Also perform linking, if needed
	if ok, objFullFilename := compilationProducedSomething(); e.mode == mode.Assembly && ok {
		linkerCmd := exec.Command("ld", "-o", exeFilename, objFullFilename)
		linkerCmd.Dir = sourceDir
		if e.debugMode {
			linkerCmd.Args = append(linkerCmd.Args, "-g")
		}
		var linkerOutput []byte
		linkerOutput, err = linkerCmd.CombinedOutput()
		if err != nil {
			output = append(output, '\n')
			output = append(output, linkerOutput...)
		} else {
			os.Remove(objFullFilename)
		}
		// Replace the result check function
		compilationProducedSomething = func() (bool, string) {
			return files.IsFile(exeFilename), exeFirstName
		}
	}

	// Special considerations for Kotlin Native
	if usingKotlinNative := strings.HasSuffix(cmd.Path, "kotlinc-native"); usingKotlinNative && files.IsFile(exeFirstName+".kexe") {
		os.Rename(exeFirstName+".kexe", exeFirstName)
	}

	// Special considerations for Koka
	if e.mode == mode.Koka && files.IsFile(exeFirstName) {
		// chmod +x
		os.Chmod(exeFirstName, 0o755)
	}

	// NOTE: Don't do anything with the output and err variables here, let the if below handle it.

	errorMarker := "error:"
	switch e.mode {
	case mode.C3, mode.Crystal, mode.ObjectPascal, mode.StandardML, mode.Python:
		errorMarker = "Error:"
	case mode.Dart:
		errorMarker = ": Error: "
	case mode.CS:
		errorMarker = ": error "
	case mode.Agda:
		errorMarker = ","
	}

	// Check if the error marker should be changed

	if e.mode == mode.Zig && bytes.Contains(output, []byte("nrecognized glibc version")) {
		byteLines := bytes.Split(output, []byte("\n"))
		fields := strings.Split(string(byteLines[0]), ":")
		errorMessage := "Error: unrecognized glibc version"
		if len(fields) > 1 {
			errorMessage += ": " + strings.TrimSpace(fields[1])
		}
		return "", errors.New(errorMessage)
	} else if e.mode == mode.Go {
		switch {
		case bytes.Contains(output, []byte(": undefined")):
			errorMarker = "undefined"
		case bytes.Contains(output, []byte(": warning")):
			errorMarker = "error"
		case bytes.Contains(output, []byte(": note")):
			errorMarker = "error"
		case bytes.Contains(output, []byte(": error")):
			errorMarker = "error"
		case bytes.Contains(output, []byte("go: cannot find main module")):
			errorMessage := "no main module, try go mod init"
			return "", errors.New(errorMessage)
		case bytes.Contains(output, []byte("go: ")):
			byteLines := bytes.SplitN(output[4:], []byte("\n"), 2)
			return "", errors.New(string(byteLines[0]))
		case bytes.Count(output, []byte(":")) >= 2:
			errorMarker = ":"
		}
	} else if e.mode == mode.Odin {
		switch {
		case bytes.Contains(output, []byte(") ")):
			errorMarker = ") "
		}
	} else if exitCode == 0 && (e.mode == mode.HTML || e.mode == mode.XML) {
		return "", nil
	}

	// Did the command return a non-zero status code, or does the output contain "error:"?
	if err != nil || bytes.Contains(output, []byte(errorMarker)) { // failed tests also end up here

		// This is not for Go, since the word "error:" may not appear when there are errors

		errorMessage := "Build error"

		if e.mode == mode.Python {
			if errorLine, errorColumn, errorMessage := ParsePythonError(string(output), filepath.Base(e.filename)); errorLine != -1 {
				ignoreIndentation := true
				e.MoveToLineColumnNumber(c, status, errorLine, errorColumn, ignoreIndentation)
				return "", errors.New(errorMessage)
			}
			// This should never happen, the error message should be handled by ParsePythonError!
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			lastLine := lines[len(lines)-1]
			return "", errors.New(lastLine)
		} else if e.mode == mode.Agda {
			lines := strings.Split(string(output), "\n")
			if len(lines) >= 4 {
				fileAndLocation := lines[1]
				errorMessage := strings.TrimSpace(lines[2]) + " " + strings.TrimSpace(lines[3])
				if strings.Contains(fileAndLocation, ":") && strings.Contains(fileAndLocation, ",") && strings.Contains(fileAndLocation, "-") {
					fields := strings.SplitN(fileAndLocation, ":", 2)
					// filename := fields[0]
					lineAndCol := fields[1]
					fields = strings.SplitN(lineAndCol, ",", 2)
					lineNumberString := fields[0] // not index
					colRange := fields[1]
					fields = strings.SplitN(colRange, "-", 2)
					lineColumnString := fields[0] // not index

					e.MoveToNumber(c, status, lineNumberString, lineColumnString)

					return "", errors.New(errorMessage)
				}
			}
		}

		// Find the first error message
		var (
			lines               = strings.Split(string(output), "\n")
			prevLine            string
			crystalLocationLine string
		)
		for _, line := range lines {
			if e.mode == mode.Haskell {
				if strings.Contains(prevLine, errorMarker) {
					if errorMessage = strings.TrimSpace(line); strings.HasPrefix(errorMessage, "• ") {
						errorMessage = string([]rune(errorMessage)[2:])
						break
					}
				}
			} else if e.mode == mode.StandardML {
				if strings.Contains(prevLine, errorMarker) && strings.Contains(prevLine, ".") {
					errorMessage = strings.TrimSpace(line)
					fields := strings.Split(prevLine, " ")
					if len(fields) > 2 {
						location := fields[2]
						fields = strings.Split(location, "-")
						if len(fields) > 0 {
							location = fields[0]
							locCol := strings.Split(location, ".")
							if len(locCol) > 0 {
								lineNumberString := locCol[0]
								lineColumnString := locCol[1]
								// Move to (x, y), line number first and then column number
								if i, err := strconv.Atoi(lineNumberString); err == nil {
									foundY := LineIndex(i - 1)
									redraw, _ := e.GoTo(foundY, c, status)
									e.redraw.Store(redraw)
									e.redrawCursor.Store(redraw)
									if x, err := strconv.Atoi(lineColumnString); err == nil { // no error
										foundX := x - 1
										tabs := strings.Count(e.Line(foundY), "\t")
										e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
										e.Center(c)
									}
								}
								return "", errors.New(errorMessage)
							}
						}
					}
					break
				}
			} else if e.mode == mode.Crystal {
				if strings.HasPrefix(line, "Error:") {
					errorMessage = line[6:]
					if len(crystalLocationLine) > 0 {
						break
					}
				} else if strings.HasPrefix(line, "In ") {
					crystalLocationLine = line
				}
			} else if e.mode == mode.Hare {
				errorMessage = ""
				if strings.Contains(line, errorMarker) && strings.Contains(line, " at ") {
					descriptionFields := strings.SplitN(line, " at ", 2)
					errorMessage = descriptionFields[0]
					if strings.Contains(errorMessage, "error:") {
						fields := strings.SplitN(errorMessage, "error:", 2)
						errorMessage = fields[1]
					}
					filenameAndError := descriptionFields[1]
					filenameAndLoc := ""
					if strings.Contains(filenameAndError, ", ") {
						fields := strings.SplitN(filenameAndError, ", ", 2)
						filenameAndLoc = fields[0]
						errorMessage += ": " + fields[1]
					}
					fields := strings.SplitN(filenameAndLoc, ":", 3)
					errorFilename := fields[0]
					baseErrorFilename := filepath.Base(errorFilename)
					lineNumberString := fields[1]
					lineColumnString := fields[2]

					e.MoveToNumber(c, status, lineNumberString, lineColumnString)

					// Return the error message
					if baseErrorFilename != baseFilename {
						return "", errors.New("in " + baseErrorFilename + ": " + errorMessage)
					}
					return "", errors.New(errorMessage)

				} else if strings.HasPrefix(line, "Error ") {
					fields := strings.Split(line[6:], ":")
					if len(fields) >= 4 {
						errorFilename := fields[0]
						baseErrorFilename := filepath.Base(errorFilename)
						lineNumberString := fields[1]
						lineColumnString := fields[2]
						errorMessage := fields[3]

						e.MoveToNumber(c, status, lineNumberString, lineColumnString)

						// Return the error message
						if baseErrorFilename != baseFilename {
							return "", errors.New("in " + baseErrorFilename + ": " + errorMessage)
						}
						return "", errors.New(errorMessage)
					}
				}
			} else if e.mode == mode.Odin {
				errorMessage = ""
				if strings.Contains(line, errorMarker) {
					whereAndWhat := strings.SplitN(line, errorMarker, 2)
					where := whereAndWhat[0]
					errorMessage = whereAndWhat[1]
					filenameAndLoc := strings.SplitN(where, "(", 2)
					errorFilename := filenameAndLoc[0]
					baseErrorFilename := filepath.Base(errorFilename)
					loc := filenameAndLoc[1]
					locCol := strings.SplitN(loc, ":", 2)
					lineNumberString := locCol[0]
					lineColumnString := locCol[1]

					const subtractOne = true
					e.MoveToIndex(c, status, lineNumberString, lineColumnString, subtractOne)

					// Return the error message
					if baseErrorFilename != baseFilename {
						return "", errors.New("in " + baseErrorFilename + ": " + errorMessage)
					}
					return "", errors.New(errorMessage)
				}
			} else if e.mode == mode.Dart {
				errorMessage = ""
				if strings.Contains(line, errorMarker) {
					whereAndWhat := strings.SplitN(line, errorMarker, 2)
					where := whereAndWhat[0]
					errorMessage = whereAndWhat[1]
					filenameAndLoc := strings.SplitN(where, ":", 2)
					errorFilename := filenameAndLoc[0]
					baseErrorFilename := filepath.Base(errorFilename)
					loc := filenameAndLoc[1]
					locCol := strings.SplitN(loc, ":", 2)
					lineNumberString := locCol[0]
					lineColumnString := locCol[1]

					const subtractOne = true
					e.MoveToIndex(c, status, lineNumberString, lineColumnString, subtractOne)

					// Return the error message
					if baseErrorFilename != baseFilename {
						return "", errors.New("in " + baseErrorFilename + ": " + errorMessage)
					}
					return "", errors.New(errorMessage)
				}
			} else if e.mode == mode.CS || e.mode == mode.ObjectPascal {
				errorMessage = ""
				if strings.Contains(line, " Error: ") {
					pos := strings.Index(line, " Error: ")
					errorMessage = line[pos+8:]
				} else if strings.Contains(line, " Fatal: ") {
					pos := strings.Index(line, " Fatal: ")
					errorMessage = line[pos+8:]
				} else if strings.Contains(line, ": error ") {
					pos := strings.Index(line, ": error ")
					errorMessage = line[pos+8:]
				}
				if len(errorMessage) > 0 {
					parts := strings.SplitN(line, "(", 2)
					errorFilename, rest := parts[0], parts[1]
					baseErrorFilename := filepath.Base(errorFilename)
					parts = strings.SplitN(rest, ",", 2)
					lineNumberString, rest := parts[0], parts[1]
					parts = strings.SplitN(rest, ")", 2)
					lineColumnString, rest := parts[0], parts[1]
					errorMessage = rest
					if e.mode == mode.CS {
						if strings.Count(rest, ":") == 2 {
							parts := strings.SplitN(rest, ":", 3)
							errorMessage = parts[2]
						}
					}

					// Move to (x, y), line number first and then column number
					if i, err := strconv.Atoi(lineNumberString); err == nil {
						foundY := LineIndex(i - 1)
						redraw, _ := e.GoTo(foundY, c, status)
						e.redraw.Store(redraw)
						e.redrawCursor.Store(redraw)
						if x, err := strconv.Atoi(lineColumnString); err == nil { // no error
							foundX := x - 1
							tabs := strings.Count(e.Line(foundY), "\t")
							e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
							e.Center(c)
						}
					}

					// Return the error message
					if baseErrorFilename != baseFilename {
						return "", errors.New("In " + baseErrorFilename + ": " + errorMessage)
					}
					return "", errors.New(errorMessage)
				}
			} else if e.mode == mode.Lua {
				if strings.Contains(line, " error near ") && strings.Count(line, ":") >= 3 {
					parts := strings.SplitN(line, ":", 4)
					errorMessage = parts[3]

					if i, err := strconv.Atoi(parts[2]); err == nil {
						foundY := LineIndex(i - 1)
						redraw, _ := e.GoTo(foundY, c, status)
						e.redraw.Store(redraw)
						e.redrawCursor.Store(redraw)
					}

					baseErrorFilename := filepath.Base(parts[1])
					if baseErrorFilename != baseFilename {
						return "", errors.New("In " + baseErrorFilename + ": " + errorMessage)
					}
					return "", errors.New(errorMessage)
				}
				break
			} else if e.mode == mode.Go && errorMarker == ":" && strings.Count(line, ":") >= 2 {
				parts := strings.SplitN(line, ":", 2)
				errorMessage = strings.Join(parts[2:], ":")
				break
			} else if strings.Contains(line, errorMarker) {
				parts := strings.SplitN(line, errorMarker, 2)
				if errorMarker == "undefined" {
					errorMessage = errorMarker + strings.TrimSpace(parts[1])
				} else {
					errorMessage = strings.TrimSpace(parts[1])
				}
				break
			}
			prevLine = line
		}

		if e.mode == mode.Crystal {
			// Crystal has the location on a different line from the error message
			fields := strings.Split(crystalLocationLine, ":")
			if len(fields) != 3 {
				return "", errors.New(errorMessage)
			}
			if y, err := strconv.Atoi(fields[1]); err == nil { // no error

				foundY := LineIndex(y - 1)
				redraw, _ := e.GoTo(foundY, c, status)
				e.redraw.Store(redraw)
				e.redrawCursor.Store(redraw)

				if x, err := strconv.Atoi(fields[2]); err == nil { // no error
					foundX := x - 1
					tabs := strings.Count(e.Line(foundY), "\t")
					e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
					e.Center(c)
				}

			}
			return "", errors.New(errorMessage)
		}

		// NOTE: Don't return here even if errorMessage contains an error message

		// Analyze all lines
		for i, line := range lines {
			// Go, C++, Haskell, Kotlin and more
			if strings.Contains(line, "fatal error") {
				return "", errors.New(line)
			}
			if strings.Count(line, ":") >= 3 && (strings.Contains(line, "error:") || strings.Contains(line, errorMarker)) {
				fields := strings.SplitN(line, ":", 4)
				baseErrorFilename := filepath.Base(fields[0])
				// Check if the filenames are matching, or if the error is in a different file
				if baseErrorFilename != baseFilename {
					return "", errors.New("In " + baseErrorFilename + ": " + strings.TrimSpace(fields[3]))
				}
				// Go to Y:X, if available
				var foundY LineIndex
				if y, err := strconv.Atoi(fields[1]); err == nil { // no error
					foundY = LineIndex(y - 1)
					redraw, _ := e.GoTo(foundY, c, status)
					e.redraw.Store(redraw)
					e.redrawCursor.Store(redraw)
					foundX := -1
					if x, err := strconv.Atoi(fields[2]); err == nil { // no error
						foundX = x - 1
					}
					if foundX != -1 {

						tabs := strings.Count(e.Line(foundY), "\t")
						e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
						e.Center(c)

						// Use the error message as the status message
						if len(fields) >= 4 {
							if ext != ".hs" {
								return "", errors.New(strings.Join(fields[3:], " "))
							}
							return "", errors.New(errorMessage)
						}
					}
				}
				return "", errors.New(errorMessage)
			} else if (i > 0) && i < (len(lines)-1) {
				// Rust
				if msgLine := lines[i-1]; strings.Contains(line, " --> ") && strings.Count(line, ":") == 2 && strings.Count(msgLine, ":") >= 1 {
					errorFields := strings.SplitN(msgLine, ":", 2)                  // Already checked for 2 colons
					errorMessage := strings.TrimSpace(errorFields[1])               // There will always be 3 elements in errorFields, so [1] is fine
					locationFields := strings.SplitN(line, ":", 3)                  // Already checked for 2 colons in line
					filenameFields := strings.SplitN(locationFields[0], " --> ", 2) // [0] is fine, already checked for " ---> "
					errorFilename := strings.TrimSpace(filenameFields[1])           // [1] is fine
					if e.filename != errorFilename {
						return "", errors.New("In " + errorFilename + ": " + errorMessage)
					}
					errorY := locationFields[1]
					errorX := locationFields[2]

					// Go to Y:X, if available
					var foundY LineIndex
					if y, err := strconv.Atoi(errorY); err == nil { // no error
						foundY = LineIndex(y - 1)
						redraw, _ := e.GoTo(foundY, c, status)
						e.redraw.Store(redraw)
						e.redrawCursor.Store(redraw)
						foundX := -1
						if x, err := strconv.Atoi(errorX); err == nil { // no error
							foundX = x - 1
						}
						if foundX != -1 {
							tabs := strings.Count(e.Line(foundY), "\t")
							e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
							e.Center(c)
							// Use the error message as the status message
							if errorMessage != "" {
								return "", errors.New(errorMessage)
							}
						}
					}
					e.redrawCursor.Store(true)
					// Nope, just the error message
					// return errorMessage, true, false
				}
			}
		}
	}

	// Do not expect successful compilation to have produced an artifact
	if e.mode == mode.Python {
		if status != nil {
			status.SetMessage("Syntax OK")
			status.Show(c, e)
		}
		return "", nil
	}

	// Could not interpret the error message, return the last line of the output
	if exitCode != 0 && len(outputString) > 0 {
		outputLines := strings.Split(outputString, "\n")
		lastLine := outputLines[len(outputLines)-1]
		return "", errors.New(lastLine)
	}

	if ok, what := compilationProducedSomething(); ok {
		// Returns the built executable, or exported file
		return what, nil
	}

	// TODO: Find ways to make the error message more informative
	return "", errors.New("could not compile")
}

// Build starts a build and is typically triggered from either ctrl-space or the o menu
func (e *Editor) Build(c *vt100.Canvas, status *StatusBar, tty *vt100.TTY) {
	// If the file is empty, there is nothing to build
	if e.Empty() {
		status.ClearAll(c, false)
		status.SetErrorMessage("Nothing to build, the file is empty")
		status.Show(c, e)
		return
	}

	// Save the current file, but only if it has changed
	if e.changed.Load() {
		if err := e.Save(c, tty); err != nil {
			status.ClearAll(c, false)
			status.SetError(err)
			status.Show(c, e)
			return
		}
	}

	// debug stepping
	if e.debugMode && e.gdb != nil {
		if !programRunning {
			e.DebugEnd()
			status.SetMessage("Program stopped")
			e.redrawCursor.Store(true)
			e.redraw.Store(true)
			status.SetMessageAfterRedraw(status.Message())
			return
		}
		status.ClearAll(c, false)
		// If we have a breakpoint, continue to it
		if e.breakpoint != nil { // exists
			// continue forward to the end or to the next breakpoint
			if err := e.DebugContinue(); err != nil {
				// logf("[continue] gdb output: %s\n", gdbOutput)
				e.DebugEnd()
				status.SetMessage("Done")
				e.GoToEnd(nil, nil)
			} else {
				status.SetMessage("Continue")
			}
		} else { // if not, make one step
			err := e.DebugStep()
			if err != nil {
				if errorMessage := err.Error(); strings.Contains(errorMessage, "is not being run") {
					e.DebugEnd()
					status.SetMessage("Done stepping")
				} else if err == errProgramStopped {
					e.DebugEnd()
					status.SetMessage("Program stopped")
				} else {
					e.DebugEnd()
					status.SetMessage(errorMessage)
				}
				// Go to the end, no status message
				e.GoToEnd(c, nil)
			} else {
				status.SetMessage("Step")
			}
		}
		e.redrawCursor.Store(true)

		// Redraw and use the triggered status message instead of Show
		status.SetMessageAfterRedraw(status.Message())

		return
	}

	// Clear the current search term, but don't redraw if there are status messages
	e.ClearSearch()
	e.redraw.Store(false)

	// Require a double ctrl-space when exporting Markdown to HTML, because it is so easy to press by accident
	if e.mode == mode.Markdown && !e.runAfterBuild {
		return
	}

	// Run after building, for some modes
	if e.building && !e.runAfterBuild {
		if e.CanRun() {
			status.ClearAll(c, false)
			const repositionCursorAfterDrawing = true
			const rightHandSide = true
			e.DrawOutput(c, 20, "", "Building and running...", e.DebugRegistersBackground, repositionCursorAfterDrawing, rightHandSide)
			e.runAfterBuild = true
		}
		return
	}
	if e.building && e.runAfterBuild {
		// do nothing when ctrl-space is pressed more than 2 times when building
		return
	}

	// Not building anything right now
	go func() {
		e.building = true
		defer func() {
			e.building = false
			if e.runAfterBuild {
				e.runAfterBuild = false

				doneRunning := false
				go func() {
					// TODO: Wait for the process to start instead of sleeping
					time.Sleep(500 * time.Millisecond)
					if !doneRunning {
						const repositionCursorAfterDrawing = true
						const rightHandSide = true
						if skipTheStatusBox := e.mode == mode.Lua && e.LuaLoveOrLovr(); !skipTheStatusBox {
							msg := "Done building. Running..."
							switch e.mode {
							case mode.ABC:
								msg = "Playing with Timidity..."
							case mode.JavaScript, mode.Lua, mode.Python, mode.Shell, mode.TypeScript:
								msg = "Running..."
							}
							e.DrawOutput(c, 20, "", "  "+msg, e.DebugStoppedBackground, repositionCursorAfterDrawing, rightHandSide)
						}
					}
				}()

				output, useErrorStyle, err := e.Run()
				doneRunning = true
				if err != nil {
					status.SetError(err)
					status.Show(c, e)
					return // from goroutine
				}

				// Clear the "Done building. Running..." box
				const drawLines = true
				const shouldHighlightCurrentLine = false
				e.FullResetRedraw(c, status, drawLines, shouldHighlightCurrentLine)

				title := "Program output"
				n := 25
				h := float64(c.Height())
				counter := 0
				for float64(n) > h*0.6 {
					n /= 2
					counter++
					if counter > 10 { // endless loop safeguard
						break
					}
				}
				if strings.Count(output, "\n") >= n {
					title = fmt.Sprintf("Last %d lines of output", n)
				}
				boxBackgroundColor := e.DebugRunningBackground
				if useErrorStyle {
					boxBackgroundColor = e.DebugStoppedBackground
					status.SetErrorMessage("Exited with error code != 0")
				} else {
					status.SetMessage("Success")
				}
				if strings.TrimSpace(output) != "" {
					const rightHandSide = false
					const repositionCursorAfterDrawing = true
					e.DrawOutput(c, n, title, output, boxBackgroundColor, repositionCursorAfterDrawing, rightHandSide)
				} else {
					e.FullResetRedraw(c, status, true, false)
				}
				// Regular success, no debug mode
				status.Show(c, e)
			}
		}()

		// Build or export the current file
		// The last argument is if the command should run in the background or not
		outputExecutable, err := e.BuildOrExport(tty, c, status)
		// All clear when it comes to status messages and redrawing
		status.ClearAll(c, false)
		if err != nil {
			// There was an error, so don't run after building after all
			e.runAfterBuild = false
			// Error while building
			status.SetError(err)
			status.ShowNoTimeout(c, e)
			e.redrawCursor.Store(true)
			return // return from goroutine
		}
		// Not building any more
		e.building = false

		// --- success ---

		// ctrl-space was pressed while in debug mode, and without a debug session running
		if e.debugMode && e.gdb == nil {
			if err := e.DebugStartSession(c, tty, status, outputExecutable); err != nil {
				status.ClearAll(c, true)
				status.SetError(err)
				status.ShowNoTimeout(c, e)
				e.redrawCursor.Store(true)
			}
			return // return from goroutine
		}

		status.SetMessage("Success")
		status.Show(c, e)

	}()
}

// OnlyBuild tries to build/export the given FilenameOrData, without starting a full editor
func OnlyBuild(fnord FilenameOrData) (string, error) {
	// Prepare an editor, without tty and canvas
	e, _, _, err := NewEditor(nil, nil, fnord, 0, 0, NewDefaultTheme(), false, true, false, false, false, false)
	if err != nil {
		return "", err
	}
	// Prepare a status bar
	status := e.NewStatusBar(2700*time.Millisecond, "")

	// Try to change directory to where the file is at
	directory, err := os.Getwd()
	if err != nil {
		directory = "."
	}
	if !fnord.stdin {
		directory = filepath.Dir(fnord.filename)
		_ = os.Chdir(directory)
	}

	// Build, without tty and canvas
	if _, err := e.BuildOrExport(nil, nil, status); err != nil {
		return "", err
	}
	// Return a message
	finalMessage := ""
	if directory != "." && directory != "" {
		finalMessage += fmt.Sprintf("cd %s\n", directory) //filepath.Join(filepath.Dir(e.filename), outputExecutable)
	}
	if lastCommand, err := readLastCommand(); err == nil { // success
		finalMessage += lastCommand
	}
	if msg := strings.TrimSpace(status.msg); status.isError && msg != "" {
		return "", errors.New(msg)
	} else if msg != "" {
		finalMessage += "\n" + msg
	}
	return finalMessage, nil
}
