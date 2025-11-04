package frame

import (
	"github.com/kpfaulkner/jxl-go/bundle"
	"github.com/kpfaulkner/jxl-go/jxlio"
)

type RestorationFilter struct {
	gab                bool
	customGab          bool
	gab1Weights        []float32
	gab2Weights        []float32
	epfIterations      uint32
	epfSharpCustom     bool
	epfSharpLut        []float32
	epfChannelScale    []float32
	epfSigmaCustom     bool
	epfQuantMul        float32
	epfPass0SigmaScale float32
	epfPass2SigmaScale float32
	epfBorderSadMul    float32
	epfSigmaForModular float32
	extensions         *bundle.Extensions
	epfWeightCustom    bool
}

func NewRestorationFilter() *RestorationFilter {
	rf := &RestorationFilter{}
	rf.epfSharpLut = []float32{0, 1 / 7, 2 / 7, 3 / 7, 4 / 7, 5 / 7, 6 / 7, 1}
	rf.epfChannelScale = []float32{40.0, 5.0, 3.5}
	rf.gab1Weights = []float32{0.115169525, 0.115169525, 0.115169525}
	rf.gab2Weights = []float32{0.061248592, 0.061248592, 0.061248592}
	rf.gab = true
	rf.customGab = false
	rf.epfIterations = 2
	rf.epfSharpCustom = false
	rf.epfWeightCustom = false
	rf.epfSigmaCustom = false
	rf.epfQuantMul = 0.46
	rf.epfPass0SigmaScale = 0.9
	rf.epfPass2SigmaScale = 6.5
	rf.epfBorderSadMul = 2.0 / 3.0
	rf.epfSigmaForModular = 1.0
	rf.extensions = bundle.NewExtensions()
	for i := 0; i < 8; i++ {
		rf.epfSharpLut[i] *= rf.epfQuantMul
	}

	return rf
}

func NewRestorationFilterWithReader(reader jxlio.BitReader, encoding uint32) (*RestorationFilter, error) {
	rf := &RestorationFilter{}
	rf.epfSharpLut = []float32{0, 1.0 / 7.0, 2.0 / 7.0, 3.0 / 7.0, 4.0 / 7.0, 5.0 / 7.0, 6.0 / 7.0, 1.0}
	rf.epfChannelScale = []float32{40.0, 5.0, 3.5}
	rf.gab1Weights = []float32{0.115169525, 0.115169525, 0.115169525}
	rf.gab2Weights = []float32{0.061248592, 0.061248592, 0.061248592}

	var allDefault bool
	var err error
	if allDefault, err = reader.ReadBool(); err != nil {
		return nil, err
	}
	if allDefault {
		rf.gab = true
	} else {
		if rf.gab, err = reader.ReadBool(); err != nil {
			return nil, err
		}
	}

	if !allDefault && rf.gab {
		if rf.customGab, err = reader.ReadBool(); err != nil {
			return nil, err
		}
	} else {
		rf.customGab = false
	}

	if rf.customGab {
		for i := 0; i < 3; i++ {
			if rf.gab1Weights[i], err = reader.ReadF16(); err != nil {
				return nil, err
			}
			if rf.gab2Weights[i], err = reader.ReadF16(); err != nil {
				return nil, err
			}
		}
	}

	if allDefault {
		rf.epfIterations = 2
	} else {
		if epfItreations, err := reader.ReadBits(2); err != nil {
			return nil, err
		} else {
			rf.epfIterations = uint32(epfItreations)
		}
	}

	if !allDefault && rf.epfIterations > 0 && encoding == VARDCT {
		if rf.epfSharpCustom, err = reader.ReadBool(); err != nil {
			return nil, err
		}
	} else {
		rf.epfSharpCustom = false
	}
	if rf.epfSharpCustom {
		for i := 0; i < len(rf.epfSharpLut); i++ {
			if rf.epfSharpLut[i], err = reader.ReadF16(); err != nil {
				return nil, err
			}
		}

	}

	if !allDefault && rf.epfIterations > 9 {
		if rf.epfWeightCustom, err = reader.ReadBool(); err != nil {
			return nil, err
		}
	} else {
		rf.epfWeightCustom = false
	}

	if rf.epfWeightCustom {
		for i := 0; i < len(rf.epfChannelScale); i++ {
			if rf.epfChannelScale[i], err = reader.ReadF16(); err != nil {
				return nil, err
			}
		}

		_, _ = reader.ReadBits(32) // ??? what do we do with this data?
	}

	if !allDefault && rf.epfIterations > 0 {
		if rf.epfSigmaCustom, err = reader.ReadBool(); err != nil {
			return nil, err
		}
	} else {
		rf.epfSigmaCustom = false
	}

	if rf.epfSigmaCustom && encoding == VARDCT {
		rf.epfQuantMul, err = reader.ReadF16()
		if err != nil {
			return nil, err
		}
	} else {
		rf.epfQuantMul = 0.46
	}
	if rf.epfSigmaCustom {
		if rf.epfPass0SigmaScale, err = reader.ReadF16(); err != nil {
			return nil, err
		}
		if rf.epfPass2SigmaScale, err = reader.ReadF16(); err != nil {
			return nil, err
		}
		if rf.epfBorderSadMul, err = reader.ReadF16(); err != nil {
			return nil, err
		}
	} else {
		rf.epfPass0SigmaScale = 0.9
		rf.epfPass2SigmaScale = 6.5
		rf.epfBorderSadMul = 2.0 / 3.0
	}

	if !allDefault && rf.epfIterations > 0 && encoding == MODULAR {
		if rf.epfSigmaForModular, err = reader.ReadF16(); err != nil {
			return nil, err
		}
	} else {
		rf.epfSigmaForModular = 1.0
	}

	if allDefault {
		rf.extensions = bundle.NewExtensions()
	} else {
		rf.extensions, err = bundle.NewExtensionsWithReader(reader)
		if err != nil {
			return nil, err
		}
	}

	for i := 0; i < 8; i++ {
		rf.epfSharpLut[i] *= rf.epfQuantMul
	}

	return rf, nil
}
