package frame

import (
	"errors"

	"github.com/kpfaulkner/jxl-go/colour"
	"github.com/kpfaulkner/jxl-go/entropy"
	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

type LFGlobal struct {
	frame           Framer
	Patches         []Patch
	splines         []SplinesBundle
	noiseParameters []NoiseParameters
	lfDequant       []float32
	hfBlockCtx      *HFBlockContext
	lfChanCorr      *LFChannelCorrelation
	globalScale     int32
	quantLF         int32
	scaledDequant   []float32
	globalModular   ModularStreamer
}

func NewLFGlobal() *LFGlobal {
	lf := &LFGlobal{}
	lf.lfDequant = []float32{1.0 / 4096.0, 1.0 / 512.0, 1.0 / 256.0}
	lf.scaledDequant = make([]float32, 3)
	lf.lfChanCorr = &LFChannelCorrelation{
		colorFactor:      0,
		baseCorrelationX: 0,
		baseCorrelationB: 0,
		xFactorLF:        0,
		bFactorLF:        0,
	}
	lf.hfBlockCtx = &HFBlockContext{
		lfThresholds: util.MakeMatrix2D[int32](3, 3),
	}
	return lf
}

func NewLFGlobalWithReader(reader jxlio.BitReader, parent Framer, hfBlockContextFunc NewHFBlockContextFunc) (*LFGlobal, error) {

	lf := NewLFGlobal()
	lf.frame = parent
	extra := len(lf.frame.getGlobalMetadata().ExtraChannelInfo)
	if lf.frame.getFrameHeader().Flags&PATCHES != 0 {

		return nil, errors.New("Patches not implemented yet")

		stream, err := entropy.NewEntropyStreamWithReaderAndNumDists(reader, 10, entropy.ReadClusterMap)
		if err != nil {
			return nil, err
		}
		numPatches, err := stream.ReadSymbol(reader, 0)
		if err != nil {
			return nil, err
		}
		lf.Patches = make([]Patch, numPatches)
		for i := 0; i < int(numPatches); i++ {
			lf.Patches[i], err = NewPatchWithStreamAndReader(stream, reader, len(parent.getGlobalMetadata().ExtraChannelInfo), len(parent.getGlobalMetadata().AlphaIndices))
			if err != nil {
				return nil, err
			}
		}

	} else {
		lf.Patches = []Patch{}
	}

	if lf.frame.getFrameHeader().Flags&SPLINES != 0 {
		return nil, errors.New("Splines not implemented yet")
	} else {
		lf.splines = nil
	}

	if lf.frame.getFrameHeader().Flags&NOISE != 0 {
		return nil, errors.New("Noise not implemented yet")
	} else {
		lf.noiseParameters = nil
	}

	var err error
	var readDequant bool
	if readDequant, err = reader.ReadBool(); err != nil {
		return nil, err
	}
	if !readDequant {
		for i := 0; i < 3; i++ {
			if lf.lfDequant[i], err = reader.ReadF16(); err != nil {
				return nil, err
			}
			lf.lfDequant[i] *= (1.0 / 128.0)
		}
	}

	if lf.frame.getFrameHeader().Encoding == VARDCT {
		globalScale, err := reader.ReadU32(1, 11, 2049, 11, 4097, 12, 8193, 16)
		if err != nil {
			return nil, err
		}
		lf.globalScale = int32(globalScale)
		quantLF, err := reader.ReadU32(16, 0, 1, 5, 1, 8, 1, 16)
		if err != nil {
			return nil, err
		}
		lf.quantLF = int32(quantLF)
		for i := 0; i < 3; i++ {
			lf.scaledDequant[i] = (1 << 16) * lf.lfDequant[i] / float32(lf.globalScale*lf.quantLF)
		}
		lf.hfBlockCtx, err = hfBlockContextFunc(reader, entropy.ReadClusterMap)
		if err != nil {
			return nil, err
		}
		lf.lfChanCorr, err = NewLFChannelCorrelationWithReader(reader)
		if err != nil {
			return nil, err
		}
	} else {
		lf.globalScale = 0
		lf.quantLF = 0
		lf.hfBlockCtx = nil
		lf.lfChanCorr, err = NewLFChannelCorrelation()
		if err != nil {
			return nil, err
		}
	}

	hasGlobalTree, err := reader.ReadBool()
	if err != nil {
		return nil, err
	}
	var globalTree *MATree
	if hasGlobalTree {
		globalTree, err = NewMATreeWithReader(reader)
		if err != nil {
			return nil, err
		}
	} else {
		globalTree = nil
	}
	lf.frame.setGlobalTree(globalTree)
	subModularChannelCount := extra
	ecStart := 0
	if lf.frame.getFrameHeader().Encoding == MODULAR {
		if !lf.frame.getFrameHeader().DoYCbCr && !lf.frame.getGlobalMetadata().XybEncoded &&
			lf.frame.getGlobalMetadata().ColourEncoding.ColourEncoding == colour.CE_GRAY {
			ecStart = 1
		} else {
			ecStart = 3
		}
	}
	subModularChannelCount += ecStart

	globalModular, err := NewModularStreamWithReader(reader, parent, 0, subModularChannelCount, ecStart)
	if err != nil {
		return nil, err
	}
	lf.globalModular = globalModular
	if err = lf.globalModular.decodeChannels(reader, true); err != nil {
		return nil, err
	}

	return lf, nil
}
