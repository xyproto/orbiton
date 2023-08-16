package ico

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"

	bmp "github.com/jsummers/gobmp"
)

const pngHeader = "\x89PNG\r\n\x1a\n"

// A FormatError reports that the input is not a valid ICO.
type FormatError string

func (e FormatError) Error() string { return "invalid ICO format: " + string(e) }

// If the io.Reader does not also have ReadByte, then decode will introduce its own buffering.
type reader interface {
	io.Reader
	io.ByteReader
}

type decoder struct {
	r     reader
	num   uint16
	dir   []entry
	image []image.Image
	cfg   image.Config
}

func (d *decoder) decode(r io.Reader, configOnly bool) error {
	// Add buffering if r does not provide ReadByte.
	if rr, ok := r.(reader); ok {
		d.r = rr
	} else {
		d.r = bufio.NewReader(r)
	}

	if err := d.readHeader(); err != nil {
		return err
	}
	if err := d.readImageDir(configOnly); err != nil {
		return err
	}
	if configOnly {
		cfg, err := d.parseConfig(d.dir[0])
		if err != nil {
			return err
		}
		d.cfg = cfg
	} else {
		d.image = make([]image.Image, d.num)
		for i, entry := range d.dir {
			img, err := d.parseImage(entry)
			if err != nil {
				return err
			}
			d.image[i] = img
		}
	}
	return nil
}

func (d *decoder) readHeader() error {
	var first, second uint16
	binary.Read(d.r, binary.LittleEndian, &first)
	binary.Read(d.r, binary.LittleEndian, &second)
	binary.Read(d.r, binary.LittleEndian, &d.num)
	if first != 0 {
		return FormatError(fmt.Sprintf("first byte is %d instead of 0", first))
	}
	if second != 1 {
		return FormatError(fmt.Sprintf("second byte is %d instead of 1", second))
	}
	return nil
}

func (d *decoder) readImageDir(configOnly bool) error {
	n := int(d.num)
	if configOnly {
		n = 1
	}
	for i := 0; i < n; i++ {
		var e entry
		err := binary.Read(d.r, binary.LittleEndian, &e)
		if err != nil {
			return err
		}
		d.dir = append(d.dir, e)
	}
	return nil
}

func (d *decoder) parseImage(e entry) (image.Image, error) {
	data := make([]byte, e.Size)
	io.ReadFull(d.r, data)

	// Check if the image is a PNG by the first 8 bytes of the image data
	if string(data[:len(pngHeader)]) == pngHeader {
		return png.Decode(bytes.NewReader(data))
	}

	// Decode as BMP instead
	bmpBytes, maskBytes, offset, err := d.setupBMP(e, data)
	if err != nil {
		return nil, err
	}

	src, err := bmp.Decode(bytes.NewReader(bmpBytes))
	if err != nil {
		return nil, err
	}

	bnd := src.Bounds()
	mask := image.NewAlpha(image.Rect(0, 0, bnd.Dx(), bnd.Dy()))
	dst := image.NewNRGBA(image.Rect(0, 0, bnd.Dx(), bnd.Dy()))
	//draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	//Fill in mask from the ICO file's AND mask data
	rowSize := ((int(e.Width) + 31) / 32) * 4
	b := make([]byte, 4)
	_, _ = offset, b
	for r := 0; r < int(e.Height); r++ {
		for c := 0; c < int(e.Width); c++ {
			_, _ = maskBytes, rowSize
			if len(maskBytes) > 0 {
				alpha := (maskBytes[r*rowSize+c/8] >> (1 * (7 - uint(c)%8))) & 0x01
				if alpha != 1 {
					mask.SetAlpha(c, int(e.Height)-r-1, color.Alpha{255})
				}

			}
			// 32 bit bmps do hacky things with an alpha channel, it's included as the 4th byte of the colors
			if e.Bits == 32 {
				imageRowSize := ((int(e.Bits)*int(e.Width) + 31) / 32) * 4
				io.ReadFull(bytes.NewReader(bmpBytes[offset+r*imageRowSize+c*4:]), b)
				mask.SetAlpha(c, int(e.Height)-r-1, color.Alpha{b[3]})
			}
		}
	}
	draw.DrawMask(dst, dst.Bounds(), src, bnd.Min, mask, bnd.Min, draw.Src)

	return dst, nil
}

func (d *decoder) parseConfig(e entry) (cfg image.Config, err error) {
	tmp := make([]byte, e.Size)
	n, err := io.ReadFull(d.r, tmp)
	if n != int(e.Size) {
		return cfg, fmt.Errorf("Only %d of %d bytes read.", n, e.Size)
	}
	if err != nil {
		return cfg, err
	}

	cfg, err = png.DecodeConfig(bytes.NewReader(tmp))
	if err != nil {
		tmp, _, _, _ = d.setupBMP(e, tmp)
		cfg, err = bmp.DecodeConfig(bytes.NewReader(tmp))
	}
	return cfg, err
}

func (d *decoder) setupBMP(e entry, data []byte) ([]byte, []byte, int, error) {
	// Ico files are made up of a XOR mask and an AND mask
	// The XOR mask is the image itself, while the AND mask is a 1 bit-per-pixel alpha channel.
	// setupBMP returns the image as a BMP format byte array, and the mask as a (1bpp) pixel array

	// calculate image sizes
	// See wikipedia en.wikipedia.org/wiki/BMP_file_format
	var imageSize, maskSize int
	if int(e.Size) < len(data) {
		imageSize = int(e.Size)
	} else {
		imageSize = len(data)
	}
	if e.Bits != 32 {
		rowSize := (1 * (int(e.Width) + 31) / 32) * 4
		maskSize = rowSize * int(e.Height)
		imageSize -= maskSize
	}

	img := make([]byte, 14+imageSize)
	mask := make([]byte, maskSize)

	var n int
	// Read in image
	n = copy(img[14:], data[:imageSize])
	if n != imageSize {
		return nil, nil, 0, FormatError(fmt.Sprintf("only %d of %d bytes read.", n, imageSize))
	}
	// Read in mask
	n = copy(mask, data[imageSize:])
	if n != maskSize {
		return nil, nil, 0, FormatError(fmt.Sprintf("only %d of %d bytes read.", n, maskSize))
	}

	var dibSize, w, h uint32
	binary.Read(bytes.NewReader(img[14:14+4]), binary.LittleEndian, &dibSize)
	binary.Read(bytes.NewReader(img[14+4:14+8]), binary.LittleEndian, &w)
	binary.Read(bytes.NewReader(img[14+8:14+12]), binary.LittleEndian, &h)

	if h > w {
		binary.LittleEndian.PutUint32(img[14+8:14+12], h/2)
	}

	// Magic number
	copy(img[0:2], "\x42\x4D")

	// File size
	binary.LittleEndian.PutUint32(img[2:6], uint32(imageSize+14))

	// Calculate offset into image data
	var numColors uint32
	binary.Read(bytes.NewReader(img[14+32:14+36]), binary.LittleEndian, &numColors)

	var bpp uint16
	binary.Read(bytes.NewReader(img[14+14:14+16]), binary.LittleEndian, &bpp)
	e.Bits = bpp

	var size uint32
	binary.Read(bytes.NewReader(img[14+20:14+24]), binary.LittleEndian, &size)
	e.Size = size

	switch int(e.Bits) {
	case 1, 2, 4, 8:
		x := uint32(1 << e.Bits)
		if numColors == 0 || numColors > x {
			numColors = x
		}
	default:
		numColors = 0
	}

	var numColorsSize uint32
	switch int(dibSize) {
	case 12, 64:
		numColorsSize = numColors * 3
	default:
		numColorsSize = numColors * 4
	}

	var offset uint32
	offset = 14 + dibSize + numColorsSize

	if dibSize > 40 {
		var iccSize uint32
		binary.Read(bytes.NewReader(img[14+dibSize-8:14+dibSize-4]), binary.LittleEndian, &iccSize)
		offset += iccSize
	}
	binary.LittleEndian.PutUint32(img[10:14], offset)

	return img, mask, int(offset), nil
}

func Decode(r io.Reader) (image.Image, error) {
	var d decoder
	if err := d.decode(r, false); err != nil {
		return nil, err
	}
	return d.image[0], nil
}

func DecodeAll(r io.Reader) (*ICO, error) {
	var d decoder
	if err := d.decode(r, false); err != nil {
		return nil, err
	}
	ico := &ICO{
		Num:   int(d.num),
		Image: d.image,
	}
	return ico, nil
}

func DecodeConfig(r io.Reader) (image.Config, error) {
	var d decoder
	if err := d.decode(r, true); err != nil {
		return image.Config{}, err
	}
	return d.cfg, nil
}

func init() {
	image.RegisterFormat("ico", "", Decode, DecodeConfig)
}
