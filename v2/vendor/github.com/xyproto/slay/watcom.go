package slay

import (
	"os"
	"path/filepath"
	"strings"
)

func findWatcomRoot() string {
	for _, dir := range []string{os.Getenv("WATCOM"), "/opt/watcom", "/usr/lib/watcom"} {
		if dir != "" && fileExists(filepath.Join(dir, "binl", "wcl")) {
			return dir
		}
	}
	return ""
}

func isWatcomCompiler(compiler string) bool {
	return filepath.Base(compiler) == "wcl"
}

func findWatcomCompiler() string {
	root := findWatcomRoot()
	if root == "" {
		return ""
	}
	os.Setenv("WATCOM", root)
	os.Setenv("INCLUDE", filepath.Join(root, "h"))
	os.Setenv("PATH", filepath.Join(root, "binl")+string(os.PathListSeparator)+os.Getenv("PATH"))
	return filepath.Join(root, "binl", "wcl")
}

func assembleWatcomFlags(proj Project, opts BuildOptions) BuildFlags {
	bf := BuildFlags{Watcom: true, Compiler: findWatcomCompiler()}
	if root := findWatcomRoot(); root != "" {
		bf.IncPaths = append(bf.IncPaths, filepath.Join(root, "h"))
	}
	bf.CFlags = append(bf.CFlags, "-q", "-bt=dos", "-ms")
	if proj.IsC {
		bf.CFlags = append(bf.CFlags, "-za99")
	}
	if opts.Debug {
		bf.CFlags = append(bf.CFlags, "-od", "-d2")
	} else if opts.Small || opts.Tiny {
		bf.CFlags = append(bf.CFlags, "-os")
	} else {
		bf.CFlags = append(bf.CFlags, "-ox")
	}
	if opts.Sloppy {
		bf.CFlags = append(bf.CFlags, "-w0")
	} else if opts.Strict {
		bf.CFlags = append(bf.CFlags, "-w4")
	} else {
		bf.CFlags = append(bf.CFlags, "-w3")
	}
	bf.Defines = dirDefines()
	return bf
}

func watcomArgs(flags BuildFlags, srcs []string, output string) []string {
	args := append([]string{}, flags.CFlags...)
	for _, d := range flags.Defines {
		args = append(args, "-d"+strings.TrimPrefix(d, "-D"))
	}
	for _, ip := range flags.IncPaths {
		args = append(args, "-i="+ip)
	}
	args = append(args, flags.LDFlags...)
	args = append(args, srcs...)
	return append(args, "-fe="+output)
}
