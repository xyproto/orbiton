//go:build trace

package main

import (
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/felixge/fgtrace"
	"github.com/spf13/pflag"
)

var (
	cpuProfileFilename string
	memProfileFilename string
	fgtraceFilename    string
	cpuProfileFile     os.File
)

func init() {
	pflag.StringVarP(&cpuProfileFilename, "cpuprofile", "u", "", "write CPU profile to `file`")
	pflag.StringVarP(&memProfileFilename, "memprofile", "w", "", "write memory profile to `file`")
	pflag.StringVarP(&fgtraceFilename, "fgtrace", "y", "", "write fgtrace to `file`")

	// Start the pprof HTTP server as well
	go func() {
		log.Println("Starting pprof server at :6060")
		log.Println("Try: http://localhost:6060/debug/pprof/goroutine?debug=2")
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	// Give it some time to be able to show the log messages from the goroutine above
	time.Sleep(1200 * time.Millisecond)
}

func traceStart() {
	// Output CPU profile information, if a filename is given
	if cpuProfileFilename != "" {
		cpuProfileFile, err := os.Create(cpuProfileFilename)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}

		// Set the rate and start profiling the CPU usage
		// runtime.SetCPUProfileRate(500)
		if err := pprof.StartCPUProfile(cpuProfileFile); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}

	if fgtraceFilename != "" {
		defer fgtrace.Config{Dst: fgtrace.File(fgtraceFilename)}.Trace().Stop()
	}
}

func traceComplete() {
	// Output memory profile information, if a filename is given
	if memProfileFilename != "" {
		f, err := os.Create(memProfileFilename)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
			logf("could not create memory profile: %v\n", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write to memory profile: ", err)
			logf("could not write to memory profile: %v\n", err)
		}
	}
	if cpuProfileFilename != "" {
		pprof.StopCPUProfile()
		cpuProfileFile.Close()
	}
}
