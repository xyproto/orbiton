package frame

import (
	"github.com/kpfaulkner/jxl-go/image"
	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

type LFGroup struct {
	lfCoeff    *LFCoefficients
	hfMetadata *HFMetadata

	lfGroupID      int32
	frame          Framer
	size           util.Dimension
	modularLFGroup ModularStreamer
}

func NewLFGroupWithReader(reader jxlio.BitReader, parent Framer, index int32, replaced []ModularChannel, lfBuffer []image.ImageBuffer, lfCoeffientsFunc NewLFCoefficientsWithReaderFunc, hfMetadataFunc NewHFMetadataWithReaderFunc) (*LFGroup, error) {
	lfg := &LFGroup{
		frame:     parent,
		lfGroupID: index,
	}

	pixelSize, err := lfg.frame.getLFGroupSize(lfg.lfGroupID)
	if err != nil {
		return nil, err
	}

	lfg.size = util.Dimension{
		Height: pixelSize.Height >> 3,
		Width:  pixelSize.Width >> 3,
	}

	if parent.getFrameHeader().Encoding == VARDCT {
		lfg.lfCoeff, err = lfCoeffientsFunc(reader, lfg, parent, lfBuffer, NewModularStreamWithChannels)
		if err != nil {
			return nil, err
		}
	} else {
		lfg.lfCoeff = nil
	}
	stream, err := NewModularStreamWithStreamIndex(reader, parent, 1+int(parent.getNumLFGroups()+uint32(lfg.lfGroupID)), replaced)
	if err != nil {
		return nil, err
	}

	lfg.modularLFGroup = stream
	err = stream.decodeChannels(reader, false)
	if err != nil {
		return nil, err
	}

	if parent.getFrameHeader().Encoding == VARDCT {
		metadata, err := hfMetadataFunc(reader, lfg, parent)
		if err != nil {
			return nil, err
		}
		lfg.hfMetadata = metadata
	} else {
		lfg.hfMetadata = nil
	}
	return lfg, nil
}
