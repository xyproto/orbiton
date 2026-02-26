package orchideous

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BuildResult contains the results of a build operation.
type BuildResult struct {
	OutputExecutable string   // name of the produced executable (relative to sourceDir)
	Output           []byte   // combined compiler output (stdout+stderr)
	CommandsRun      []string // shell-style command strings that were executed
}

// Build compiles the C/C++ project in the given directory.
// It detects sources, dependencies, and required flags automatically.
// The sourceDir is the directory containing the source files.
func Build(sourceDir string, opts BuildOptions) (BuildResult, error) {
	var result BuildResult

	// Save and restore the working directory
	origDir, err := os.Getwd()
	if err != nil {
		return result, fmt.Errorf("could not get working directory: %w", err)
	}
	if err := os.Chdir(sourceDir); err != nil {
		return result, fmt.Errorf("could not change to source directory: %w", err)
	}
	defer os.Chdir(origDir)

	proj := detectProject()

	// Auto-detect win64 from source
	if proj.HasWin64 && !opts.Win64 {
		opts.Win64 = true
	}

	if proj.MainSource == "" {
		return result, fmt.Errorf("no source files found")
	}

	exe := executableName()
	if opts.Win64 || proj.HasWin64 {
		exe += ".exe"
	}
	result.OutputExecutable = exe

	flags := assembleFlags(proj, opts)

	// Override directory defines with install paths if InstallPrefix is set
	if opts.InstallPrefix != "" {
		flags.Defines = installDirDefines(opts.InstallPrefix)
	}

	srcs := append([]string{proj.MainSource}, proj.DepSources...)
	output, cmds, err := compileSourcesCaptured(srcs, exe, flags)
	result.Output = output
	result.CommandsRun = cmds
	if err != nil {
		recommendPackage(proj.Includes)
		return result, err
	}
	return result, nil
}

// compileSourcesCaptured compiles and links the given source files, capturing
// all output instead of printing to stdout/stderr. Returns combined output,
// the list of commands that were run, and any error.
func compileSourcesCaptured(srcs []string, output string, flags BuildFlags) ([]byte, []string, error) {
	var combinedOutput bytes.Buffer
	var commandsRun []string

	// For a single source file, compile directly
	if len(srcs) == 1 {
		args := buildCompileArgs(flags, srcs, output)
		cmd := exec.Command(flags.Compiler, args...)
		commandsRun = append(commandsRun, cmdToString(cmd))
		cmdOutput, err := cmd.CombinedOutput()
		combinedOutput.Write(cmdOutput)
		if err != nil {
			return combinedOutput.Bytes(), commandsRun, fmt.Errorf("compilation failed: %w", err)
		}
		return combinedOutput.Bytes(), commandsRun, nil
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

		args := []string{"-std=" + flags.Std, "-MMD"}
		args = append(args, flags.CFlags...)
		args = append(args, flags.Defines...)
		for _, ip := range flags.IncPaths {
			args = append(args, "-I"+ip)
		}
		args = append(args, "-c", "-o", obj, src)

		cmd := exec.Command(flags.Compiler, args...)
		commandsRun = append(commandsRun, cmdToString(cmd))
		cmdOutput, err := cmd.CombinedOutput()
		combinedOutput.Write(cmdOutput)
		if err != nil {
			return combinedOutput.Bytes(), commandsRun, fmt.Errorf("compiling %s: %w", src, err)
		}
	}

	if !fileExists(output) {
		needLink = true
	}

	if !needLink {
		return combinedOutput.Bytes(), commandsRun, nil
	}

	// Link
	args := []string{"-o", output}
	args = append(args, objFiles...)
	args = append(args, flags.LDFlags...)

	cmd := exec.Command(flags.Compiler, args...)
	commandsRun = append(commandsRun, cmdToString(cmd))
	cmdOutput, err := cmd.CombinedOutput()
	combinedOutput.Write(cmdOutput)
	if err != nil {
		return combinedOutput.Bytes(), commandsRun, fmt.Errorf("linking failed: %w", err)
	}

	return combinedOutput.Bytes(), commandsRun, nil
}

// cmdToString returns a shell-style string representation of a command.
func cmdToString(cmd *exec.Cmd) string {
	return cmd.Path + " " + strings.Join(cmd.Args[1:], " ")
}
