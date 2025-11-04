package frame

import (
	"math"

	"github.com/kpfaulkner/jxl-go/util"
)

type NoiseGroup struct {
	rng *XorShiro
}

func NewNoiseGroupWithHeader(header *FrameHeader, seed0 int64, noiseBuffer [][][]float32, x0 int32, y0 int32) *NoiseGroup {
	ng := &NoiseGroup{}

	seed1 := (int64(x0) << 32) | int64(y0)
	xSize := util.Min(header.groupDim, header.Width-uint32(x0))
	ySize := util.Min(header.groupDim, header.Height-uint32(y0))
	ng.rng = NewXorShiroWith2Seeds(seed0, seed1)
	bits := make([]int64, 16)
	for c := 0; c < 3; c++ {
		for y := 0; y < int(ySize); y++ {
			for x := 0; x < int(xSize); x++ {
				ng.rng.fill(bits)
				for i := 0; i < 16 && x+i < int(xSize); i++ {
					f := (uint32(bits[i]) >> 9) | 0x3f_80_00_00
					noiseBuffer[c][y0+int32(y)][x0+int32(x)+int32(i)] = math.Float32frombits(f)
				}
			}
		}
	}
	return ng
}
