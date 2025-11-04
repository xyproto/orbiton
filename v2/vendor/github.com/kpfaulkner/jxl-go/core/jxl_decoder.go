package core

import (
	"io"

	"github.com/kpfaulkner/jxl-go/bundle"
	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/options"
)

// JXLDecoder decodes the JXL image
type JXLDecoder struct {

	// input Stream
	in io.ReadSeeker

	// decoder
	decoder *JXLCodestreamDecoder
}

func NewJXLDecoder(in io.ReadSeeker, opts *options.JXLOptions) *JXLDecoder {
	jxl := &JXLDecoder{
		in: in,
	}

	br := jxlio.NewBitStreamReader(in)

	// if nil options, then create one
	if opts == nil {
		opts = options.NewJXLOptions(nil)
	}
	jxl.decoder = NewJXLCodestreamDecoder(br, opts)
	return jxl
}

func (jxl *JXLDecoder) Decode() (*JXLImage, error) {

	jxlImage, err := jxl.decoder.decode()
	if err != nil {
		return nil, err
	}

	return jxlImage, nil
}

func (jxl *JXLDecoder) GetImageHeader() (*bundle.ImageHeader, error) {

	header, err := jxl.decoder.GetImageHeader()
	if err != nil {
		return nil, err
	}

	return header, nil

}
