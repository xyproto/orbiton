package main

import (
	"bufio"
	"io"

	"github.com/xyproto/stringpainter"
)

func yetAnotherLogColorizer(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	// Increase the maximum line length to 1MB
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	// Colorize the input data
	for scanner.Scan() {
		w.Write([]byte((stringpainter.Colorize(scanner.Text())) + "\n"))
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
