package main

import (
	"github.com/xyproto/mode"
)

// cLikeSwitch checks if the given mode is a language with C-like for expressions
func cLikeFor(m mode.Mode) bool {
	return m == mode.Arduino || m == mode.Beef || m == mode.C || m == mode.Cpp || m == mode.ObjC || m == mode.Shader || m == mode.Zig || m == mode.Java || m == mode.JavaScript || m == mode.Kotlin || m == mode.TypeScript || m == mode.D || m == mode.Dart || m == mode.Hare || m == mode.Jakt || m == mode.Scala
}

// cLikeSwitch checks if the given mode is a language with C-like switch/case expressions
func cLikeSwitch(m mode.Mode) bool {
	return m == mode.Arduino || m == mode.Beef || m == mode.C || m == mode.Cpp || m == mode.ObjC || m == mode.Shader || m == mode.Go || m == mode.Java || m == mode.JavaScript || m == mode.Kotlin || m == mode.TypeScript || m == mode.D || m == mode.Dart || m == mode.Hare || m == mode.Jakt || m == mode.Scala
}

// ProgrammingLanguage returns true if the current mode appears to be a programming language and not a document, configuration format or similar
func ProgrammingLanguage(m mode.Mode) bool {
	// TODO: Update this, and the NoSmartIndentation. Make sure all languages are covered.
	switch m {
	case mode.AIDL, mode.Arduino, mode.Assembly, mode.ASCIIDoc, mode.Amber, mode.Bazel, mode.Blank, mode.Config, mode.Email, mode.FSTAB, mode.Git, mode.GoAssembly, mode.HIDL, mode.HTML, mode.Ini, mode.JSON, mode.Log, mode.M4, mode.ManPage, mode.Markdown, mode.Nmap, mode.Nroff, mode.Odin, mode.PolicyLanguage, mode.ReStructured, mode.SCDoc, mode.SQL, mode.Shader, mode.Text, mode.XML:
		return false
	}
	return true
}

// NoSmartIndentation returns true if the current mode should probably not have smart tab indentation
func (e *Editor) NoSmartIndentation() bool {
	switch e.mode {
	case mode.Assembly, mode.Blank, mode.Email, mode.GoAssembly, mode.Ini, mode.Log, mode.ManPage, mode.Markdown, mode.Nroff, mode.OCaml, mode.Perl, mode.SQL, mode.StandardML, mode.Text:
		return true
	}
	return false
}

// UsingGDBMightWork evaluates if usig GDB might work, for this file type
func (e *Editor) UsingGDBMightWork() bool {
	switch e.mode {
	case mode.AIDL, mode.ASCIIDoc, mode.Amber, mode.Arduino, mode.Basic, mode.Bat, mode.Bazel, mode.Blank, mode.CMake, mode.CS, mode.Clojure, mode.Config, mode.Dart, mode.Email, mode.Erlang, mode.FSTAB, mode.Git, mode.Gradle, mode.HIDL, mode.HTML, mode.Ini, mode.JSON, mode.Java, mode.JavaScript, mode.Just, mode.Kotlin, mode.Lisp, mode.Log, mode.Lua, mode.M4, mode.Make, mode.ManPage, mode.Markdown, mode.Nix, mode.Nroff, mode.Oak, mode.Perl, mode.PolicyLanguage, mode.Python, mode.SCDoc, mode.Starlark, mode.SQL, mode.Scala, mode.Shell, mode.Teal, mode.Text, mode.TypeScript, mode.Vim, mode.XML:
		// Most likely "no"
		return false
	case mode.Zig:
		// Could maybe have worked, but it didn't
		return false
	case mode.Ada, mode.Agda, mode.Algol68, mode.Assembly, mode.Battlestar, mode.Cpp, mode.Crystal, mode.D, mode.Go, mode.GoAssembly, mode.Haskell, mode.Nim, mode.Mojo, mode.ObjC, mode.OCaml, mode.ObjectPascal, mode.Odin, mode.StandardML, mode.V:
		// Maybe, but needs testing!
		return true
	case mode.C, mode.Rust:
		// Yes, tested
		return true
	}
	// Unrecognized, assume that gdb might work with it?
	return true
}

// CanRun checks if the current file mode supports running executables after building
func (e *Editor) CanRun() bool {
	switch e.mode {
	case mode.AIDL, mode.ASCIIDoc, mode.Amber, mode.Bazel, mode.Blank, mode.Config, mode.Email, mode.FSTAB, mode.Git, mode.HIDL, mode.HTML, mode.JSON, mode.Log, mode.M4, mode.ManPage, mode.Markdown, mode.Nroff, mode.PolicyLanguage, mode.ReStructured, mode.SCDoc, mode.SQL, mode.Shader, mode.Text, mode.XML:
		return false
	case mode.Shell: // don't run, because it's not a good idea
		return false
	case mode.Zig: // TODO: Find out why running Zig programs is problematic, terminal emulator wise
		return false
	}
	return true
}
