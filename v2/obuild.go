package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xyproto/slay"
)

func slayDoRun(opts slay.BuildOptions, runArgs []string) error {
	if err := slay.DoBuild(opts); err != nil {
		return err
	}
	exe := slay.ExecutableName()
	if exe == "" {
		return fmt.Errorf("no main source file found")
	}
	if opts.Win64 {
		exe += ".exe"
	}
	exePath := slay.DotSlash(exe)
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

func slayDoClean() {
	exe := slay.ExecutableName()
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
	for _, ts := range slay.GetTestSources() {
		testExe := strings.TrimSuffix(ts, filepath.Ext(ts))
		if err := os.Remove(testExe); err == nil {
			fmt.Println("Removed", testExe)
		}
	}
}

func slayDoFastClean() {
	exe := slay.ExecutableName()
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

func slayDoTest(opts slay.BuildOptions) error {
	testSrcs := slay.GetTestSources()
	if len(testSrcs) == 0 {
		fmt.Println("Nothing to test")
		return nil
	}
	proj := slay.DetectProject()
	flags := slay.AssembleFlags(proj, opts)
	for _, ts := range testSrcs {
		testExe := strings.TrimSuffix(ts, filepath.Ext(ts))
		if opts.Win64 || proj.HasWin64 {
			testExe += ".exe"
		}
		srcs := append([]string{ts}, proj.DepSources...)
		if err := slay.CompileSources(srcs, testExe, flags); err != nil {
			return fmt.Errorf("building test %s: %w", testExe, err)
		}
		fmt.Printf("Running %s...\n", testExe)
		c := exec.Command(slay.DotSlash(testExe))
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("test %s failed: %w", testExe, err)
		}
	}
	return nil
}

func slayDoTestBuild(opts slay.BuildOptions) error {
	proj := slay.DetectProject()
	flags := slay.AssembleFlags(proj, opts)
	if proj.MainSource != "" {
		exe := slay.ExecutableName()
		if opts.Win64 || proj.HasWin64 {
			exe += ".exe"
		}
		srcs := append([]string{proj.MainSource}, proj.DepSources...)
		if err := slay.CompileSources(srcs, exe, flags); err != nil {
			return err
		}
	}
	for _, ts := range proj.TestSources {
		testExe := strings.TrimSuffix(ts, filepath.Ext(ts))
		if opts.Win64 || proj.HasWin64 {
			testExe += ".exe"
		}
		srcs := append([]string{ts}, proj.DepSources...)
		if err := slay.CompileSources(srcs, testExe, flags); err != nil {
			return fmt.Errorf("building test %s: %w", testExe, err)
		}
	}
	if proj.MainSource == "" && len(proj.TestSources) == 0 {
		fmt.Println("Nothing to build")
	}
	return nil
}

func slayDoRec(runArgs []string) error {
	slayDoClean()
	if err := slay.DoBuild(slay.BuildOptions{Opt: true, ProfileGenerate: true}); err != nil {
		return fmt.Errorf("profile generation build: %w", err)
	}
	exe := slay.ExecutableName()
	if exe == "" {
		return fmt.Errorf("no executable to run for profiling")
	}
	c := exec.Command(slay.DotSlash(exe), runArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	_ = c.Run()
	return slay.DoBuild(slay.BuildOptions{Opt: true, ProfileUse: true})
}

func slayDoFmt() error {
	if !has("clang-format") {
		return fmt.Errorf("clang-format not found in PATH")
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
	return nil
}

func slayDoValgrind(opts slay.BuildOptions) error {
	if err := slay.DoBuild(opts); err != nil {
		return err
	}
	exe := slay.ExecutableName()
	if exe == "" {
		return fmt.Errorf("no executable to profile")
	}
	if !has("valgrind") {
		return fmt.Errorf("valgrind not found in PATH")
	}
	exePath := slay.DotSlash(exe)
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

func slayDoDebug(opts slay.BuildOptions, useClang bool) error {
	if err := slay.DoBuild(opts); err != nil {
		return err
	}
	exe := slay.ExecutableName()
	if exe == "" {
		return fmt.Errorf("no executable to debug")
	}
	exePath := slay.DotSlash(exe)
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

func slayDoTiny(opts slay.BuildOptions) error {
	if err := slay.DoBuild(opts); err != nil {
		return err
	}
	exe := slay.ExecutableName()
	if exe == "" {
		return nil
	}
	if opts.Win64 {
		exe += ".exe"
	}
	exePath := slay.DotSlash(exe)
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

// obuildIsModifier returns true if the word is a recognized modifier.
func obuildIsModifier(word string) bool {
	switch word {
	case "clang", "zap", "opt", "strict", "sloppy", "small", "tiny",
		"win64", "win", "debug", "nosan", "nosanitizers":
		return true
	}
	return false
}

// obuildIsAction returns true if the word is a recognized action.
func obuildIsAction(word string) bool {
	switch word {
	case "build", "run", "rebuild", "clean", "fastclean",
		"test", "testbuild", "pgo", "rec", "fmt", "generate",
		"cmakelists", "cmakelist", "cmakelists.txt", "CMakeLists.txt",
		"cmake", "make", "ninja", "install", "pkg", "export", "script",
		"valgrind", "pro", "makefile", "Makefile",
		"ninjainstall", "ninja_install", "ninjaclean", "ninja_clean",
		"makeinstall", "make_install", "makeclean", "make_clean",
		"debug", "version":
		return true
	}
	return false
}

// obuildExpandLegacy expands legacy compound commands into modifier+action pairs.
func obuildExpandLegacy(word string) []string {
	switch word {
	case "debugbuild":
		return []string{"debug", "build"}
	case "debugnosan":
		return []string{"debug", "nosan", "build"}
	case "clangdebug":
		return []string{"clang", "debug"}
	case "clangstrict":
		return []string{"clang", "strict", "build"}
	case "clangsloppy":
		return []string{"clang", "sloppy", "build"}
	case "clangrebuild":
		return []string{"clang", "rebuild"}
	case "clangtest":
		return []string{"clang", "test"}
	case "smallwin64", "smallwin":
		return []string{"small", "win64", "build"}
	case "tinywin64", "tinywin":
		return []string{"tiny", "win64", "build"}
	}
	return nil
}

// runObuild dispatches to slay subcommands using composable modifiers and actions.
// args mirrors the argument style of the "slay" command.
func runObuild(args []string, commandName string) error {
	// Support "-C <dir>" to run in a different directory
	if len(args) >= 2 && args[0] == "-C" {
		if err := os.Chdir(args[1]); err != nil {
			return fmt.Errorf("could not change directory: %w", err)
		}
		args = args[2:]
	}

	// Help
	if len(args) > 0 {
		switch args[0] {
		case "-h", "--help", "help":
			helpTemplate := `Usage: CMD [modifiers...] [action] [args...]

Modifiers (combinable):
  clang       use clang/clang++ compiler
  zap         use zapcc++ compiler
  debug       enable debug flags and sanitizers
  nosan       disable sanitizers (use with debug)
  opt         enable optimizations (-Ofast/-O3, -flto)
  strict      enable strict warning flags
  sloppy      enable permissive flags
  small       optimize for size (-Os)
  tiny        minimize size (-Os + sstrip/upx)
  win64       cross-compile for 64-bit Windows

Actions:
  build       compile the project (default)
  run         build and run
  debug       debug build and launch debugger
  rebuild     clean and build
  clean       remove built files
  fastclean   only remove executable and *.o
  test        build and run tests
  testbuild   build tests (without running)
  pgo         profile-guided optimization (build, run, rebuild)
  fmt         format source code with clang-format
  generate    generate CMakeLists.txt
  makefile    generate a standalone Makefile
  cmake       build with cmake (prefers ninja, falls back to make)
  make        build with make (falls back to cmake+make)
  ninja       build with ninja (falls back to cmake+ninja)
  install     install the project (PREFIX, DESTDIR)
  pkg         package the project into pkg/
  export      export a standalone Makefile and build.sh
  script      generate build.sh and clean.sh
  valgrind    build and profile with valgrind
  pro         generate QtCreator project file

Compound actions:
  ninjainstall  install from ninja build
  ninjaclean    clean ninja build
  makeinstall   install from make/cmake+make build
  makeclean     clean make/cmake+make build

Examples:
  CMD                   standard build
  CMD clang strict      build with clang and strict warnings
  CMD debug             debug build and launch debugger
  CMD debug build       debug build (without launching debugger)
  CMD opt run           optimized build and run
  CMD -C <dir> ...     run in the given directory
`
			fmt.Print(strings.ReplaceAll(helpTemplate, "CMD", commandName))
			return nil
		}
	}

	// No args = default build
	if len(args) == 0 {
		return slay.DoBuild(slay.BuildOptions{})
	}

	// Expand legacy compound commands into modifier+action tokens
	var tokens []string
	for _, arg := range args {
		if expanded := obuildExpandLegacy(arg); expanded != nil {
			tokens = append(tokens, expanded...)
		} else {
			tokens = append(tokens, arg)
		}
	}

	// Parse tokens into modifiers, action, and trailing args
	var opts slay.BuildOptions
	action := ""
	var actionArgs []string

	for i, tok := range tokens {
		// Once we find an action that takes trailing args, capture the rest
		if action == "run" || action == "pgo" {
			actionArgs = tokens[i:]
			break
		}

		// Apply modifiers
		switch tok {
		case "clang":
			opts.Clang = true
			continue
		case "zap":
			opts.Zap = true
			continue
		case "opt":
			opts.Opt = true
			continue
		case "strict":
			opts.Strict = true
			continue
		case "sloppy":
			opts.Sloppy = true
			continue
		case "small":
			opts.Small = true
			continue
		case "tiny":
			opts.Small = true
			opts.Tiny = true
			continue
		case "win64", "win":
			opts.Win64 = true
			continue
		case "nosan", "nosanitizers":
			opts.NoSanitizers = true
			continue
		}

		// "debug" is both a modifier and an action
		if tok == "debug" {
			opts.Debug = true
			if action == "" {
				action = "debug"
			}
			continue
		}

		// Recognized actions
		if obuildIsAction(tok) {
			action = tok
			if tok == "run" || tok == "pgo" || tok == "rec" {
				action = tok
				actionArgs = tokens[i+1:]
				break
			}
			continue
		}

		return fmt.Errorf("unknown build command: %s", tok)
	}

	// Default action
	if action == "" {
		action = "build"
	}

	// Dispatch
	switch action {
	case "build":
		if opts.Tiny {
			return slayDoTiny(opts)
		}
		return slay.DoBuild(opts)
	case "run":
		if opts.Tiny {
			if err := slayDoTiny(opts); err != nil {
				return err
			}
			return slayDoRun(slay.BuildOptions{Win64: opts.Win64}, actionArgs)
		}
		return slayDoRun(opts, actionArgs)
	case "debug":
		return slayDoDebug(opts, opts.Clang)
	case "rebuild":
		slayDoClean()
		if opts.Tiny {
			return slayDoTiny(opts)
		}
		return slay.DoBuild(opts)
	case "clean":
		slayDoClean()
	case "fastclean":
		slayDoFastClean()
	case "test":
		return slayDoTest(opts)
	case "testbuild":
		return slayDoTestBuild(opts)
	case "pgo", "rec":
		return slayDoRec(actionArgs)
	case "fmt":
		return slayDoFmt()
	case "generate", "cmakelists", "cmakelist", "cmakelists.txt", "CMakeLists.txt":
		return slay.DoGenerate(opts)
	case "makefile", "Makefile":
		return slay.DoGenerateMakefile()
	case "cmake":
		return slay.DoCMakeBuild(opts)
	case "make":
		return slay.DoMake()
	case "ninja":
		return slay.DoNinja()
	case "ninjainstall", "ninja_install":
		return slay.DoNinjaInstall()
	case "ninjaclean", "ninja_clean":
		slay.DoNinjaClean()
	case "makeinstall", "make_install":
		return slay.DoMakeInstall()
	case "makeclean", "make_clean":
		slay.DoMakeClean()
	case "install":
		return slay.DoInstall()
	case "pkg":
		return slay.DoPkg()
	case "export":
		return slay.DoExport()
	case "script":
		return slay.DoScript()
	case "valgrind":
		return slayDoValgrind(opts)
	case "pro":
		return slay.DoPro(opts)
	}
	return nil
}
