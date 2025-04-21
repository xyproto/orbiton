package main

import (
	"github.com/xyproto/mode"
	"github.com/xyproto/textoutput"
)

func CatBytes(sourceCodeData []byte, o *textoutput.TextOutput) error {
	detectedMode := mode.SimpleDetectBytes(sourceCodeData)
	taggedTextBytes, err := AsText(sourceCodeData, detectedMode)
	if err == nil { // success
		o.OutputTags(string(taggedTextBytes))
	}
	return err // can be nil
}
