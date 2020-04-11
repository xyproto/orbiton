package main

// Mode is a per-filetype mode, like for Markdown
type Mode int

const (
	blankMode    = iota
	gitMode      // for git commits and interactive rebases
	markdownMode // for Markdown (and asciidoctor and rst files)
	makefileMode // for Makefiles
	shellMode    // for shell scripts and PKGBUILD files
	ymlMode      // for yml and toml files
)
