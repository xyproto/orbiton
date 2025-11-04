package core

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"io"

	"github.com/kpfaulkner/jxl-go/colour"
)

// WritePNG instead of using standard golang image/png package since we need
// to write out ICC Profile which doesn't seem to be supported by the standard package.
func WritePNG(jxlImage *JXLImage, output io.Writer) error {

	// PNG header
	header := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	output.Write(header)

	if err := writeIHDR(jxlImage, output); err != nil {
		return err
	}

	if jxlImage.iccProfile != nil || len(jxlImage.iccProfile) != 0 {
		if err := writeICCP(jxlImage, output); err != nil {
			return err
		}
	} else {
		// if we have an ICC profile then write it out
		if err := writeSRGB(jxlImage, output); err != nil {
			return err
		}
	}

	if err := writeIDAT(jxlImage, output); err != nil {
		return err
	}
	output.Write([]byte{0, 0, 0, 0})
	output.Write([]byte{0x49, 0x45, 0x4E, 0x44})
	output.Write([]byte{0xAE, 0x42, 0x60, 0x82})
	return nil
}

func writeICCP(image *JXLImage, output io.Writer) error {

	var buf bytes.Buffer
	//output.Write([]byte{0x69, 0x43, 0x43, 0x50})
	buf.Write([]byte{0x69, 0x43, 0x43, 0x50})
	buf.Write([]byte("jxlatte")) // using jxlatte just to compare files
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)

	var iccProfile []int8
	for i := 0; i < len(image.iccProfile); i++ {
		iccProfile = append(iccProfile, int8(i))
	}

	var compressedICC bytes.Buffer
	w, err := zlib.NewWriterLevel(&compressedICC, 1)
	if err != nil {
		return err
	}
	w.Write(image.iccProfile)
	w.Flush()
	w.Close()

	b := compressedICC.Bytes()
	buf.Write(b)

	rawBytes := buf.Bytes()
	buf2 := make([]byte, 4)
	binary.BigEndian.PutUint32(buf2, uint32(len(rawBytes))-4)
	output.Write(buf2)
	output.Write(rawBytes)

	checksum := crc32.ChecksumIEEE(rawBytes)
	binary.BigEndian.PutUint32(buf2, checksum)
	output.Write(buf2)
	return nil
}

func writeSRGB(image *JXLImage, output io.Writer) error {

	var buf bytes.Buffer
	//output.Write([]byte{0x69, 0x43, 0x43, 0x50})
	buf.Write([]byte{0x00, 0x00, 0x00, 0x01})
	buf.Write([]byte{0x73, 0x52, 0x47, 0x42}) // using jxlatte just to compare files
	buf.WriteByte(0x01)
	buf.Write([]byte{0xD9, 0xC9, 0x2C, 0x7F})

	rawBytes := buf.Bytes()
	//buf2 := make([]byte, 4)
	//binary.BigEndian.PutUint32(buf2, uint32(len(rawBytes))-4)
	//output.Write(buf2)
	output.Write(rawBytes)

	return nil
}

func writeIHDR(jxlImage *JXLImage, output io.Writer) error {

	// colourmode is 6 if include alpha...  otherwise 2
	colourMode := byte(2)

	if jxlImage.ColorEncoding == colour.CE_GRAY {
		if jxlImage.alphaIndex >= 0 {
			colourMode = 4
		} else {
			colourMode = 0
		}
	} else {
		if jxlImage.alphaIndex >= 0 {
			colourMode = 6
		} else {
			colourMode = 2
		}
	}
	var bitDepth int32
	if jxlImage.imageHeader.BitDepth.BitsPerSample > 8 {
		bitDepth = 16
	} else {
		bitDepth = 8
	}

	ihdr := make([]byte, 17)
	copy(ihdr[:4], []byte{'I', 'H', 'D', 'R'})
	binary.BigEndian.PutUint32(ihdr[4:], jxlImage.Width)
	binary.BigEndian.PutUint32(ihdr[8:], jxlImage.Height)
	ihdr[12] = byte(bitDepth)
	ihdr[13] = colourMode
	ihdr[14] = 0
	ihdr[15] = 0
	ihdr[16] = 0

	sizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeBytes, uint32(len(ihdr)-4))
	if _, err := output.Write(sizeBytes); err != nil {
		return err
	}

	if _, err := output.Write(ihdr); err != nil {
		return err
	}

	checksum := crc32.ChecksumIEEE(ihdr)
	checkSumBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(checkSumBytes, checksum)
	if _, err := output.Write(checkSumBytes); err != nil {
		return err
	}

	return nil
}

func writeIDAT(jxlImage *JXLImage, output io.Writer) error {

	var buf bytes.Buffer
	buf.Write([]byte("IDAT"))

	var compressedBytes bytes.Buffer
	w, err := zlib.NewWriterLevel(&compressedBytes, zlib.DefaultCompression)
	if err != nil {
		return err
	}

	var bitDepth int32
	if jxlImage.imageHeader.BitDepth.BitsPerSample > 8 {
		bitDepth = 16
	} else {
		bitDepth = 8
	}

	maxValue := int32(^(^0 << bitDepth))

	if err = jxlImage.Buffer[0].CastToIntIfFloat(maxValue); err != nil {
		return err
	}

	if len(jxlImage.Buffer) > 1 {
		if err = jxlImage.Buffer[1].CastToIntIfFloat(maxValue); err != nil {
			return err
		}
	}

	if len(jxlImage.Buffer) > 2 {
		if err = jxlImage.Buffer[2].CastToIntIfFloat(maxValue); err != nil {
			return err
		}
	}

	if jxlImage.HasAlpha() {
		if err = jxlImage.Buffer[3].CastToIntIfFloat(maxValue); err != nil {
			return err
		}
	}

	for c := 0; c < len(jxlImage.Buffer); c++ {
		if jxlImage.Buffer[c].IsInt() && jxlImage.bitDepths[c] == uint32(bitDepth) {
			if err := jxlImage.Buffer[c].Clamp(maxValue); err != nil {
				return err
			}
		} else {
			if err := jxlImage.Buffer[c].CastToIntIfFloat(maxValue); err != nil {
				return err
			}
		}
	}
	for y := uint32(0); y < jxlImage.Height; y++ {
		w.Write([]byte{0})
		for x := uint32(0); x < jxlImage.Width; x++ {

			// FIXME(kpfaulkner) remove 3 assumption
			for c := 0; c < jxlImage.imageHeader.GetColourChannelCount(); c++ {
				dat := jxlImage.Buffer[c].IntBuffer[y][x]
				if jxlImage.bitDepths[c] == 8 {
					w.Write([]byte{byte(dat)})
				} else {
					byte1 := dat & 0xFF
					byte2 := dat & 0xFF00
					byte2 >>= 8
					w.Write([]byte{byte(byte2), byte(byte1)})
				}

			}
			if jxlImage.HasAlpha() {
				dat := jxlImage.Buffer[3].IntBuffer[y][x]
				if jxlImage.bitDepths[3] == 8 {
					w.Write([]byte{byte(dat)})
				} else {
					byte1 := dat & 0xFF
					byte2 := dat & 0xFF00
					byte2 >>= 8

					w.Write([]byte{byte(byte2), byte(byte1)})
				}
			}
		}
	}
	w.Close()

	buf.Write(compressedBytes.Bytes())
	bb := buf.Bytes()
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(len(bb))-4)
	output.Write(b)
	output.Write(bb)
	checksum := crc32.ChecksumIEEE(bb)
	binary.BigEndian.PutUint32(b, checksum)
	output.Write(b)

	return nil
}
