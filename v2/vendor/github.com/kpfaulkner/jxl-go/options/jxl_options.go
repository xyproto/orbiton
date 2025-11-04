package options

import "runtime"

type JXLOptions struct {
	debug           bool
	ParseOnly       bool
	RenderVarblocks bool
	MaxGoroutines   int
}

func NewJXLOptions(options *JXLOptions) *JXLOptions {

	// default goroutines to max.
	opt := &JXLOptions{
		MaxGoroutines: runtime.GOMAXPROCS(-1),
	}

	if options != nil {
		opt.debug = options.debug
		opt.ParseOnly = options.ParseOnly

		if options.MaxGoroutines > 1 {
			opt.MaxGoroutines = options.MaxGoroutines
		}
	}
	return opt
}
