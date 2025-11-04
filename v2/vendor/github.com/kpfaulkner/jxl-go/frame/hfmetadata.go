package frame

import (
	"errors"
	"fmt"

	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

type HFMetadata struct {
	nbBlocks       uint64
	dctSelect      [][]*TransformType
	hfMultiplier   [][]int32
	hfStreamBuffer [][][]int32
	parent         *LFGroup
	blockList      []util.Point
}
type NewHFMetadataWithReaderFunc func(reader jxlio.BitReader, parent *LFGroup, frame Framer) (*HFMetadata, error)

func NewHFMetadataWithReader(reader jxlio.BitReader, parent *LFGroup, frame Framer) (*HFMetadata, error) {
	hf := &HFMetadata{
		parent: parent,
	}

	n := util.CeilLog2(parent.size.Height * parent.size.Width)
	nbBlocks, err := reader.ReadBits(uint32(n))
	if err != nil {
		return nil, err
	}
	hf.nbBlocks = nbBlocks + 1

	correlationHeight := int32((parent.size.Height + 7) / 8)
	correlationWidth := int32((parent.size.Width + 7) / 8)
	xFromY := NewModularChannelWithAllParams(correlationHeight, correlationWidth, 0, 0, false)
	bFromY := NewModularChannelWithAllParams(correlationHeight, correlationWidth, 0, 0, false)
	blockInfo := NewModularChannelWithAllParams(2, int32(hf.nbBlocks), 0, 0, false)
	sharpness := NewModularChannelWithAllParams(int32(parent.size.Height), int32(parent.size.Width), 0, 0, false)
	hfStream, err := NewModularStreamWithStreamIndex(reader, frame, 1+2*int(frame.getNumLFGroups())+int(parent.lfGroupID), []ModularChannel{*xFromY, *bFromY, *blockInfo, *sharpness})
	if err != nil {
		return nil, err
	}
	err = hfStream.decodeChannels(reader, false)
	if err != nil {
		return nil, err
	}

	hf.hfStreamBuffer = hfStream.getDecodedBuffer()
	hf.dctSelect = util.MakeMatrix2D[*TransformType](parent.size.Height, parent.size.Width)
	hf.hfMultiplier = util.MakeMatrix2D[int32](parent.size.Height, parent.size.Width)
	blockInfoBuffer := hf.hfStreamBuffer[2]
	lastBlock := util.Point{X: 0, Y: 0}
	tta := allDCT
	hf.blockList = make([]util.Point, hf.nbBlocks)
	for i := uint64(0); i < hf.nbBlocks; i++ {
		t := blockInfoBuffer[0][i]
		if t > 26 || t < 0 {
			return nil, errors.New(fmt.Sprintf("Invalid transform Type %d", t))
		}
		tt := tta[t]
		pos, err := hf.placeBlock(lastBlock, tt, 1+blockInfoBuffer[1][i])
		if err != nil {
			return nil, err
		}
		lastBlock = util.Point{
			X: pos.X,
			Y: pos.Y,
		}
		hf.blockList[i] = pos
	}
	return hf, nil
}

// FIXME(kpfaulkner) 20241102 think somethings wrong in here...  llfScale is bad
func (m *HFMetadata) placeBlock(lastBlock util.Point, block TransformType, mul int32) (util.Point, error) {

	x := lastBlock.X
outerY:

	for y := lastBlock.Y; y < int32(len(m.dctSelect)); y++ {
		x = 0
		dctY := m.dctSelect[y]
	outerX:
		for ; x < int32(len(dctY)); x++ {

			if block.dctSelectWidth+x > int32(len(dctY)) {
				x = lastBlock.X
				continue outerY
			}

			for ix := int32(0); ix < block.dctSelectWidth; ix++ {
				tt := dctY[x+ix]
				if tt != nil {
					x += tt.dctSelectWidth - 1
					continue outerX
				}
			}
			pos := util.Point{
				X: x,
				Y: y,
			}
			//m.hfMultiplier[y][x] = mul
			for iy := int32(0); iy < block.dctSelectHeight; iy++ {
				for f := x; f < x+block.dctSelectWidth; f++ {
					m.dctSelect[y+iy][f] = &block
					m.hfMultiplier[y+iy][f] = mul
				}
			}
			return pos, nil
		}
	}
	return util.Point{}, errors.New("No space for block")
}
