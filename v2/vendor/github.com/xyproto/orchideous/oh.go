package orchideous

// Exported functions for use by cmd/oh

func DoBuild(opts BuildOptions) error                          { return doBuild(opts) }
func ExecutableName() string                                   { return executableName() }
func GetTestSources() []string                                 { return getTestSources() }
func DetectProject() Project                                   { return detectProject() }
func AssembleFlags(proj Project, opts BuildOptions) BuildFlags { return assembleFlags(proj, opts) }
func CompileSources(srcs []string, output string, flags BuildFlags) error {
	return compileSources(srcs, output, flags)
}
func DoCMake(opts BuildOptions) error { return doCMake(opts) }
func DoPro(opts BuildOptions) error   { return doPro(opts) }
func DoNinja() error                  { return doNinja() }
func DoNinjaInstall() error           { return doNinjaInstall() }
func DoNinjaClean()                   { doNinjaClean() }
func DoInstall() error                { return doInstall() }
func DoPkg() error                    { return doPkg() }
func DoExport() error                 { return doExport() }
func DoMakeFile() error               { return doMakeFile() }
func DoScript() error                 { return doScript() }
func DotSlash(name string) string     { return dotSlash(name) }
