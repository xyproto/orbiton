package frame

import (
	"github.com/kpfaulkner/jxl-go/entropy"
	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

type Pass struct {
	replacedChannels []*ModularChannel
	hfPass           *HFPass
	minShift         uint32
	maxShift         uint32
}

func NewPassWithReader(reader jxlio.BitReader, frame *Frame, passIndex uint32, prevMinShift uint32) (Pass, error) {
	p := Pass{}

	if passIndex > 0 {
		p.maxShift = prevMinShift
	} else {
		p.maxShift = 3
	}

	n := -1
	passes := frame.Header.passes
	for i := 0; i < len(passes.lastPass); i++ {
		if passes.lastPass[i] == passIndex {
			n = i
			break
		}
	}

	if n >= 0 {
		p.minShift = uint32(util.CeilLog1p(int64(passes.downSample[n] - 1)))
	} else {
		p.minShift = p.maxShift
	}

	stream := frame.LfGlobal.globalModular
	p.replacedChannels = make([]*ModularChannel, len(stream.getChannels()))
	for i := 0; i < len(p.replacedChannels); i++ {
		ch := stream.getChannels()[i]
		if !ch.decoded {
			m := uint32(min(ch.hshift, ch.vshift))
			if p.minShift <= m && m < p.maxShift {
				p.replacedChannels[i] = NewModularChannelFromChannel(*ch)
			}
		}
	}
	var err error
	if frame.Header.Encoding == VARDCT {
		p.hfPass, err = NewHFPassWithReader(reader, frame, passIndex, entropy.ReadClusterMap, entropy.NewEntropyStreamWithReaderAndNumDists, readPermutation)
		if err != nil {
			return Pass{}, err
		}
	} else {
		p.hfPass = nil
	}

	return p, nil

}
