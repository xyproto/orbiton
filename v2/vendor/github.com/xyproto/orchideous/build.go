package orchideous

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BuildOptions holds the configuration for a build.
type BuildOptions struct {
	InstallPrefix   string // If set, override dir defines with install paths
	Debug           bool
	Opt             bool
	Strict          bool
	Sloppy          bool
	Small           bool
	Tiny            bool
	Clang           bool
	Zap             bool
	Win64           bool
	NoSanitizers    bool
	ProfileGenerate bool
	ProfileUse      bool
}

// BuildFlags holds the assembled compiler and linker flags.
type BuildFlags struct {
	Compiler    string
	Std         string
	CFlags      []string
	LDFlags     []string
	Defines     []string
	IncPaths    []string
	DockerImage string // if set, compile via "docker run" with this image
}

// assembleFlags creates the full set of build flags for a project.
func assembleFlags(proj Project, opts BuildOptions) BuildFlags {
	// Determine if this is win64 (from options or detected from source)
	win64 := opts.Win64 || proj.HasWin64

	var bf BuildFlags

	// Find the compiler
	var compiler string
	if opts.Zap {
		if p, err := exec.LookPath("zapcc++"); err == nil {
			compiler = p
		}
	}
	if compiler == "" && win64 {
		compiler = findWin64Compiler(proj.IsC)
		if compiler == "" {
			// Fallback: use Docker with mingw image
			if _, err := exec.LookPath("docker"); err == nil {
				const dockerImage = "jhasse/mingw:latest"
				fmt.Fprintf(os.Stderr, "warning: x86_64-w64-mingw32-g++ not found, using Docker image: %s\n", dockerImage)
				if proj.IsC {
					compiler = "x86_64-w64-mingw32-gcc"
				} else {
					compiler = "x86_64-w64-mingw32-g++"
				}
				bf.DockerImage = dockerImage
			} else {
				fmt.Fprintln(os.Stderr, "error: no mingw cross-compiler found for win64 and docker is not available")
				fmt.Fprintln(os.Stderr, "Install x86_64-w64-mingw32-g++ or docker to cross-compile for Windows.")
				os.Exit(1)
			}
		}
	}
	if compiler == "" {
		compiler = findCompiler(opts.Clang, proj.IsC)
	}
	if compiler == "" {
		fmt.Fprintln(os.Stderr, "error: no C/C++ compiler found")
		os.Exit(1)
	}

	bf.Compiler = compiler

	// Determine standard
	if proj.IsC {
		if win64 {
			bf.Std = "c11"
		} else {
			bf.Std = "c18"
		}
	} else if opts.Zap {
		bf.Std = "c++14"
	} else {
		bf.Std = bestStdFlag(compiler)
	}

	// Optimization flags
	if opts.Debug {
		bf.CFlags = append(bf.CFlags, "-O0", "-g", "-fno-omit-frame-pointer")
		// Add sanitizers unless disabled
		if !opts.NoSanitizers {
			bf.CFlags = append(bf.CFlags, "-fsanitize=address")
			bf.LDFlags = append(bf.LDFlags, "-fsanitize=address")
			if !isDarwin() {
				if isCompilerClang(compiler) {
					bf.CFlags = append(bf.CFlags, "-static-libsan")
				} else if isCompilerGCC(compiler) {
					bf.CFlags = append(bf.CFlags, "-static-libasan")
				}
			}
		}
	} else if opts.Small {
		bf.CFlags = append(bf.CFlags, "-Os", "-ffunction-sections", "-fdata-sections")
		bf.LDFlags = append(bf.LDFlags, "-ffunction-sections", "-fdata-sections", "-Wl,-s", "-Wl,-gc-sections")
		if opts.Tiny {
			bf.CFlags = append(bf.CFlags, "-s", "-nostdlib", "-fno-rtti", "-fno-ident", "-fomit-frame-pointer")
			bf.LDFlags = append(bf.LDFlags, "-Wl,-z,norelro")
		}
	} else if opts.Opt {
		bf.CFlags = append(bf.CFlags, "-Ofast", "-flto")
		bf.LDFlags = append(bf.LDFlags, "-flto")
	} else if proj.HasOpenMP {
		bf.CFlags = append(bf.CFlags, "-O3")
	} else {
		bf.CFlags = append(bf.CFlags, "-O2")
	}

	// Common flags
	bf.CFlags = append(bf.CFlags, "-pipe")
	if !opts.Small {
		bf.CFlags = append(bf.CFlags, "-fPIC")
	}

	// Linux hardening
	if isLinux() && !win64 && !opts.Sloppy && !opts.Zap && !opts.Small && !opts.Debug {
		bf.CFlags = append(bf.CFlags, "-fno-plt", "-fstack-protector-strong")
	}

	// Warning flags
	if opts.Sloppy {
		bf.CFlags = append(bf.CFlags, "-fpermissive", "-fms-extensions", "-w")
	} else {
		bf.CFlags = append(bf.CFlags, "-Wall", "-Wshadow", "-Wpedantic", "-Wno-parentheses", "-Wfatal-errors", "-Wvla", "-Wignored-qualifiers")
		if opts.Strict {
			bf.CFlags = append(bf.CFlags, "-Wextra", "-Wconversion", "-Wparentheses", "-Weffc++", "-Wunused-function")
		}
	}

	// Include paths
	for _, ip := range localIncludePaths {
		if fileExists(ip) {
			bf.IncPaths = appendUnique(bf.IncPaths, ip)
		}
	}

	// Directory defines
	bf.Defines = dirDefines()

	// C-specific defines
	if proj.IsC {
		bf.Defines = append(bf.Defines, platformCDefine)
	}

	// OpenMP
	if proj.HasOpenMP {
		bf.CFlags = append(bf.CFlags, "-fopenmp")
		bf.LDFlags = append(bf.LDFlags, "-fopenmp", "-pthread", "-lpthread")
	}

	// Boost
	if proj.HasBoost {
		bf.CFlags = append(bf.CFlags, "-Wno-unknown-pragmas")
		bf.LDFlags = append(bf.LDFlags, "-pthread", "-lpthread")
		// Link boost libraries
		for _, lib := range proj.BoostLibs {
			bf.LDFlags = appendUnique(bf.LDFlags, "-l"+lib)
		}
		// boost_system must come last
		hasBoostLib := false
		for _, lib := range proj.BoostLibs {
			if strings.HasPrefix(lib, "boost_") {
				hasBoostLib = true
				break
			}
		}
		if hasBoostLib {
			// Check if boost_system is available
			out, err := exec.Command("sh", "-c", "ldconfig -p 2>/dev/null | grep boost_system").Output()
			if err == nil && strings.Contains(string(out), "boost_system") {
				bf.LDFlags = appendUnique(bf.LDFlags, "-lboost_system")
			} else if fileExists("/usr/lib/libboost_system.so") {
				bf.LDFlags = appendUnique(bf.LDFlags, "-lboost_system")
			}
		}
	}

	// Threads
	if proj.HasThreads {
		bf.LDFlags = appendUnique(bf.LDFlags, "-lpthread")
	}

	// dlopen
	if proj.HasDlopen {
		bf.LDFlags = appendUnique(bf.LDFlags, "-ldl")
	}

	// Math library
	if proj.HasMathLib {
		bf.LDFlags = appendUnique(bf.LDFlags, "-lm")
	}

	// Filesystem
	if proj.HasFS && isLinux() {
		bf.LDFlags = appendUnique(bf.LDFlags, "-lstdc++fs")
	}

	// Qt6
	if proj.HasQt6 {
		for f := range strings.FieldsSeq(qt6CxxFlags) {
			bf.CFlags = appendUnique(bf.CFlags, f)
		}
		for f := range strings.FieldsSeq(qt6LinkFlags) {
			bf.LDFlags = appendUnique(bf.LDFlags, f)
		}
	}

	// GLFW + Vulkan
	if proj.HasGLFWVulkan {
		bf.LDFlags = appendUnique(bf.LDFlags, "-lvulkan")
	}

	// Profile-guided optimization
	if opts.ProfileGenerate {
		if isCompilerGCC(compiler) {
			bf.CFlags = append(bf.CFlags, "-coverage", "-fprofile-generate", "-fprofile-correction")
			bf.LDFlags = append(bf.LDFlags, "-coverage", "-fprofile-generate", "-fprofile-correction")
		} else if isCompilerClang(compiler) {
			bf.CFlags = append(bf.CFlags, "-fprofile-generate")
			bf.LDFlags = append(bf.LDFlags, "-fprofile-generate")
		}
	} else if opts.ProfileUse {
		if isCompilerGCC(compiler) {
			bf.CFlags = append(bf.CFlags, "-fprofile-use", "-fprofile-correction")
			bf.LDFlags = append(bf.LDFlags, "-fprofile-use", "-fprofile-correction")
		} else if isCompilerClang(compiler) {
			bf.CFlags = append(bf.CFlags, "-fprofile-use")
			bf.LDFlags = append(bf.LDFlags, "-fprofile-use")
		}
	} else {
		// Auto-detect profile data for non-rec builds
		gcdaFiles, _ := filepath.Glob("*.gcda")
		if len(gcdaFiles) > 0 {
			if isCompilerGCC(compiler) {
				bf.CFlags = append(bf.CFlags, "-fprofile-use", "-fprofile-correction")
				bf.LDFlags = append(bf.LDFlags, "-fprofile-use", "-fprofile-correction")
			} else if isCompilerClang(compiler) {
				bf.CFlags = append(bf.CFlags, "-fprofile-use")
				bf.LDFlags = append(bf.LDFlags, "-fprofile-use")
			}
		}
	}

	// Win64 specific flags
	if win64 {
		bf.CFlags = append(bf.CFlags, "-Wno-unused-variable")
		if !proj.IsC {
			bf.CFlags = append(bf.CFlags, "-mwindows", "-fms-extensions")
			bf.LDFlags = append(bf.LDFlags, "-mwindows", "-fms-extensions")
		}
		bf.LDFlags = appendUnique(bf.LDFlags, "-lm")
		// Check for mingw include dir
		mingwDir := "/usr/x86_64-w64-mingw32/include"
		if fileExists(mingwDir) {
			bf.IncPaths = appendUnique(bf.IncPaths, mingwDir)
		}
	}

	// Resolve pkg-config flags for external includes
	if hasPkgConfig() {
		seen := make(map[string]bool)
		for _, inc := range proj.Includes {
			pkgName := pkgNameFromInclude(inc)
			if pkgName == "" || seen[pkgName] {
				continue
			}
			seen[pkgName] = true
			flags := pkgConfigFlags(pkgName)
			if flags != "" {
				bf.CFlags, bf.LDFlags = mergeFlags(bf.CFlags, bf.LDFlags, flags)
			}
		}
	}

	// Resolve extra flags for special includes
	extraCFlags, extraLDFlags := resolveExtraFlags(proj.Includes, win64)
	bf.CFlags = append(bf.CFlags, extraCFlags...)
	bf.LDFlags = append(bf.LDFlags, extraLDFlags...)

	// Resolve via platform package manager for unresolved includes
	pkgCFlags, pkgLDFlags := resolveIncludesViaPackageManager(proj.Includes, systemIncludeDirs(), win64, bf.Compiler)
	bf.CFlags = append(bf.CFlags, pkgCFlags...)
	bf.LDFlags = append(bf.LDFlags, pkgLDFlags...)

	// lib/ directory
	if fileExists("lib") {
		bf.LDFlags = append(bf.LDFlags, "-Llib", "-Wl,-rpath", "./lib")
		soFiles, _ := filepath.Glob("lib/*.so")
		for _, so := range soFiles {
			name := filepath.Base(so)
			name = strings.TrimPrefix(name, "lib")
			name = strings.TrimSuffix(name, ".so")
			bf.LDFlags = append(bf.LDFlags, "-l"+name)
		}
	}

	// Platform-specific library paths (e.g. -L/usr/pkg/lib on NetBSD)
	bf.LDFlags = append(bf.LDFlags, extraLDLibPaths()...)

	// --as-needed (platform-specific: omitted on macOS, -zignore on Solaris)
	bf.LDFlags = prependAsNeededFlag(bf.LDFlags)

	// Append user compile flags from environment
	if proj.IsC {
		if userFlags := os.Getenv("CFLAGS"); userFlags != "" {
			bf.CFlags = append(bf.CFlags, strings.Fields(userFlags)...)
		}
	} else {
		if userFlags := os.Getenv("CXXFLAGS"); userFlags != "" {
			bf.CFlags = append(bf.CFlags, strings.Fields(userFlags)...)
		}
	}

	// Append user LDFLAGS from environment
	if userLDFlags := os.Getenv("LDFLAGS"); userLDFlags != "" {
		bf.LDFlags = append(bf.LDFlags, strings.Fields(userLDFlags)...)
	}

	return bf
}

// doBuild compiles the project.
func doBuild(opts BuildOptions) error {
	proj := detectProject()
	return doBuildWithDirOverrides(opts, proj)
}

// doBuildWithDirOverrides compiles the project with optional install prefix overrides.
func doBuildWithDirOverrides(opts BuildOptions, proj Project) error {
	// Auto-detect win64 from source
	if proj.HasWin64 && !opts.Win64 {
		opts.Win64 = true
	}

	if proj.MainSource == "" && len(proj.TestSources) == 0 {
		return fmt.Errorf("no source files found")
	}

	if proj.MainSource == "" {
		fmt.Println("No main source file found, nothing to build")
		return nil
	}

	exe := executableName()
	if opts.Win64 || proj.HasWin64 {
		exe += ".exe"
	}

	flags := assembleFlags(proj, opts)

	// Override directory defines with install paths if InstallPrefix is set
	if opts.InstallPrefix != "" {
		flags.Defines = installDirDefines(opts.InstallPrefix)
	}

	srcs := append([]string{proj.MainSource}, proj.DepSources...)
	if err := compileSources(srcs, exe, flags); err != nil {
		recommendPackage(proj.Includes)
		platformHints(proj.Includes)
		return err
	}
	return nil
}

// compileSources compiles and links the given source files into the output executable.
// Uses incremental compilation: each source is compiled to a .o file, then linked.
func compileSources(srcs []string, output string, flags BuildFlags) error {
	dirName := filepath.Base(mustGetwd())
	fmt.Printf("[%s] ", dirName)

	// For a single source file, compile directly (no incremental needed)
	if len(srcs) == 1 {
		args := buildCompileArgs(flags, srcs, output)
		cmd := runCompiler(flags, args)
		fmt.Println(flags.Compiler, strings.Join(compactArgs(args), " "))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("compilation failed: %w", err)
		}
		return nil
	}

	// Incremental: compile each source to .o, then link
	var objFiles []string
	needLink := false

	for _, src := range srcs {
		obj := strings.TrimSuffix(src, filepath.Ext(src)) + ".o"
		objFiles = append(objFiles, obj)

		if !needsRecompile(src, obj) {
			continue
		}
		needLink = true

		// Compile source to object (with -MMD for dependency tracking)
		args := []string{"-std=" + flags.Std, "-MMD"}
		args = append(args, flags.CFlags...)
		args = append(args, flags.Defines...)
		for _, ip := range flags.IncPaths {
			args = append(args, "-I"+ip)
		}
		args = append(args, "-c", "-o", obj, src)

		cmd := runCompiler(flags, args)
		fmt.Println(flags.Compiler, strings.Join(compactArgs(args), " "))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("compiling %s: %w", src, err)
		}
	}

	// Check if the output binary exists
	if !fileExists(output) {
		needLink = true
	}

	if !needLink {
		fmt.Println("up to date")
		return nil
	}

	// Link
	args := []string{"-o", output}
	args = append(args, objFiles...)
	args = append(args, flags.LDFlags...)

	cmd := runCompiler(flags, args)
	fmt.Printf("[%s] ", dirName)
	fmt.Println(flags.Compiler, strings.Join(compactArgs(args), " "))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("linking failed: %w", err)
	}

	return nil
}

// buildCompileArgs builds the full compiler arguments for a single-shot compile+link.
func buildCompileArgs(flags BuildFlags, srcs []string, output string) []string {
	args := []string{"-std=" + flags.Std}
	args = append(args, flags.CFlags...)
	args = append(args, flags.Defines...)
	for _, ip := range flags.IncPaths {
		args = append(args, "-I"+ip)
	}
	args = append(args, "-o", output)
	args = append(args, srcs...)
	args = append(args, flags.LDFlags...)
	return args
}

// runCompiler executes the compiler, routing through Docker if DockerImage is set.
func runCompiler(flags BuildFlags, args []string) *exec.Cmd {
	if flags.DockerImage != "" {
		cwd, _ := os.Getwd()
		dockerArgs := []string{"run", "-v", cwd + ":/home", "-w", "/home", "--rm", flags.DockerImage, flags.Compiler}
		dockerArgs = append(dockerArgs, args...)
		return exec.Command("docker", dockerArgs...)
	}
	return exec.Command(flags.Compiler, args...)
}

// needsRecompile checks if the object file needs to be rebuilt.
// Checks both source and header dependencies (via .d files from -MMD).
func needsRecompile(src, obj string) bool {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return true
	}
	objInfo, err := os.Stat(obj)
	if err != nil {
		return true // object doesn't exist
	}
	if srcInfo.ModTime().After(objInfo.ModTime()) {
		return true
	}
	// Check header dependencies from .d file
	depFile := strings.TrimSuffix(obj, ".o") + ".d"
	data, err := os.ReadFile(depFile)
	if err != nil {
		return false // no dep file, trust the source check
	}
	// Parse the .d file: format is "obj: src header1 header2 ..."
	// Lines may be continued with backslash
	content := strings.ReplaceAll(string(data), "\\\n", " ")
	for line := range strings.SplitSeq(content, "\n") {
		if _, after, ok := strings.Cut(line, ":"); ok {
			deps := strings.FieldsSeq(after)
			for dep := range deps {
				depInfo, err := os.Stat(dep)
				if err != nil {
					continue
				}
				if depInfo.ModTime().After(objInfo.ModTime()) {
					return true
				}
			}
		}
	}
	return false
}

// compactArgs shortens the args for display purposes.
func compactArgs(args []string) []string {
	if len(args) <= 20 {
		return args
	}
	result := make([]string, 0, 16)
	result = append(result, args[:10]...)
	result = append(result, "...")
	result = append(result, args[len(args)-5:]...)
	return result
}

func mustGetwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}
