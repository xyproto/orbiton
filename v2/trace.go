//go:build trace

package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/felixge/fgtrace"
)

var (
	cpuProfileFilename *string
	memProfileFilename *string
	fgtraceFilename    *string
)

func init() {
	cpuProfileFilename = flag.String("cpuprofile", "", "write CPU profile to `file`")
	memProfileFilename = flag.String("memprofile", "", "write memory profile to `file`")
	fgtraceFilename = flag.String("fgtrace", "", "write fgtrace to `file`")
}

func traceStart() {
	// Output CPU profile information, if a filename is given
	if *cpuProfileFilename != "" {
		f, err := os.Create(*cpuProfileFilename)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		// runtime.SetCPUProfileRate(500)
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	if *fgtraceFilename != "" {
		defer fgtrace.Config{Dst: fgtrace.File(*fgtraceFilename)}.Trace().Stop()
	}
}

func traceComplete() {
	// Output memory profile information, if a filename is given
	if *memProfileFilename != "" {
		f, err := os.Create(*memProfileFilename)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}
