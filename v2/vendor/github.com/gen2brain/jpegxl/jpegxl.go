// Package jpegxl implements an JPEG XL image decoder based on libjxl compiled to WASM.
package jpegxl

import (
	"errors"
	"image"
	"image/draw"
	"io"
)

// JXL represents the possibly multiple images stored in a JXL file.
type JXL struct {
	// Decoded images, NRGBA or NRGBA64.
	Image []image.Image
	// Delay times, one per frame, in seconds of a tick.
	Delay []int
}

// DefaultQuality is the default quality encoding parameter.
const DefaultQuality = 75

// DefaultEffort is the default effort encoding parameter.
const DefaultEffort = 7

// Options are the encoding parameters.
type Options struct {
	// Quality in the range [0,100]. Quality of 100 enables lossless. Default is 75.
	Quality int
	// Effort in the range [1,10]. Sets encoder effort/speed level without affecting decoding speed. Default is 7.
	Effort int
}

// Errors .
var (
	ErrMemRead  = errors.New("jpegxl: mem read failed")
	ErrMemWrite = errors.New("jpegxl: mem write failed")
	ErrDecode   = errors.New("jpegxl: decode failed")
	ErrEncode   = errors.New("jpegxl: encode failed")
)

// Decode reads a JPEG XL image from r and returns it as an image.Image.
func Decode(r io.Reader) (image.Image, error) {
	var err error
	var ret *JXL

	if dynamic {
		ret, _, err = decodeDynamic(r, false, false)
		if err != nil {
			return nil, err
		}
	} else {
		ret, _, err = decode(r, false, false)
		if err != nil {
			return nil, err
		}
	}

	return ret.Image[0], nil
}

// DecodeConfig returns the color model and dimensions of a JPEG XL image without decoding the entire image.
func DecodeConfig(r io.Reader) (image.Config, error) {
	var err error
	var cfg image.Config

	if dynamic {
		_, cfg, err = decodeDynamic(r, true, false)
		if err != nil {
			return image.Config{}, err
		}
	} else {
		_, cfg, err = decode(r, true, false)
		if err != nil {
			return image.Config{}, err
		}
	}

	return cfg, nil
}

// DecodeAll reads a JPEG XL image from r and returns the sequential frames and timing information.
func DecodeAll(r io.Reader) (*JXL, error) {
	var err error
	var ret *JXL

	if dynamic {
		ret, _, err = decodeDynamic(r, false, true)
		if err != nil {
			return nil, err
		}
	} else {
		ret, _, err = decode(r, false, true)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

// Encode writes the image m to w with the given options.
func Encode(w io.Writer, m image.Image, o ...Options) error {
	effort := DefaultEffort
	quality := DefaultQuality

	if o != nil {
		opt := o[0]
		effort = opt.Effort
		quality = opt.Quality

		if effort <= 0 {
			effort = DefaultEffort
		} else if effort > 10 {
			effort = 10
		}

		if quality <= 0 {
			quality = DefaultQuality
		} else if quality > 100 {
			quality = 100
		}
	}

	if dynamic {
		err := encodeDynamic(w, m, quality, effort)
		if err != nil {
			return err
		}
	} else {
		err := encode(w, m, quality, effort)
		if err != nil {
			return err
		}
	}

	return nil
}

// Dynamic returns error (if there was any) during opening dynamic/shared library.
func Dynamic() error {
	return dynamicErr
}

// InitDecoder initializes wazero runtime and compiles the module.
// This function does nothing if a dynamic/shared library is used and Dynamic() returns nil.
// There is no need to explicitly call this function, first Decode will initialize the runtime.
func InitDecoder() {
	if dynamic && dynamicErr == nil {
		return
	}

	initDecoderOnce()
}

// InitEncoder initializes wazero runtime and compiles the module.
// This function does nothing if a dynamic/shared library is used and Dynamic() returns nil.
// There is no need to explicitly call this function, first Encode will initialize the runtime.
func InitEncoder() {
	if dynamic && dynamicErr == nil {
		return
	}

	initEncoderOnce()
}

func imageToNRGBA(src image.Image) *image.NRGBA {
	if dst, ok := src.(*image.NRGBA); ok {
		return dst
	}

	b := src.Bounds()
	dst := image.NewNRGBA(b)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)

	return dst
}

func init() {
	image.RegisterFormat("jxl", "????JXL", Decode, DecodeConfig)
	image.RegisterFormat("jxl", "\xff\x0a", Decode, DecodeConfig)
}
