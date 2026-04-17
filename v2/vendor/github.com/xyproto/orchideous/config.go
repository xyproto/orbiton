package orchideous

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xyproto/files"
)

// Config holds the full configuration for an orchideous operation.
type Config struct {
	BuildOptions
	SourceDir string // if set, operate in this directory instead of the current one
}

// NewConfig returns a Config with default settings (standard build).
func NewConfig() *Config {
	return &Config{}
}

// DebugConfig returns a Config for debug builds with sanitizers.
func DebugConfig() *Config {
	return &Config{BuildOptions: BuildOptions{Debug: true}}
}

// DebugNoSanConfig returns a Config for debug builds without sanitizers.
func DebugNoSanConfig() *Config {
	return &Config{BuildOptions: BuildOptions{Debug: true, NoSanitizers: true}}
}

// OptConfig returns a Config for optimized builds (-Ofast/-O3, -flto).
func OptConfig() *Config {
	return &Config{BuildOptions: BuildOptions{Opt: true}}
}

// SmallConfig returns a Config for size-optimized builds (-Os).
func SmallConfig() *Config {
	return &Config{BuildOptions: BuildOptions{Small: true}}
}

// TinyConfig returns a Config for minimal-size builds (-Os + sstrip/upx).
func TinyConfig() *Config {
	return &Config{BuildOptions: BuildOptions{Small: true, Tiny: true}}
}

// StrictConfig returns a Config with strict warning flags.
func StrictConfig() *Config {
	return &Config{BuildOptions: BuildOptions{Strict: true}}
}

// SloppyConfig returns a Config with permissive compilation flags.
func SloppyConfig() *Config {
	return &Config{BuildOptions: BuildOptions{Sloppy: true}}
}

// ClangConfig returns a Config that uses clang/clang++.
func ClangConfig() *Config {
	return &Config{BuildOptions: BuildOptions{Clang: true}}
}

// Win64Config returns a Config for cross-compiling to 64-bit Windows.
func Win64Config() *Config {
	return &Config{BuildOptions: BuildOptions{Win64: true}}
}

// SmallWin64Config returns a Config for size-optimized Windows cross-compilation.
func SmallWin64Config() *Config {
	return &Config{BuildOptions: BuildOptions{Win64: true, Small: true}}
}

// TinyWin64Config returns a Config for minimal-size Windows cross-compilation.
func TinyWin64Config() *Config {
	return &Config{BuildOptions: BuildOptions{Win64: true, Small: true, Tiny: true}}
}

// ZapConfig returns a Config that uses zapcc++.
func ZapConfig() *Config {
	return &Config{BuildOptions: BuildOptions{Zap: true}}
}

// withDir executes fn in the configured SourceDir, restoring the original
// directory afterward. If SourceDir is empty, fn runs in the current directory.
func (c *Config) withDir(fn func() error) error {
	if c.SourceDir == "" {
		return fn()
	}
	orig, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get working directory: %w", err)
	}
	if err := os.Chdir(c.SourceDir); err != nil {
		return fmt.Errorf("could not change to %s: %w", c.SourceDir, err)
	}
	defer os.Chdir(orig)
	return fn()
}

// withDirNoErr is like withDir but for functions that don't return errors.
func (c *Config) withDirNoErr(fn func()) {
	if c.SourceDir != "" {
		if orig, err := os.Getwd(); err == nil {
			if err := os.Chdir(c.SourceDir); err == nil {
				defer os.Chdir(orig)
			}
		}
	}
	fn()
}

// hasCommand returns true if the given command is available in PATH.
func hasCommand(name string) bool {
	return files.WhichCached(name) != ""
}

// Build compiles the project using the configured options.
func (c *Config) Build() error {
	return c.withDir(func() error {
		return doBuild(c.BuildOptions)
	})
}

// Run builds the project and runs the resulting executable.
func (c *Config) Run(args ...string) error {
	return c.withDir(func() error {
		if err := doBuild(c.BuildOptions); err != nil {
			return err
		}
		exe := executableName()
		if exe == "" {
			return fmt.Errorf("no main source file found")
		}
		// Handle both explicit win64 and auto-detected win64 (proj.HasWin64)
		if c.Win64 || !fileExists(exe) && fileExists(exe+".exe") {
			exe += ".exe"
		}
		exePath := dotSlash(exe)
		if strings.HasSuffix(exePath, ".exe") {
			if winePath := files.WhichCached("wine"); winePath != "" {
				cmd := exec.Command(winePath, append([]string{exePath}, args...)...)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				return cmd.Run()
			}
		}
		cmd := exec.Command(exePath, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	})
}

// cleanFiles removes build artifacts from the current directory.
func cleanFiles() {
	exe := executableName()
	patterns := []string{"*.o", "*.d", "common/*.o", "common/*.d", "include/*.o", "include/*.d", "*.profraw", "*.gcda", "*.gcno", ".sconsign.dblite", "callgrind.out.*"}
	for _, pat := range patterns {
		matches, _ := filepath.Glob(pat)
		for _, f := range matches {
			os.Remove(f)
			fmt.Println("Removed", f)
		}
	}
	if exe != "" {
		if err := os.Remove(exe); err == nil {
			fmt.Println("Removed", exe)
		}
		if err := os.Remove(exe + ".exe"); err == nil {
			fmt.Println("Removed", exe+".exe")
		}
	}
	testSrcs := getTestSources()
	for _, ts := range testSrcs {
		testExe := strings.TrimSuffix(ts, filepath.Ext(ts))
		if err := os.Remove(testExe); err == nil {
			fmt.Println("Removed", testExe)
		}
	}
}

// fastCleanFiles removes only the executable and object files from the current directory.
func fastCleanFiles() {
	exe := executableName()
	matches, _ := filepath.Glob("*.o")
	for _, f := range matches {
		os.Remove(f)
		fmt.Println("Removed", f)
	}
	if exe != "" {
		if err := os.Remove(exe); err == nil {
			fmt.Println("Removed", exe)
		}
		if err := os.Remove(exe + ".exe"); err == nil {
			fmt.Println("Removed", exe+".exe")
		}
	}
}

// Clean removes all build artifacts: runs make clean, ninja clean,
// removes the build/ directory, and cleans object files and executables.
func (c *Config) Clean() {
	c.withDirNoErr(doCleanAll)
}

// FastClean removes only the executable and object files.
func (c *Config) FastClean() {
	c.withDirNoErr(fastCleanFiles)
}

// Rebuild cleans and then builds the project.
func (c *Config) Rebuild() error {
	c.Clean()
	return c.Build()
}

// Test builds and runs all test files.
func (c *Config) Test() error {
	return c.withDir(func() error {
		testSrcs := getTestSources()
		if len(testSrcs) == 0 {
			fmt.Println("Nothing to test")
			return nil
		}
		proj := detectProject()
		flags := assembleFlags(proj, c.BuildOptions)
		for _, ts := range testSrcs {
			testExe := strings.TrimSuffix(ts, filepath.Ext(ts))
			if c.Win64 || proj.HasWin64 {
				testExe += ".exe"
			}
			srcs := append([]string{ts}, proj.DepSources...)
			if err := compileSources(srcs, testExe, flags); err != nil {
				return fmt.Errorf("building test %s: %w", testExe, err)
			}
			fmt.Printf("Running %s...\n", testExe)
			cmd := exec.Command(dotSlash(testExe))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("test %s failed: %w", testExe, err)
			}
		}
		return nil
	})
}

// TestBuild builds the main executable and all test files without running them.
func (c *Config) TestBuild() error {
	return c.withDir(func() error {
		proj := detectProject()
		flags := assembleFlags(proj, c.BuildOptions)
		if proj.MainSource != "" {
			exe := executableName()
			if c.Win64 || proj.HasWin64 {
				exe += ".exe"
			}
			srcs := append([]string{proj.MainSource}, proj.DepSources...)
			if err := compileSources(srcs, exe, flags); err != nil {
				return err
			}
		}
		for _, ts := range proj.TestSources {
			testExe := strings.TrimSuffix(ts, filepath.Ext(ts))
			if c.Win64 || proj.HasWin64 {
				testExe += ".exe"
			}
			srcs := append([]string{ts}, proj.DepSources...)
			if err := compileSources(srcs, testExe, flags); err != nil {
				return fmt.Errorf("building test %s: %w", testExe, err)
			}
		}
		if proj.MainSource == "" && len(proj.TestSources) == 0 {
			fmt.Println("Nothing to build")
		}
		return nil
	})
}

// LaunchDebugger builds a debug version and launches the appropriate debugger.
// Uses lldb first when Clang is set, otherwise prefers cgdb/gdb.
func (c *Config) LaunchDebugger() error {
	return c.withDir(func() error {
		if err := doBuild(c.BuildOptions); err != nil {
			return err
		}
		exe := executableName()
		if exe == "" {
			return fmt.Errorf("no executable to debug")
		}
		exePath := dotSlash(exe)

		var debugger string
		if c.Clang {
			for _, d := range []string{"lldb", "gdb", "cgdb"} {
				if hasCommand(d) {
					debugger = d
					break
				}
			}
		} else {
			for _, d := range []string{"cgdb", "gdb", "lldb"} {
				if hasCommand(d) {
					debugger = d
					break
				}
			}
		}
		if debugger == "" {
			return fmt.Errorf("no debugger found\n  hint: %s", installHint("gdb"))
		}

		cmd := exec.Command(debugger, exePath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
		return cmd.Run()
	})
}

// TinyBuild builds with the configured options and applies sstrip/upx post-processing.
func (c *Config) TinyBuild() error {
	return c.withDir(func() error {
		if err := doBuild(c.BuildOptions); err != nil {
			return err
		}
		exe := executableName()
		if exe == "" {
			return nil
		}
		// Handle both explicit win64 and auto-detected win64 (proj.HasWin64)
		if c.Win64 || !fileExists(exe) && fileExists(exe+".exe") {
			exe += ".exe"
		}
		exePath := dotSlash(exe)

		if hasCommand("sstrip") {
			cmd := exec.Command("sstrip", exePath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				fmt.Println("sstrip", exePath)
			}
		}
		if hasCommand("upx") {
			cmd := exec.Command("upx", "--brute", exePath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				fmt.Println("upx --brute", exePath)
			}
		}
		return nil
	})
}

// Rec performs profile-guided optimization: clean, build with profiling, run, rebuild with profile data.
func (c *Config) Rec(args ...string) error {
	return c.withDir(func() error {
		cleanFiles()

		if err := doBuild(BuildOptions{Opt: true, ProfileGenerate: true}); err != nil {
			return fmt.Errorf("profile generation build: %w", err)
		}
		exe := executableName()
		if exe == "" {
			return fmt.Errorf("no executable to run for profiling")
		}
		exePath := dotSlash(exe)
		cmd := exec.Command(exePath, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()

		// For clang-style PGO: merge .profraw files into default.profdata
		if profrawFiles, _ := filepath.Glob("*.profraw"); len(profrawFiles) > 0 {
			profdata := files.WhichCached("llvm-profdata")
			if profdata == "" {
				if out, err := exec.Command("xcrun", "-f", "llvm-profdata").Output(); err == nil {
					profdata = strings.TrimSpace(string(out))
				}
			}
			if profdata != "" {
				mergeArgs := append([]string{"merge", "-o", "default.profdata"}, profrawFiles...)
				mergeCmd := exec.Command(profdata, mergeArgs...)
				mergeCmd.Stdout = os.Stdout
				mergeCmd.Stderr = os.Stderr
				if err := mergeCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "warning: llvm-profdata merge failed: %v\n", err)
				}
			}
		}

		// Remove object files so the next build recompiles with PGO
		for _, pat := range []string{"*.o", "common/*.o", "include/*.o"} {
			if matches, _ := filepath.Glob(pat); len(matches) > 0 {
				for _, f := range matches {
					os.Remove(f)
				}
			}
		}
		return doBuild(BuildOptions{Opt: true, ProfileUse: true})
	})
}

// Fmt formats source code using clang-format.
func (c *Config) Fmt() error {
	return c.withDir(func() error {
		if !hasCommand("clang-format") {
			return fmt.Errorf("clang-format not found in PATH\n  hint: %s", installHint("clang-format"))
		}
		exts := []string{"cpp", "cc", "cxx", "h", "hpp", "hh", "h++"}
		dirs := []string{".", "include", "common"}
		formatted := 0
		for _, dir := range dirs {
			for _, ext := range exts {
				matches, _ := filepath.Glob(filepath.Join(dir, "*."+ext))
				for _, f := range matches {
					cmd := exec.Command("clang-format", "-style={BasedOnStyle: Webkit, ColumnLimit: 99}", "-i", f)
					if err := cmd.Run(); err != nil {
						fmt.Fprintf(os.Stderr, "warning: clang-format failed on %s: %v\n", f, err)
					} else {
						formatted++
					}
				}
			}
		}
		if formatted > 0 {
			fmt.Printf("Formatted %d file(s)\n", formatted)
		} else {
			fmt.Println("No source files found to format")
		}
		return nil
	})
}

// Valgrind builds the project and profiles it with valgrind/callgrind.
func (c *Config) Valgrind() error {
	return c.withDir(func() error {
		if err := doBuild(c.BuildOptions); err != nil {
			return err
		}
		exe := executableName()
		if exe == "" {
			return fmt.Errorf("no executable to profile")
		}
		if !hasCommand("valgrind") {
			return fmt.Errorf("valgrind not found in PATH\n  hint: %s", installHint("valgrind"))
		}
		exePath := dotSlash(exe)
		cmd := exec.Command("valgrind", "--tool=callgrind", exePath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: valgrind exited with: %v\n", err)
		}
		callgrindFiles, _ := filepath.Glob("callgrind.out.*")
		if len(callgrindFiles) > 0 && hasCommand("gprof2dot") && hasCommand("dot") {
			cmd = exec.Command("sh", "-c",
				"gprof2dot -f callgrind "+callgrindFiles[0]+" | dot -Tsvg -o output.svg")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			_ = cmd.Run()
		}
		if len(callgrindFiles) > 0 && hasCommand("kcachegrind") {
			cmd = exec.Command("kcachegrind", callgrindFiles[0])
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			_ = cmd.Run()
		}
		return nil
	})
}

// Generate generates a CMakeLists.txt file.
func (c *Config) Generate() error {
	return c.withDir(func() error {
		return doGenerate(c.BuildOptions)
	})
}

// CMakeBuild builds using cmake, preferring ninja over make.
// Generates CMakeLists.txt first if it does not exist.
func (c *Config) CMakeBuild() error {
	return c.withDir(func() error {
		return doCMakeBuild(c.BuildOptions)
	})
}

// Ninja builds using ninja. If build/build.ninja exists, runs ninja directly.
// Otherwise falls back to cmake+ninja if CMakeLists.txt exists.
func (c *Config) Ninja() error {
	return c.withDir(func() error {
		return doNinja()
	})
}

// Make builds using make. If a Makefile exists, runs make directly.
// Otherwise falls back to cmake+make if CMakeLists.txt exists.
func (c *Config) Make() error {
	return c.withDir(func() error {
		return doMake()
	})
}

// NinjaInstall installs from a ninja build.
func (c *Config) NinjaInstall() error {
	return c.withDir(func() error {
		return doNinjaInstall()
	})
}

// NinjaClean removes the ninja build directory.
func (c *Config) NinjaClean() {
	c.withDirNoErr(doNinjaClean)
}

// CMakeMakeInstall installs from a cmake+make build.
func (c *Config) CMakeMakeInstall() error {
	return c.withDir(func() error {
		return doCMakeMakeInstall()
	})
}

// CMakeMakeClean removes the cmake+make build directory.
func (c *Config) CMakeMakeClean() {
	c.withDirNoErr(doCMakeMakeClean)
}

// Pro generates a QtCreator .pro project file.
func (c *Config) Pro() error {
	return c.withDir(func() error {
		return doPro(c.BuildOptions)
	})
}

// Install builds and installs the project (using PREFIX and DESTDIR environment variables).
func (c *Config) Install() error {
	return c.withDir(func() error {
		return doInstall()
	})
}

// Pkg packages the project into a pkg/ directory.
func (c *Config) Pkg() error {
	return c.withDir(func() error {
		return doPkg()
	})
}

// Export generates a standalone Makefile, build.sh, and clean.sh.
func (c *Config) Export() error {
	return c.withDir(func() error {
		return doExport()
	})
}

// GenerateMakefile generates a standalone Makefile.
func (c *Config) GenerateMakefile() error {
	return c.withDir(func() error {
		return doGenerateMakefile()
	})
}

// Script generates build.sh and clean.sh scripts.
func (c *Config) Script() error {
	return c.withDir(func() error {
		return doScript()
	})
}
