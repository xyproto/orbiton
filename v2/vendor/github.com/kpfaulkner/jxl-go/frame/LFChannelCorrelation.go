package frame

import "github.com/kpfaulkner/jxl-go/jxlio"

type LFChannelCorrelation struct {
	colorFactor      uint32
	baseCorrelationX float32
	baseCorrelationB float32
	xFactorLF        uint32
	bFactorLF        uint32
}

func NewLFChannelCorrelation() (*LFChannelCorrelation, error) {
	return NewLFChannelCorrelationWithReaderAndDefault(nil, true)
}

func NewLFChannelCorrelationWithReaderAndDefault(reader jxlio.BitReader, allDefault bool) (*LFChannelCorrelation, error) {
	lf := &LFChannelCorrelation{}

	if allDefault {
		lf.colorFactor = 84
		lf.baseCorrelationX = 0.0
		lf.baseCorrelationB = 1.0
		lf.xFactorLF = 128
		lf.bFactorLF = 128
	} else {
		var err error
		if lf.colorFactor, err = reader.ReadU32(84, 0, 256, 0, 2, 8, 258, 16); err != nil {
			return nil, err
		}
		if lf.baseCorrelationX, err = reader.ReadF16(); err != nil {
			return nil, err
		}
		if lf.baseCorrelationB, err = reader.ReadF16(); err != nil {
			return nil, err
		}

		bits := uint64(0)
		if bits, err = reader.ReadBits(8); err != nil {
			return nil, err
		}
		lf.xFactorLF = uint32(bits)
		if bits, err = reader.ReadBits(8); err != nil {
			return nil, err
		}
		lf.bFactorLF = uint32(bits)
	}
	return lf, nil
}

func NewLFChannelCorrelationWithReader(reader jxlio.BitReader) (*LFChannelCorrelation, error) {
	var allDefault bool
	var err error
	if allDefault, err = reader.ReadBool(); err != nil {
		return nil, err
	}
	return NewLFChannelCorrelationWithReaderAndDefault(reader, allDefault)
}
