package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// logf, for quick "printf-style" debugging
// Will call log.Fatalln if there are problems!
func logf(format string, args ...interface{}) {
	logFilename := filepath.Join(tempDir, "o.log")
	if isDarwin() {
		logFilename = "/tmp/o.log"
	}
	err := flogf(logFilename, format, args...)
	if err != nil {
		log.Fatalln(err)
	}
}

// Silence the "logf is unused" message by staticcheck
var _ = logf

// flogf, for logging to a file with a fprintf-style function
func flogf(logfile, format string, args ...interface{}) error {
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		f, err = os.Create(logfile)
		if err != nil {
			return err
		}
	}
	_, err = f.WriteString(fmt.Sprintf(format, args...))
	if err != nil {
		return err
	}
	err = f.Sync()
	if err != nil {
		return err
	}
	return f.Close()
}

// // pplogf, for pretty print logging to a file, using
// // import "github.com/k0kubun/pp/v3"
// func pplogf(format string, args ...interface{}) error {
// 	logFilename := filepath.Join(tempDir, "o.log")
// 	f, err := os.OpenFile(logFilename, os.O_APPEND|os.O_WRONLY, 0644)
// 	if err != nil {
// 		f, err = os.Create(logFilename)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	prettyPrinter := pp.New()
// 	prettyPrinter.SetOutput(f)
// 	prettyPrinter.Printf(format, args...)
// 	err = f.Sync()
// 	if err != nil {
// 		return err
// 	}
// 	return f.Close()
// }
