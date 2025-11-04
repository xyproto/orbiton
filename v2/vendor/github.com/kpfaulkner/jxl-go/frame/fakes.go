package frame

import (
	"github.com/kpfaulkner/jxl-go/bundle"
	"github.com/kpfaulkner/jxl-go/image"
	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

type FakeFramer struct {
	lfGroup                *LFGroup
	hfGlobal               *HFGlobal
	lfGlobal               *LFGlobal
	header                 *FrameHeader
	passes                 []Pass
	groupSize              *util.Dimension
	groupPosInLFGroupPoint *util.Point
	imageHeader            *bundle.ImageHeader
}

func (f *FakeFramer) getLFGroupSize(lfGroupID int32) (util.Dimension, error) {
	//TODO implement me
	return util.Dimension{
		Width:  5,
		Height: 5,
	}, nil
}

func (f *FakeFramer) getNumLFGroups() uint32 {
	return 0
}

func (f *FakeFramer) getLFGroupLocation(lfGroupID int32) *util.Point {
	//TODO implement me
	panic("implement me")
}

func (f *FakeFramer) getGlobalTree() *MATree {
	//TODO implement me
	panic("implement me")
}
func (f *FakeFramer) setGlobalTree(tree *MATree) {}

func (f *FakeFramer) getLFGroupForGroup(groupID int32) *LFGroup {
	return f.lfGroup
}

func (f *FakeFramer) getHFGlobal() *HFGlobal {
	return f.hfGlobal
}

func (f *FakeFramer) getLFGlobal() *LFGlobal {
	return f.lfGlobal
}

func (f *FakeFramer) getFrameHeader() *FrameHeader {
	return f.header
}

func (f *FakeFramer) getPasses() []Pass {
	return f.passes
}

func (f *FakeFramer) getGroupSize(groupID int32) (util.Dimension, error) {
	return *f.groupSize, nil
}

func (f *FakeFramer) groupPosInLFGroup(lfGroupID int32, groupID uint32) util.Point {
	return *f.groupPosInLFGroupPoint
}

func (f *FakeFramer) getGlobalMetadata() *bundle.ImageHeader {
	return f.imageHeader
}

func NewFakeFramer() Framer {
	ff := &FakeFramer{
		header: &FrameHeader{
			jpegUpsamplingX: []int32{0, 0, 0},
			jpegUpsamplingY: []int32{0, 0, 0},
		},
		lfGlobal:    NewLFGlobal(),
		imageHeader: &bundle.ImageHeader{},
	}
	ff.lfGlobal.scaledDequant = []float32{1, 1, 1}
	return ff
}

func NewFakeHFBlockContextFunc(reader jxlio.BitReader, readClusterMap func(reader jxlio.BitReader, clusterMap []int, maxClusters int) (int, error)) (*HFBlockContext, error) {
	return nil, nil
}

func NewFakeHFMetadataFunc(reader jxlio.BitReader, parent *LFGroup, frame Framer) (*HFMetadata, error) {

	return &HFMetadata{}, nil
}

func NewFakeLFCoeffientsFunc(reader jxlio.BitReader, parent *LFGroup, frame Framer, lfBuffer []image.ImageBuffer, modularStreamFunc NewModularStreamFunc) (*LFCoefficients, error) {
	return &LFCoefficients{}, nil
}
