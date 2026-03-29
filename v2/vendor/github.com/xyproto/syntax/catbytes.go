package syntax

import (
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

// CatBytes highlights sourceCodeData and writes it to stdout via the given TextOutput.
func CatBytes(sourceCodeData []byte, o *vt.TextOutput) error {
	detectedMode := mode.SimpleDetectBytes(sourceCodeData)
	taggedTextBytes, err := AsText(sourceCodeData, detectedMode)
	if err == nil {
		o.OutputTags(string(taggedTextBytes))
	}
	return err
}
