package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/kpfaulkner/jxl-go/colour"
	image2 "github.com/kpfaulkner/jxl-go/image"
)

func WritePFM(jxlImage *JXLImage, output io.Writer) error {

	gray := jxlImage.ColorEncoding == colour.CE_GRAY
	width := jxlImage.Width
	height := jxlImage.Height

	pf := "Pf"
	if !gray {
		pf = "PF"
	}
	header := fmt.Sprintf("%s\n%d %d\n1.0\n", pf, width, height)
	output.Write([]byte(header))
	cCount := 1
	if !gray {
		cCount = 3
	}

	buffer2, err := jxlImage.getBuffer(false)
	if err != nil {
		return err
	}
	nb := make([]image2.ImageBuffer, len(buffer2))
	for c := 0; c < len(nb); c++ {
		if buffer2[c].IsInt() {
			panic("not implemented")
		} else {
			nb[c] = buffer2[c]
		}
	}

	if gray {
		cCount = 1
	}
	var buf bytes.Buffer

	for y := int32(height - 1); y >= 0; y-- {
		for x := int32(0); x < int32(width); x++ {
			for c := 0; c < cCount; c++ {
				err := binary.Write(&buf, binary.BigEndian, nb[c].FloatBuffer[y][x])
				if err != nil {
					fmt.Println("binary.Write failed:", err)
					return err
				}
			}
		}
	}
	output.Write(buf.Bytes())

	return nil
}
