package orchideous

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xyproto/files"
)

// doGenerate generates a CMakeLists.txt file.
func doGenerate(opts BuildOptions) error {
	proj := detectProject()
	if proj.MainSource == "" {
		return fmt.Errorf("no main source file found")
	}

	if fileExists("CMakeLists.txt") {
		return fmt.Errorf("not overwriting existing CMakeLists.txt")
	}

	flags := assembleFlags(proj, opts)
	exe := executableName()

	f, err := os.Create("CMakeLists.txt")
	if err != nil {
		return err
	}
	defer f.Close()

	date := time.Now().Format("2006-01-02")
	srcs := append([]string{proj.MainSource}, proj.DepSources...)
	incPaths := flags.IncPaths

	fmt.Fprintf(f, "# Generated using oh from https://github.com/xyproto/orchideous, %s\n", date)
	fmt.Fprintln(f, "cmake_minimum_required(VERSION 3.12)")
	fmt.Fprintf(f, "project(%s)\n", exe)
	fmt.Fprintf(f, "set(SOURCES %s)\n", strings.Join(srcs, " "))
	fmt.Fprintf(f, "add_executable(%s ${SOURCES})\n", exe)
	// set_property requires the target to exist, so these come after add_executable
	if !proj.IsC {
		fmt.Fprintf(f, "set_property(TARGET %s PROPERTY CXX_STANDARD 23)\n", exe)
	}
	fmt.Fprintf(f, "set_property(TARGET %s PROPERTY C_STANDARD 18)\n", exe)

	// Libraries
	libs := extractLibs(flags.LDFlags)
	if len(libs) > 0 {
		fmt.Fprintf(f, "target_link_libraries(%s %s)\n", exe, strings.Join(libs, " "))
	}

	// Include dirs
	sortedIncs := make([]string, len(incPaths))
	copy(sortedIncs, incPaths)
	sort.Strings(sortedIncs)
	if len(sortedIncs) > 0 {
		fmt.Fprintf(f, "target_include_directories(%s PRIVATE %s)\n", exe, strings.Join(sortedIncs, " "))
	}

	// Compiler
	if !proj.IsC {
		fmt.Fprintf(f, "set(CMAKE_CXX_COMPILER %s)\n", flags.Compiler)
	} else {
		fmt.Fprintf(f, "set(CMAKE_C_COMPILER %s)\n", flags.Compiler)
	}

	// CXX flags
	cxxflags := filterNonLinkFlags(flags.CFlags)
	if len(cxxflags) > 0 {
		// Remove -Wfatal-errors for IDE usage
		var filtered []string
		for _, f := range cxxflags {
			if f != "-Wfatal-errors" {
				filtered = append(filtered, f)
			}
		}
		fmt.Fprintf(f, "set(CMAKE_CXX_FLAGS \"%s\")\n", strings.Join(filtered, " "))
	}

	// Link flags
	linkFlags := extractLinkFlags(flags.LDFlags)
	if len(linkFlags) > 0 {
		fmt.Fprintf(f, "set_property(TARGET %s PROPERTY LINK_FLAGS %s)\n", exe, strings.Join(linkFlags, " "))
		fmt.Fprintf(f, "target_link_libraries(%s %s)\n", exe, strings.Join(linkFlags, " "))
	}

	// Defines
	if len(flags.Defines) > 0 {
		var defs []string
		for _, d := range flags.Defines {
			d = strings.Replace(d, "'\"", "${CMAKE_CURRENT_SOURCE_DIR}/", 1)
			d = strings.Replace(d, "\"'", "\"", 1)
			defs = append(defs, d)
		}
		fmt.Fprintf(f, "add_definitions(%s)\n", strings.Join(defs, " "))
	}

	fmt.Println("Generated CMakeLists.txt")
	return nil
}

// doPro generates a QtCreator .pro project file.
func doPro(opts BuildOptions) error {
	proj := detectProject()
	if proj.MainSource == "" {
		return fmt.Errorf("no main source file found")
	}

	flags := assembleFlags(proj, opts)
	exe := executableName()
	proFile := exe + ".pro"

	if fileExists(proFile) {
		fmt.Println("overwriting", proFile)
	}

	f, err := os.Create(proFile)
	if err != nil {
		return err
	}
	defer f.Close()

	srcs := append([]string{proj.MainSource}, proj.DepSources...)

	fmt.Fprintf(f, "TEMPLATE = app\n\n")
	fmt.Fprintln(f, "CONFIG += c++23")
	fmt.Fprintln(f, "CONFIG -= console")
	fmt.Fprintln(f, "CONFIG -= app_bundle")
	fmt.Fprintf(f, "CONFIG -= qt\n\n")
	fmt.Fprintf(f, "SOURCES += %s\n\n", strings.Join(srcs, " \\\n           "))

	libs := extractLibs(flags.LDFlags)
	if len(libs) > 0 {
		fmt.Fprintf(f, "LIBS += %s\n\n", strings.Join(libs, " \\\n        "))
	}

	sortedIncs := make([]string, len(flags.IncPaths))
	copy(sortedIncs, flags.IncPaths)
	sort.Strings(sortedIncs)
	if len(sortedIncs) > 0 {
		fmt.Fprintf(f, "INCLUDEPATH += %s\n\n", strings.Join(sortedIncs, " \\\n               "))
	}

	fmt.Fprintf(f, "QMAKE_CXX = %s\n", flags.Compiler)

	cxxflags := filterNonLinkFlags(flags.CFlags)
	var filtered []string
	for _, fl := range cxxflags {
		if fl != "-Wfatal-errors" {
			filtered = append(filtered, fl)
		}
	}
	if len(filtered) > 0 {
		fmt.Fprintf(f, "QMAKE_CXXFLAGS += %s\n", strings.Join(filtered, " "))
	}

	linkFlags := extractLinkFlags(flags.LDFlags)
	if len(linkFlags) > 0 {
		fmt.Fprintf(f, "QMAKE_LFLAGS += %s\n\n", strings.Join(linkFlags, " "))
	}

	if len(flags.Defines) > 0 {
		var s strings.Builder
		s.WriteString("DEFINES += ")
		for _, d := range flags.Defines {
			if strings.Contains(d, "=") {
				parts := strings.SplitN(d, "=", 2)
				key := strings.TrimPrefix(parts[0], "-D")
				value := parts[1]
				value = strings.Replace(value, "'\"", "'\\\"$$_PRO_FILE_PWD_/", 1)
				s.WriteString(key + "=\"" + value + "\" ")
			} else {
				s.WriteString(strings.TrimPrefix(d, "-D") + " ")
			}
		}
		fmt.Fprintln(f, strings.TrimSpace(s.String()))
	}

	fmt.Println("Generated", proFile)
	return nil
}

// doNinja builds using ninja. If build/build.ninja exists, runs ninja directly.
// Otherwise falls back to cmake+ninja if CMakeLists.txt exists.
func doNinja() error {
	// Try running ninja directly if a previous cmake+ninja build exists
	if fileExists(filepath.Join("build", "build.ninja")) {
		if _, err := exec.LookPath("ninja"); err != nil {
			return fmt.Errorf("ninja not found in PATH\n  hint: %s", installHint("ninja"))
		}
		ninja := exec.Command("ninja", "-C", "build")
		ninja.Stdout = os.Stdout
		ninja.Stderr = os.Stderr
		return ninja.Run()
	}
	// Fall back to cmake+ninja if CMakeLists.txt exists
	if fileExists("CMakeLists.txt") {
		return doCMakeNinja()
	}
	return fmt.Errorf("no build/build.ninja or CMakeLists.txt found\n  hint: run 'oh generate' to create a CMakeLists.txt")
}

// needsCMakeRegen checks whether cmake needs to be re-run in the build directory.
// Returns true if build/ doesn't exist or CMakeLists.txt is newer than the cmake cache.
func needsCMakeRegen() bool {
	cacheFile := filepath.Join("build", "CMakeCache.txt")
	cacheInfo, err := os.Stat(cacheFile)
	if err != nil {
		return true // no cache, needs generation
	}
	cmakeInfo, err := os.Stat("CMakeLists.txt")
	if err != nil {
		return true
	}
	return cmakeInfo.ModTime().After(cacheInfo.ModTime())
}

// doCMakeNinja builds the project using CMake + Ninja.
// Only re-runs cmake if CMakeLists.txt has changed; ninja handles incremental builds.
func doCMakeNinja() error {
	if !fileExists("CMakeLists.txt") {
		return fmt.Errorf("could not find CMakeLists.txt\n  hint: run 'oh generate' to create one")
	}

	if _, err := exec.LookPath("cmake"); err != nil {
		return fmt.Errorf("cmake not found in PATH\n  hint: %s", installHint("cmake"))
	}

	if _, err := exec.LookPath("ninja"); err != nil {
		return fmt.Errorf("ninja not found in PATH\n  hint: %s", installHint("ninja"))
	}

	if needsCMakeRegen() {
		if err := os.MkdirAll("build", 0o755); err != nil {
			return err
		}

		cmakeArgs := []string{"-G", "Ninja", ".."}
		if files.WhichCached("ccache") != "" {
			cmakeArgs = []string{"-D", "CMAKE_CXX_COMPILER_LAUNCHER=ccache", "-G", "Ninja", ".."}
		}
		cmake := exec.Command("cmake", cmakeArgs...)
		cmake.Dir = "build"
		cmake.Stdout = os.Stdout
		cmake.Stderr = os.Stderr
		if err := cmake.Run(); err != nil {
			return fmt.Errorf("cmake failed: %w", err)
		}
	}

	// ninja handles parallelism and incremental builds automatically
	ninja := exec.Command("ninja", "-C", "build")
	ninja.Stdout = os.Stdout
	ninja.Stderr = os.Stderr
	if err := ninja.Run(); err != nil {
		return fmt.Errorf("ninja failed: %w", err)
	}

	return nil
}

// doMake builds using make. If a Makefile exists, runs make directly.
// Otherwise falls back to cmake+make if CMakeLists.txt exists.
func doMake() error {
	// Try running make directly if a Makefile exists
	if fileExists("Makefile") {
		if _, err := exec.LookPath("make"); err != nil {
			return fmt.Errorf("make not found in PATH\n  hint: %s", installHint("make"))
		}
		jobs := strconv.Itoa(runtime.NumCPU())
		cmd := exec.Command("make", "-j"+jobs)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	// Fall back to cmake+make if CMakeLists.txt exists
	if fileExists("CMakeLists.txt") {
		return doCMakeMake()
	}
	return fmt.Errorf("no Makefile or CMakeLists.txt found\n  hint: run 'oh generate' to create a CMakeLists.txt, or 'oh export' to create a Makefile")
}

// doCMakeBuild builds using cmake, preferring ninja over make.
// Generates CMakeLists.txt first if it does not exist.
func doCMakeBuild(opts BuildOptions) error {
	if !fileExists("CMakeLists.txt") {
		if err := doGenerate(opts); err != nil {
			return err
		}
	}
	// Prefer ninja if available, otherwise fall back to make
	if _, err := exec.LookPath("ninja"); err == nil {
		return doCMakeNinja()
	}
	return doCMakeMake()
}

// doInstall installs the built executable and data directories.
func doInstall() error {
	prefix := os.Getenv("PREFIX")
	if prefix == "" {
		prefix = "/usr/local"
	}
	destdir := os.Getenv("DESTDIR")
	exe := executableName()

	// Build with install-time directory defines
	proj := detectProject()
	opts := BuildOptions{}
	if proj.HasWin64 {
		opts.Win64 = true
		exe += ".exe"
	}

	// Override directory defines to point to installed paths
	installOpts := opts
	installOpts.InstallPrefix = filepath.Join(prefix, "share", exe)

	if err := doBuildWithDirOverrides(installOpts, proj); err != nil {
		// Fallback to normal build
		if err := doBuild(opts); err != nil {
			return err
		}
	}

	binDir := filepath.Join(destdir, prefix, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}

	// Install executable
	src := exe
	if !fileExists(src) {
		src = filepath.Join("src", "main")
	}
	dst := filepath.Join(binDir, exe)
	if err := copyFile(src, dst, 0o755); err != nil {
		return fmt.Errorf("installing executable: %w", err)
	}
	fmt.Printf("Installed %s -> %s\n", src, dst)

	shareDir := filepath.Join(destdir, prefix, "share", exe)

	// Install data directories
	dataDirs := map[string]string{
		"img": "img", "imgs": "img",
		"data": "data", "datas": "data",
		"shaders": "shaders", "shader": "shaders",
		"resources": "resources", "resource": "resources",
		"res": "res", "scripts": "scripts",
		"share": "", "shared": "",
	}

	for srcDir, dstName := range dataDirs {
		if !fileExists(srcDir) {
			continue
		}
		var targetDir string
		if dstName == "" {
			targetDir = shareDir
		} else {
			targetDir = filepath.Join(shareDir, dstName)
		}
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return err
		}
		c := exec.Command("cp", "-r", srcDir+"/.", targetDir+"/")
		if err := c.Run(); err != nil {
			return fmt.Errorf("copying %s: %w", srcDir, err)
		}
		fmt.Printf("Installed %s/ -> %s/\n", srcDir, targetDir)
	}

	// Install license files
	licDir := filepath.Join(destdir, prefix, "share", "licenses", exe)
	for _, lic := range []string{"COPYING", "LICENSE"} {
		if fileExists(lic) {
			if err := os.MkdirAll(licDir, 0o755); err != nil {
				return err
			}
			if err := copyFile(lic, filepath.Join(licDir, lic), 0o644); err != nil {
				return fmt.Errorf("installing %s: %w", lic, err)
			}
			fmt.Printf("Installed %s -> %s\n", lic, filepath.Join(licDir, lic))
		}
	}

	// For win64 builds, create a wine wrapper script
	if opts.Win64 {
		wrapperPath := filepath.Join(binDir, strings.TrimSuffix(exe, ".exe"))
		wrapper := fmt.Sprintf("#!/bin/sh\nwine %s \"$@\"\n", filepath.Join(prefix, "bin", exe))
		if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o755); err == nil {
			fmt.Printf("Installed wine wrapper -> %s\n", wrapperPath)
		}
	}

	return nil
}

// doPkg packages the project to a pkg/ directory.
func doPkg() error {
	pkgDir := os.Getenv("pkgdir")
	if pkgDir == "" {
		pkgDir = filepath.Join(".", "pkg")
	}
	os.Setenv("DESTDIR", pkgDir)
	return doInstall()
}

// doExport generates a standalone Makefile and build.sh for users without oh.
func doExport() error {
	// Generate Makefile
	if err := doGenerateMakefile(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not generate Makefile: %v\n", err)
	}

	// Generate build/clean scripts
	if err := doScript(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not generate scripts: %v\n", err)
	}

	return nil
}

// doGenerateMakefile generates a standalone Makefile.
func doGenerateMakefile() error {
	if fileExists("Makefile") {
		return fmt.Errorf("makefile already exists, will not overwrite")
	}

	proj := detectProject()
	if proj.MainSource == "" {
		return fmt.Errorf("no main source file found")
	}

	flags := assembleFlags(proj, BuildOptions{})
	exe := executableName()
	srcs := append([]string{proj.MainSource}, proj.DepSources...)
	objs := make([]string, len(srcs))
	for i, s := range srcs {
		objs[i] = strings.TrimSuffix(s, filepath.Ext(s)) + ".o"
	}

	f, err := os.Create("Makefile")
	if err != nil {
		return err
	}
	defer f.Close()

	var compileCmd strings.Builder
	compileCmd.WriteString(flags.Compiler + " -std=" + flags.Std + " " + strings.Join(flags.CFlags, " "))
	if len(flags.Defines) > 0 {
		compileCmd.WriteString(" " + strings.Join(flags.Defines, " "))
	}
	for _, ip := range flags.IncPaths {
		compileCmd.WriteString(" -I" + ip)
	}

	linkCmd := flags.Compiler + " -o " + exe + " " + strings.Join(objs, " ")
	if len(flags.LDFlags) > 0 {
		linkCmd += " " + strings.Join(flags.LDFlags, " ")
	}

	fmt.Fprintf(f, ".PHONY: clean\n\n")
	fmt.Fprintf(f, "%s: %s\n", exe, strings.Join(objs, " "))
	fmt.Fprintf(f, "\t%s\n\n", linkCmd)

	for i, src := range srcs {
		fmt.Fprintf(f, "%s: %s\n", objs[i], src)
		fmt.Fprintf(f, "\t%s -c -o %s %s\n\n", compileCmd.String(), objs[i], src)
	}

	fmt.Fprintln(f, "clean:")
	fmt.Fprintf(f, "\trm -f %s *.o common/*.o include/*.o\n", exe)

	fmt.Println("Generated Makefile")
	return nil
}

// doScript generates standalone build.sh and clean.sh scripts.
func doScript() error {
	if fileExists("build.sh") {
		return fmt.Errorf("build.sh already exists, will not overwrite")
	}
	if fileExists("clean.sh") {
		return fmt.Errorf("clean.sh already exists, will not overwrite")
	}

	proj := detectProject()
	if proj.MainSource == "" {
		return fmt.Errorf("no main source file found")
	}

	flags := assembleFlags(proj, BuildOptions{})
	exe := executableName()
	srcs := append([]string{proj.MainSource}, proj.DepSources...)

	// build.sh
	bf, err := os.Create("build.sh")
	if err != nil {
		return err
	}
	fmt.Fprintln(bf, "#!/bin/sh")
	fmt.Fprintln(bf, `printf "Building... "`)

	// If multiple sources, compile each to .o then link
	if len(srcs) > 1 {
		var compileCmd strings.Builder
		compileCmd.WriteString(flags.Compiler + " -std=" + flags.Std + " " + strings.Join(flags.CFlags, " "))
		if len(flags.Defines) > 0 {
			compileCmd.WriteString(" " + strings.Join(flags.Defines, " "))
		}
		for _, ip := range flags.IncPaths {
			compileCmd.WriteString(" -I" + ip)
		}

		var objs []string
		for _, src := range srcs {
			obj := strings.TrimSuffix(src, filepath.Ext(src)) + ".o"
			objs = append(objs, obj)
			fmt.Fprintf(bf, "%s -c -o %s %s || exit 1\n", compileCmd.String(), obj, src)
		}
		linkCmd := flags.Compiler + " -o " + exe + " " + strings.Join(objs, " ")
		if len(flags.LDFlags) > 0 {
			linkCmd += " " + strings.Join(flags.LDFlags, " ")
		}
		fmt.Fprintf(bf, "%s || exit 1\n", linkCmd)
	} else {
		args := buildCompileArgs(flags, srcs, exe)
		fmt.Fprintf(bf, "%s %s || exit 1\n", flags.Compiler, strings.Join(args, " "))
	}

	fmt.Fprintln(bf, `test $? -eq 0 && echo OK`)
	bf.Close()
	os.Chmod("build.sh", 0o755)
	fmt.Println("Generated build.sh")

	// clean.sh
	cf, err := os.Create("clean.sh")
	if err != nil {
		return err
	}
	fmt.Fprintln(cf, "#!/bin/sh")
	fmt.Fprintln(cf, `printf "Cleaning... "`)
	fmt.Fprintf(cf, "rm -f %s *.o common/*.o include/*.o\n", exe)
	fmt.Fprintln(cf, `test $? -eq 0 && echo OK`)
	cf.Close()
	os.Chmod("clean.sh", 0o755)
	fmt.Println("Generated clean.sh")

	return nil
}

// extractLibs extracts -l flags from ldflags.
func extractLibs(ldflags []string) []string {
	var libs []string
	for _, f := range ldflags {
		if strings.HasPrefix(f, "-l") {
			libs = append(libs, f)
		}
	}
	return libs
}

// extractLinkFlags returns non-library link flags (like -Wl,...).
func extractLinkFlags(ldflags []string) []string {
	var flags []string
	for _, f := range ldflags {
		if !strings.HasPrefix(f, "-l") && !strings.HasPrefix(f, "-L") {
			flags = append(flags, f)
		}
	}
	return flags
}

// filterNonLinkFlags returns only compile flags (not link flags).
func filterNonLinkFlags(flags []string) []string {
	var result []string
	for _, f := range flags {
		if !strings.HasPrefix(f, "-l") && !strings.HasPrefix(f, "-L") && !strings.HasPrefix(f, "-Wl,") {
			result = append(result, f)
		}
	}
	return result
}

// doCMakeMake builds the project using CMake + Make.
// Only re-runs cmake if CMakeLists.txt has changed; make handles incremental builds.
func doCMakeMake() error {
	if !fileExists("CMakeLists.txt") {
		return fmt.Errorf("could not find CMakeLists.txt\n  hint: run 'oh generate' to create one")
	}

	if _, err := exec.LookPath("cmake"); err != nil {
		return fmt.Errorf("cmake not found in PATH\n  hint: %s", installHint("cmake"))
	}

	if _, err := exec.LookPath("make"); err != nil {
		return fmt.Errorf("make not found in PATH\n  hint: %s", installHint("make"))
	}

	if needsCMakeRegen() {
		if err := os.MkdirAll("build", 0o755); err != nil {
			return err
		}

		cmakeArgs := []string{".."}
		if files.WhichCached("ccache") != "" {
			cmakeArgs = []string{"-D", "CMAKE_CXX_COMPILER_LAUNCHER=ccache", ".."}
		}
		cmake := exec.Command("cmake", cmakeArgs...)
		cmake.Dir = "build"
		cmake.Stdout = os.Stdout
		cmake.Stderr = os.Stderr
		if err := cmake.Run(); err != nil {
			return fmt.Errorf("cmake failed: %w", err)
		}
	}

	// Run make in build/ with parallel jobs
	jobs := strconv.Itoa(runtime.NumCPU())
	makeCmd := exec.Command("make", "-C", "build", "-j"+jobs)
	makeCmd.Stdout = os.Stdout
	makeCmd.Stderr = os.Stderr
	if err := makeCmd.Run(); err != nil {
		return fmt.Errorf("make failed: %w", err)
	}

	return nil
}

// doCMakeMakeInstall installs from a cmake+make build.
func doCMakeMakeInstall() error {
	if !fileExists("build") {
		return fmt.Errorf("no build/ directory found (run 'oh cmake' or 'oh make' first)")
	}
	if _, err := exec.LookPath("make"); err != nil {
		return fmt.Errorf("make not found in PATH")
	}
	makeCmd := exec.Command("make", "install", "-C", "build")
	makeCmd.Stdout = os.Stdout
	makeCmd.Stderr = os.Stderr
	return makeCmd.Run()
}

// doCMakeMakeClean cleans a cmake+make build.
func doCMakeMakeClean() {
	if fileExists("build") {
		os.RemoveAll("build")
		fmt.Println("Removed build/")
	}
}

// doNinjaInstall installs from a ninja build.
func doNinjaInstall() error {
	if !fileExists("build") {
		return fmt.Errorf("no build/ directory found (run 'oh ninja' first)")
	}
	if _, err := exec.LookPath("ninja"); err != nil {
		return fmt.Errorf("ninja not found in PATH")
	}
	args := []string{"install", "-C", "build"}
	ninja := exec.Command("ninja", args...)
	ninja.Stdout = os.Stdout
	ninja.Stderr = os.Stderr
	return ninja.Run()
}

// doNinjaClean cleans a ninja build.
func doNinjaClean() {
	if fileExists("build") {
		os.RemoveAll("build")
		fmt.Println("Removed build/")
	}
}

// doCleanAll performs comprehensive cleaning: make clean, ninja clean,
// removes the build/ directory, and cleans regular build artifacts.
func doCleanAll() {
	// Try make clean if Makefile exists
	if fileExists("Makefile") {
		if _, err := exec.LookPath("make"); err == nil {
			cmd := exec.Command("make", "clean")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			_ = cmd.Run()
		}
	}
	// Try ninja clean if build/build.ninja exists
	if fileExists(filepath.Join("build", "build.ninja")) {
		if _, err := exec.LookPath("ninja"); err == nil {
			cmd := exec.Command("ninja", "-C", "build", "-t", "clean")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			_ = cmd.Run()
		}
	}
	// Remove build directory
	if fileExists("build") {
		os.RemoveAll("build")
		fmt.Println("Removed build/")
	}
	// Clean regular build artifacts
	cleanFiles()
}

// copyFile copies a file with the given permissions.
func copyFile(src, dst string, perm os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, perm)
}
