package bundle

import "github.com/kpfaulkner/jxl-go/jxlio"

type BitDepthHeader struct {
	BitsPerSample    uint32
	ExpBits          uint32
	UsesFloatSamples bool
}

func NewBitDepthHeader() *BitDepthHeader {
	bh := &BitDepthHeader{}
	bh.UsesFloatSamples = false
	bh.BitsPerSample = 8
	bh.ExpBits = 0
	return bh
}

func NewBitDepthHeaderWithReader(reader jxlio.BitReader) (*BitDepthHeader, error) {
	bh := &BitDepthHeader{}
	var err error
	if bh.UsesFloatSamples, err = reader.ReadBool(); err != nil {
		return nil, err
	}

	if bh.UsesFloatSamples {
		if bh.BitsPerSample, err = reader.ReadU32(32, 0, 16, 0, 24, 0, 1, 6); err != nil {
			return nil, err
		}
		if expBits, err := reader.ReadBits(4); err != nil {
			return nil, err
		} else {
			bh.ExpBits = 1 + uint32(expBits)
		}
	} else {
		if bh.BitsPerSample, err = reader.ReadU32(8, 0, 10, 0, 12, 0, 1, 6); err != nil {
			return nil, err
		}

		bh.ExpBits = 0
	}
	return bh, nil
}
