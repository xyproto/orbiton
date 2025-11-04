package frame

import (
	"github.com/kpfaulkner/jxl-go/jxlio"
)

type BlendingInfo struct {
	Mode         uint32
	AlphaChannel uint32
	Clamp        bool
	Source       uint32
}

func NewBlendingInfo() *BlendingInfo {
	bi := &BlendingInfo{}
	bi.Mode = BLEND_REPLACE
	bi.AlphaChannel = 0
	bi.Clamp = false
	bi.Source = 0
	return bi
}

func NewBlendingInfoWithReader(reader jxlio.BitReader, extra bool, fullFrame bool) (*BlendingInfo, error) {

	bi := &BlendingInfo{}
	if mode, err := reader.ReadU32(0, 0, 1, 0, 2, 0, 3, 2); err != nil {
		return nil, err
	} else {
		bi.Mode = mode
	}

	if extra && (bi.Mode == BLEND_BLEND || bi.Mode == BLEND_MULADD) {
		if alphaChannel, err := reader.ReadU32(0, 0, 1, 0, 2, 0, 3, 3); err != nil {
			return nil, err
		} else {
			bi.AlphaChannel = alphaChannel
		}
	} else {
		bi.AlphaChannel = 0
	}

	var err error
	if extra && (bi.Mode == BLEND_BLEND ||
		bi.Mode == BLEND_MULT ||
		bi.Mode == BLEND_MULADD) {
		if bi.Clamp, err = reader.ReadBool(); err != nil {
			return nil, err
		}
	} else {
		bi.Clamp = false
	}

	if bi.Mode != BLEND_REPLACE || !fullFrame {
		if bits, err := reader.ReadBits(2); err != nil {
			return nil, err
		} else {
			bi.Source = uint32(bits)
		}
	} else {
		bi.Source = 0
	}

	return bi, nil
}
