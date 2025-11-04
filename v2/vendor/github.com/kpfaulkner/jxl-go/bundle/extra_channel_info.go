package bundle

import (
	"errors"

	"github.com/kpfaulkner/jxl-go/jxlio"
)

type ExtraChannelInfo struct {
	Name                       string
	EcType                     int32
	CfaIndex                   int32
	DimShift                   int32
	Red, Green, Blue, Solidity float32
	BitDepth                   BitDepthHeader
	AlphaAssociated            bool
}

func NewExtraChannelInfoWithReader(reader jxlio.BitReader) (*ExtraChannelInfo, error) {

	eci := &ExtraChannelInfo{}
	var err error
	var dAlpha bool
	if dAlpha, err = reader.ReadBool(); err != nil {
		return nil, err
	}
	if !dAlpha {
		if eci.EcType, err = reader.ReadEnum(); err != nil {
			return nil, err
		}
		if !ValidateExtraChannel(eci.EcType) {
			return nil, errors.New("Illegal extra channel type")
		}
		if bitDepth, err := NewBitDepthHeaderWithReader(reader); err != nil {
			return nil, err
		} else {
			eci.BitDepth = *bitDepth
		}

		if dimShift, err := reader.ReadU32(0, 0, 3, 0, 4, 0, 1, 3); err != nil {
			return nil, err
		} else {
			eci.DimShift = int32(dimShift)
		}
		var nameLen uint32
		var err error
		if nameLen, err = reader.ReadU32(0, 0, 0, 4, 16, 5, 48, 10); err != nil {
			return nil, err
		}

		nameBuffer := make([]byte, nameLen)
		for i := uint32(0); i < nameLen; i++ {
			if nb, err := reader.ReadBits(8); err != nil {
				return nil, err
			} else {
				nameBuffer[i] = byte(nb)
			}
		}
		eci.Name = string(nameBuffer)

		if eci.EcType == ALPHA {
			var alphaBool bool
			if alphaBool, err = reader.ReadBool(); err != nil {
				return nil, err
			}
			eci.AlphaAssociated = alphaBool
		}

	} else {
		eci.EcType = ALPHA
		eci.BitDepth = *NewBitDepthHeader()
		eci.DimShift = 0
		eci.Name = ""
		eci.AlphaAssociated = false
	}

	if eci.EcType == SPOT_COLOR {
		var err error
		if eci.Red, err = reader.ReadF16(); err != nil {
			return nil, err
		}
		if eci.Green, err = reader.ReadF16(); err != nil {
			return nil, err
		}
		if eci.Blue, err = reader.ReadF16(); err != nil {
			return nil, err
		}
		if eci.Solidity, err = reader.ReadF16(); err != nil {
			return nil, err
		}
	} else {
		eci.Red = 0
		eci.Green = 0
		eci.Blue = 0
		eci.Solidity = 0
	}

	if eci.EcType == COLOR_FILTER_ARRAY {
		if cfaIndex, err := reader.ReadU32(1, 0, 0, 2, 3, 4, 19, 8); err != nil {
			return nil, err
		} else {
			eci.CfaIndex = int32(cfaIndex)
		}
	} else {
		eci.CfaIndex = 1
	}
	return eci, nil
}
