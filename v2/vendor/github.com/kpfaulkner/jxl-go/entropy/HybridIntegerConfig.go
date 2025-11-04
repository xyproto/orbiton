package entropy

import (
	"errors"

	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

type HybridIntegerConfig struct {
	SplitExponent int32
	MsbInToken    int32
	LsbInToken    int32
}

func NewHybridIntegerConfig(splitExponent int32, msbInToken int32, lsbInToken int32) *HybridIntegerConfig {
	hic := &HybridIntegerConfig{}
	hic.SplitExponent = splitExponent
	hic.MsbInToken = msbInToken
	hic.LsbInToken = lsbInToken
	return hic
}

func NewHybridIntegerConfigWithReader(reader jxlio.BitReader, logAlphabetSize int32) (*HybridIntegerConfig, error) {
	hic := &HybridIntegerConfig{}
	if splitExp, err := reader.ReadBits(uint32(util.CeilLog1p(int64(logAlphabetSize)))); err != nil {
		return nil, err
	} else {
		hic.SplitExponent = int32(splitExp)
	}
	if hic.SplitExponent == logAlphabetSize {
		hic.MsbInToken = 0
		hic.LsbInToken = 0
		return hic, nil
	}
	var bits uint64
	var err error
	if bits, err = reader.ReadBits(uint32(util.CeilLog1p(int64(hic.SplitExponent)))); err != nil {
		return nil, err
	} else {
		hic.MsbInToken = int32(bits)
	}

	if hic.MsbInToken > hic.SplitExponent {
		return nil, errors.New("msbInToken is too large")
	}
	if bits, err = reader.ReadBits(uint32(util.CeilLog1p(int64(hic.SplitExponent - hic.MsbInToken)))); err != nil {
		return nil, err
	} else {
		hic.LsbInToken = int32(bits)
	}
	if hic.MsbInToken+hic.LsbInToken > hic.SplitExponent {
		return nil, errors.New("msbInToken + lsbInToken is too large")
	}
	return hic, nil
}
