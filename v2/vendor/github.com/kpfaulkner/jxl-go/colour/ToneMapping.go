package colour

import (
	"errors"

	"github.com/kpfaulkner/jxl-go/jxlio"
)

type ToneMapping struct {
	IntensityTarget      float32
	MinNits              float32
	LinearBelow          float32
	RelativeToMaxDisplay bool
}

func NewToneMapping() *ToneMapping {
	tm := &ToneMapping{}

	tm.IntensityTarget = 255.0
	tm.MinNits = 0.0
	tm.RelativeToMaxDisplay = false
	tm.LinearBelow = 0

	return tm
}

func NewToneMappingWithReader(reader jxlio.BitReader) (*ToneMapping, error) {
	tm := &ToneMapping{}
	var err error
	var useToneMapping bool
	if useToneMapping, err = reader.ReadBool(); err != nil {
		return nil, err
	}
	if useToneMapping {
		tm.IntensityTarget = 255.0
		tm.MinNits = 0.0
		tm.RelativeToMaxDisplay = false
		tm.LinearBelow = 0
	} else {
		if tm.IntensityTarget, err = reader.ReadF16(); err != nil {
			return nil, err
		}
		if tm.IntensityTarget <= 0 {
			return nil, errors.New("Intensity Target must be positive")
		}
		if tm.MinNits, err = reader.ReadF16(); err != nil {
			return nil, err
		}
		if tm.MinNits < 0 {
			return nil, errors.New("Min Nits must be positive")
		}
		if tm.MinNits > tm.IntensityTarget {
			return nil, errors.New("Min Nits must be at most the Intensity Target")
		}
		if tm.RelativeToMaxDisplay, err = reader.ReadBool(); err != nil {
			return nil, err
		}
		if tm.LinearBelow, err = reader.ReadF16(); err != nil {
			return nil, err
		}
		if tm.RelativeToMaxDisplay && (tm.LinearBelow < 0 || tm.LinearBelow > 1) {
			return nil, errors.New("Linear Below out of relative range")
		}
		if !tm.RelativeToMaxDisplay && tm.LinearBelow < 0 {
			return nil, errors.New("Linear Below must be nonnegative")
		}
	}
	return tm, nil
}

func (tm *ToneMapping) GetIntensityTarget() float32 {
	return tm.IntensityTarget
}
