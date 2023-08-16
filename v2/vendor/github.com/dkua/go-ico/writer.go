package ico

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"io"

	bmp "github.com/jsummers/gobmp"
)

type dib struct {
	DibSize     uint32
	Width       uint32
	Height      uint32
	Planes      uint16
	Bpp         uint16
	Compression uint32
	XPMM        uint32
	YPMM        uint32
	Colors      uint32
}

type encoder struct {
	w     io.Writer
	m     image.Image
	entry bytes.Buffer
}

func (e *encoder) writeHeader(num int) (err error) {
	// Write the required zero byte
	if err = binary.Write(e.w, binary.LittleEndian, uint16(0)); err != nil {
		return err
	}
	// Write the required one byte
	if err = binary.Write(e.w, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	// Write the number of images
	if err = binary.Write(e.w, binary.LittleEndian, uint16(num)); err != nil {
		return err
	}
	return nil
}

func getSize(n []byte) (uint8, error) {
	var orig uint32
	var size uint8

	if err := binary.Read(bytes.NewReader(n), binary.LittleEndian, &orig); err != nil {
		return size, err
	}
	switch {
	case orig == 256:
		size = uint8(0)
	case orig > 256:
		return size, fmt.Errorf("width too big")
	default:
		size = uint8(orig)
	}
	return size, nil
}

func (e *encoder) writeEntry(b []byte, size, offset int) (err error) {
	width, err := getSize(b[4:8])
	if err != nil {
		return err
	}
	if err = binary.Write(e.w, binary.LittleEndian, width); err != nil {
		return err
	}

	height, err := getSize(b[8:12])
	if err != nil {
		return err
	}
	if err = binary.Write(e.w, binary.LittleEndian, height); err != nil {
		return err
	}

	var palette uint16
	if err = binary.Read(bytes.NewReader(b[32:36]), binary.LittleEndian, &palette); err != nil {
		return err
	}
	fmt.Println("palette", palette, uint8(palette))
	if err = binary.Write(e.w, binary.LittleEndian, uint8(palette)); err != nil {
		return err
	}

	if err = binary.Write(e.w, binary.LittleEndian, uint8(0)); err != nil {
		return err
	}

	var planes uint16
	if err = binary.Read(bytes.NewReader(b[12:14]), binary.LittleEndian, &planes); err != nil {
		return err
	}
	fmt.Println("planes", planes)
	if err = binary.Write(e.w, binary.LittleEndian, planes); err != nil {
		return err
	}

	var bpp uint16
	if err = binary.Read(bytes.NewReader(b[14:16]), binary.LittleEndian, &bpp); err != nil {
		return err
	}
	fmt.Println("bpp", bpp)
	if err = binary.Write(e.w, binary.LittleEndian, bpp); err != nil {
		return err
	}

	if err = binary.Write(e.w, binary.LittleEndian, uint32(size)); err != nil {
		return err
	}
	fmt.Println("size", size, uint32(size))
	if err = binary.Write(e.w, binary.LittleEndian, uint32(offset)); err != nil {
		return err
	}
	fmt.Println("offset", offset, uint32(offset))

	return nil
}

func (e *encoder) writeImage(m []byte) (err error) {
	n, err := e.w.Write(m)
	if n != len(m) {
		return fmt.Errorf("not enough bytes written")
	}
	if err != nil {
		return err
	}
	return nil
}

func Encode(w io.Writer, m image.Image) error {
	var err error

	e := new(encoder)
	e.w = w
	e.m = m

	if err = e.writeHeader(1); err != nil {
		return err
	}

	var buf bytes.Buffer
	wr := bufio.NewWriter(&buf)

	if err = bmp.Encode(wr, e.m); err != nil {
		return err
	}
	wr.Flush()

	if n := buf.Next(14); len(n) != 14 {
		return fmt.Errorf("image not proper size")
	} else {
		fmt.Println(n)
		var fSize uint32
		binary.Read(bytes.NewReader(n[2:6]), binary.LittleEndian, &fSize)
		fmt.Println("fSize", fSize)
	}
	b := buf.Bytes()

	if err = e.writeEntry(b[:40], len(b), 22); err != nil {
		return err
	}
	if err = e.writeImage(b); err != nil {
		return err
	}
	return nil
}
