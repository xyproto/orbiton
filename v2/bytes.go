package main

import "bytes"

func containsInTheFirstNLines(data []byte, n int, target []byte) bool {
	var (
		lineCount int
		lineStart int
		line      []byte
	)
	for i := range data {
		if data[i] == '\n' {
			line = data[lineStart:i]
			if bytes.Contains(line, target) {
				return true
			}
			lineCount++
			if lineCount >= n {
				return false
			}
			lineStart = i + 1
		}
	}
	if lineCount < n && lineStart < len(data) {
		line = data[lineStart:]
		if bytes.Contains(line, target) {
			return true
		}
	}
	return false
}
