package colour

import (
	"github.com/kpfaulkner/jxl-go/jxlio"
)

type CustomXY struct {
	CIEXY
}

func NewCustomXY(reader jxlio.BitReader) (*CustomXY, error) {
	cxy := &CustomXY{}

	ciexy, err := cxy.readCustom(reader)
	if err != nil {
		return nil, err
	}
	cxy.CIEXY = *ciexy
	return cxy, nil
}

func (cxy *CustomXY) readCustom(reader jxlio.BitReader) (*CIEXY, error) {
	var x float32
	if ux, err := reader.ReadU32(0, 19, 524288, 19, 1048576, 20, 2097152, 21); err != nil {
		return nil, err
	} else {
		x = float32(jxlio.UnpackSigned(ux)) * 1e-6
	}

	var y float32
	if uy, err := reader.ReadU32(0, 19, 524288, 19, 1048576, 20, 2097152, 21); err != nil {
		return nil, err
	} else {
		y = float32(jxlio.UnpackSigned(uy)) * 1e-6
	}

	return NewCIEXY(x, y), nil
}
