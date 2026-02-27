package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xyproto/orchideous"
)

func ohDoRun(opts orchideous.BuildOptions, runArgs []string) error {
	if err := orchideous.DoBuild(opts); err != nil {
		return err
	}
	exe := orchideous.ExecutableName()
	if exe == "" {
		return fmt.Errorf("no main source file found")
	}
	if opts.Win64 {
		exe += ".exe"
	}
	exePath := orchideous.DotSlash(exe)
	if strings.HasSuffix(exePath, ".exe") {
		if has("wine") {
			c := exec.Command("wine", append([]string{exePath}, runArgs...)...)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}
	}
	c := exec.Command(exePath, runArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func ohDoClean() {
	exe := orchideous.ExecutableName()
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
	for _, ts := range orchideous.GetTestSources() {
		testExe := strings.TrimSuffix(ts, filepath.Ext(ts))
		if err := os.Remove(testExe); err == nil {
			fmt.Println("Removed", testExe)
		}
	}
}

func ohDoFastClean() {
	exe := orchideous.ExecutableName()
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

func ohDoTest(opts orchideous.BuildOptions) error {
	testSrcs := orchideous.GetTestSources()
	if len(testSrcs) == 0 {
		fmt.Println("Nothing to test")
		return nil
	}
	proj := orchideous.DetectProject()
	flags := orchideous.AssembleFlags(proj, opts)
	for _, ts := range testSrcs {
		testExe := strings.TrimSuffix(ts, filepath.Ext(ts))
		if opts.Win64 || proj.HasWin64 {
			testExe += ".exe"
		}
		srcs := append([]string{ts}, proj.DepSources...)
		if err := orchideous.CompileSources(srcs, testExe, flags); err != nil {
			return fmt.Errorf("building test %s: %w", testExe, err)
		}
		fmt.Printf("Running %s...\n", testExe)
		c := exec.Command(orchideous.DotSlash(testExe))
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("test %s failed: %w", testExe, err)
		}
	}
	return nil
}

func ohDoTestBuild(opts orchideous.BuildOptions) error {
	proj := orchideous.DetectProject()
	flags := orchideous.AssembleFlags(proj, opts)
	if proj.MainSource != "" {
		exe := orchideous.ExecutableName()
		if opts.Win64 || proj.HasWin64 {
			exe += ".exe"
		}
		srcs := append([]string{proj.MainSource}, proj.DepSources...)
		if err := orchideous.CompileSources(srcs, exe, flags); err != nil {
			return err
		}
	}
	for _, ts := range proj.TestSources {
		testExe := strings.TrimSuffix(ts, filepath.Ext(ts))
		if opts.Win64 || proj.HasWin64 {
			testExe += ".exe"
		}
		srcs := append([]string{ts}, proj.DepSources...)
		if err := orchideous.CompileSources(srcs, testExe, flags); err != nil {
			return fmt.Errorf("building test %s: %w", testExe, err)
		}
	}
	if proj.MainSource == "" && len(proj.TestSources) == 0 {
		fmt.Println("Nothing to build")
	}
	return nil
}

func ohDoRec(runArgs []string) error {
	ohDoClean()
	if err := orchideous.DoBuild(orchideous.BuildOptions{Opt: true, ProfileGenerate: true}); err != nil {
		return fmt.Errorf("profile generation build: %w", err)
	}
	exe := orchideous.ExecutableName()
	if exe == "" {
		return fmt.Errorf("no executable to run for profiling")
	}
	c := exec.Command(orchideous.DotSlash(exe), runArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	_ = c.Run()
	return orchideous.DoBuild(orchideous.BuildOptions{Opt: true, ProfileUse: true})
}

func ohDoFmt() {
	if !has("clang-format") {
		fmt.Fprintln(os.Stderr, "error: clang-format not found in PATH")
		os.Exit(1)
	}
	exts := []string{"cpp", "cc", "cxx", "h", "hpp", "hh", "h++"}
	dirs := []string{".", "include", "common"}
	for _, dir := range dirs {
		for _, ext := range exts {
			matches, _ := filepath.Glob(filepath.Join(dir, "*."+ext))
			for _, f := range matches {
				c := exec.Command("clang-format", "-style={BasedOnStyle: Webkit, ColumnLimit: 99}", "-i", f)
				_ = c.Run()
			}
		}
	}
}

func ohDoValgrind(opts orchideous.BuildOptions) error {
	if err := orchideous.DoBuild(opts); err != nil {
		return err
	}
	exe := orchideous.ExecutableName()
	if exe == "" {
		return fmt.Errorf("no executable to profile")
	}
	if !has("valgrind") {
		return fmt.Errorf("valgrind not found in PATH")
	}
	exePath := orchideous.DotSlash(exe)
	c := exec.Command("valgrind", "--tool=callgrind", exePath)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: valgrind exited with: %v\n", err)
	}
	callgrindFiles, _ := filepath.Glob("callgrind.out.*")
	if len(callgrindFiles) > 0 && has("gprof2dot") && has("dot") {
		c = exec.Command("sh", "-c", "gprof2dot -f callgrind "+callgrindFiles[0]+" | dot -Tsvg -o output.svg")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		_ = c.Run()
	}
	if len(callgrindFiles) > 0 && has("kcachegrind") {
		c = exec.Command("kcachegrind", callgrindFiles[0])
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		_ = c.Run()
	}
	return nil
}

func ohDoDebug(opts orchideous.BuildOptions, useClang bool) error {
	if err := orchideous.DoBuild(opts); err != nil {
		return err
	}
	exe := orchideous.ExecutableName()
	if exe == "" {
		return fmt.Errorf("no executable to debug")
	}
	exePath := orchideous.DotSlash(exe)
	var debugger string
	if useClang {
		for _, d := range []string{"lldb", "gdb", "cgdb"} {
			if has(d) {
				debugger = d
				break
			}
		}
	} else {
		for _, d := range []string{"cgdb", "gdb", "lldb"} {
			if has(d) {
				debugger = d
				break
			}
		}
	}
	if debugger == "" {
		return fmt.Errorf("no debugger found (tried cgdb, gdb, lldb)")
	}
	c := exec.Command(debugger, exePath)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
	return c.Run()
}

func ohDoTiny(opts orchideous.BuildOptions) error {
	if err := orchideous.DoBuild(opts); err != nil {
		return err
	}
	exe := orchideous.ExecutableName()
	if exe == "" {
		return nil
	}
	if opts.Win64 {
		exe += ".exe"
	}
	exePath := orchideous.DotSlash(exe)
	if has("sstrip") {
		c := exec.Command("sstrip", exePath)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err == nil {
			fmt.Println("sstrip", exePath)
		}
	}
	if has("upx") {
		c := exec.Command("upx", "--brute", exePath)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err == nil {
			fmt.Println("upx --brute", exePath)
		}
	}
	return nil
}

// runObuild dispatches to orchideous (oh) subcommands.
// args mirrors the argument style of the "oh" command.
func runObuild(args []string, commandName string) error {
	// Support "-C <dir>" to run in a different directory
	if len(args) >= 2 && args[0] == "-C" {
		if err := os.Chdir(args[1]); err != nil {
			return fmt.Errorf("could not change directory: %w", err)
		}
		args = args[2:]
	}

	cmd := "build"
	if len(args) > 0 {
		cmd = args[0]
	}

	subArgs := args
	if len(subArgs) > 0 {
		subArgs = subArgs[1:]
	}

	switch cmd {
	case "-h", "--help", "help":
		helpString := `oh              - build the project
oh run          - build and run
oh debug        - debug build and launch debugger (gdb/cgdb)
oh debugbuild   - debug build (without launching debugger)
oh debugnosan   - debug build (without sanitizers)
oh opt          - optimized build
oh strict       - build with strict warning flags
oh sloppy       - build with sloppy flags
oh small        - build a smaller executable
oh tiny         - build a tiny executable (+ sstrip/upx)
oh clang        - build using clang++
oh clangdebug   - debug build using clang++ (launches lldb)
oh clangstrict  - use clang++ and strict flags
oh clangsloppy  - use clang++ and sloppy flags
oh clangrebuild - clean and build with clang++
oh clangtest    - build and run tests with clang++
oh clean        - remove built files
oh fastclean    - only remove executable and *.o
oh rebuild      - clean and build
oh test         - build and run tests
oh testbuild    - build tests (without running)
oh rec          - profile-guided optimization (build, run, rebuild)
oh fmt          - format source code with clang-format
oh cmake        - generate CMakeLists.txt
oh cmake ninja  - generate CMakeLists.txt and build with ninja
oh ninja        - build using existing CMakeLists.txt and ninja
oh ninja_install- install from ninja build
oh ninja_clean  - clean ninja build
oh pro          - generate QtCreator project file
oh install      - install the project (PREFIX, DESTDIR)
oh pkg          - package the project into pkg/
oh export       - export a standalone Makefile and build.sh
oh make         - generate a standalone Makefile
oh script       - generate build.sh and clean.sh
oh valgrind     - build and profile with valgrind
oh win64        - cross-compile for 64-bit Windows
oh smallwin64   - small win64 build
oh tinywin64    - tiny win64 build
oh zap          - build using zapcc++
oh -C <dir> ... - run in the given directory
`
		fmt.Print(strings.ReplaceAll(helpString, "oh ", commandName+" "))
	case "build":
		return orchideous.DoBuild(orchideous.BuildOptions{})
	case "rebuild":
		ohDoClean()
		return orchideous.DoBuild(orchideous.BuildOptions{})
	case "clean":
		ohDoClean()
	case "fastclean":
		ohDoFastClean()
	case "run":
		return ohDoRun(orchideous.BuildOptions{}, subArgs)
	case "debug":
		return ohDoDebug(orchideous.BuildOptions{Debug: true}, false)
	case "debugbuild":
		return orchideous.DoBuild(orchideous.BuildOptions{Debug: true})
	case "debugnosan":
		return orchideous.DoBuild(orchideous.BuildOptions{Debug: true, NoSanitizers: true})
	case "opt":
		return orchideous.DoBuild(orchideous.BuildOptions{Opt: true})
	case "strict":
		return orchideous.DoBuild(orchideous.BuildOptions{Strict: true})
	case "sloppy":
		return orchideous.DoBuild(orchideous.BuildOptions{Sloppy: true})
	case "small":
		return orchideous.DoBuild(orchideous.BuildOptions{Small: true})
	case "tiny":
		return ohDoTiny(orchideous.BuildOptions{Small: true, Tiny: true})
	case "clang":
		return orchideous.DoBuild(orchideous.BuildOptions{Clang: true})
	case "clangdebug":
		return ohDoDebug(orchideous.BuildOptions{Clang: true, Debug: true}, true)
	case "clangstrict":
		return orchideous.DoBuild(orchideous.BuildOptions{Clang: true, Strict: true})
	case "clangsloppy":
		return orchideous.DoBuild(orchideous.BuildOptions{Clang: true, Sloppy: true})
	case "clangrebuild":
		ohDoClean()
		return orchideous.DoBuild(orchideous.BuildOptions{Clang: true})
	case "clangtest":
		return ohDoTest(orchideous.BuildOptions{Clang: true})
	case "test":
		return ohDoTest(orchideous.BuildOptions{})
	case "testbuild":
		return ohDoTestBuild(orchideous.BuildOptions{})
	case "rec":
		return ohDoRec(subArgs)
	case "fmt":
		ohDoFmt()
	case "cmake":
		if len(subArgs) > 0 && subArgs[0] == "ninja" {
			if err := orchideous.DoCMake(orchideous.BuildOptions{}); err != nil {
				return err
			}
			return orchideous.DoNinja()
		}
		return orchideous.DoCMake(orchideous.BuildOptions{})
	case "pro":
		return orchideous.DoPro(orchideous.BuildOptions{})
	case "ninja":
		return orchideous.DoNinja()
	case "ninja_install":
		return orchideous.DoNinjaInstall()
	case "ninja_clean":
		orchideous.DoNinjaClean()
	case "install":
		return orchideous.DoInstall()
	case "pkg":
		return orchideous.DoPkg()
	case "export":
		return orchideous.DoExport()
	case "make":
		return orchideous.DoMakeFile()
	case "script":
		return orchideous.DoScript()
	case "valgrind":
		return ohDoValgrind(orchideous.BuildOptions{})
	case "win", "win64":
		return orchideous.DoBuild(orchideous.BuildOptions{Win64: true})
	case "smallwin", "smallwin64":
		return orchideous.DoBuild(orchideous.BuildOptions{Win64: true, Small: true})
	case "tinywin", "tinywin64":
		return ohDoTiny(orchideous.BuildOptions{Win64: true, Small: true, Tiny: true})
	case "zap":
		return orchideous.DoBuild(orchideous.BuildOptions{Zap: true})
	default:
		return fmt.Errorf("unknown build command: %s", cmd)
	}
	return nil
}
