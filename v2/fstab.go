package main

import (
	"bytes"
)

// formatFstab can format the contents of /etc/fstab files. The suggested number of spaces is 2.
func formatFstab(data []byte, spaces int) []byte {
	var (
		buf       bytes.Buffer
		nl        = []byte{'\n'}
		longest   = make(map[int]int) // The longest length of a field, for each field index
		byteLines = bytes.Split(data, nl)
	)

	// Find the longest field length for each field on each line
	for _, line := range byteLines {
		trimmedLine := bytes.TrimSpace(line)
		if len(trimmedLine) == 0 || bytes.HasPrefix(trimmedLine, []byte{'#'}) {
			continue
		}
		// Find the longest field length for each field
		for i, field := range bytes.Fields(trimmedLine) {
			fieldLength := len(string(field))
			if val, ok := longest[i]; ok {
				if fieldLength > val {
					longest[i] = fieldLength
				}
			} else {
				longest[i] = fieldLength
			}
		}
	}

	// Format the lines nicely
	for i, line := range byteLines {

		// Get the previous line, if possible
		var prevLineTrimmed []byte
		if (i - 1) > 0 {
			prevLineTrimmed = bytes.TrimSpace(byteLines[i-1])
		}

		// Get the current line
		thisLineTrimmed := bytes.TrimSpace(line)

		// Get the next line, if possible
		var nextLineTrimmed []byte
		if (i + 1) < len(byteLines) {
			nextLineTrimmed = bytes.TrimSpace(byteLines[i+1])
		}

		// Get the next next line, if possible
		var nextNextLineTrimmed []byte
		if (i + 2) < len(byteLines) {
			nextNextLineTrimmed = bytes.TrimSpace(byteLines[i+2])
		}

		// Get the next next next line, if possible
		var nextNextNextLineTrimmed []byte
		if (i + 3) < len(byteLines) {
			nextNextNextLineTrimmed = bytes.TrimSpace(byteLines[i+3])
		}

		// Gather stats for if the lines are blank
		prevLineIsBlank := len(prevLineTrimmed) == 0
		thisLineIsBlank := len(thisLineTrimmed) == 0
		nextLineIsBlank := len(nextLineTrimmed) == 0
		nextNextLineIsBlank := len(nextNextLineTrimmed) == 0
		nextNextNextLineIsBlank := len(nextNextNextLineTrimmed) == 0

		// Gether stats for if the lines are comments
		prevLineIsComment := bytes.HasPrefix(prevLineTrimmed, []byte{'#'})
		thisLineIsComment := bytes.HasPrefix(thisLineTrimmed, []byte{'#'})
		nextLineIsComment := bytes.HasPrefix(nextLineTrimmed, []byte{'#'})
		nextNextLineIsComment := bytes.HasPrefix(nextNextLineTrimmed, []byte{'#'})
		nextNextNextLineIsComment := bytes.HasPrefix(nextNextNextLineTrimmed, []byte{'#'})

		// Gether stats for if the lines have content
		prevLineIsContent := !prevLineIsBlank && !prevLineIsComment
		nextLineIsContent := !nextLineIsBlank && !nextLineIsComment
		nextNextLineIsContent := !nextNextLineIsBlank && !nextNextLineIsComment
		nextNextNextLineIsContent := !nextNextNextLineIsBlank && !nextNextNextLineIsComment

		if thisLineIsBlank {
			if prevLineIsContent && nextLineIsComment {
				buf.Write(nl)
			} else if nextLineIsComment && nextNextLineIsContent {
				buf.Write(nl)
			} else if prevLineIsComment && nextLineIsComment && (nextNextLineIsContent || (nextNextLineIsBlank && nextNextNextLineIsContent)) {
				buf.Write(nl)
			}
		} else if thisLineIsComment {
			if prevLineIsComment && nextLineIsContent {
				buf.Write(nl)
			} else if prevLineIsBlank && nextLineIsContent {
				buf.Write(nl)
			} else if prevLineIsContent && nextLineIsContent {
				buf.Write(nl)
			}
			buf.Write(thisLineTrimmed)
			buf.Write(nl)
		} else { // This line has contents, format the fields
			for i, field := range bytes.Fields(thisLineTrimmed) {
				fieldLength := len(string(field))
				padCount := spaces // Space between the fields if all fields have equal length
				if longest[i] > fieldLength {
					padCount += longest[i] - fieldLength
				}
				buf.Write(field)
				if padCount > 0 {
					buf.Write(bytes.Repeat([]byte{' '}, padCount))
				}
			}
			buf.Write(nl)
		}
	}
	// TODO: Find out why double blank lines are sometimes inserted below, when pressing ctrl-w twice,
	//       to avoid using bytes.ReplaceAll.
	return bytes.ReplaceAll(buf.Bytes(), []byte{'\n', '\n', '\n'}, []byte{'\n', '\n'})
}
