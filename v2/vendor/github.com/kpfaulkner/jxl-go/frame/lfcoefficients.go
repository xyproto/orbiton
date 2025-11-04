package frame

import (
	"errors"
	"math"

	"github.com/kpfaulkner/jxl-go/image"
	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

type NewModularStreamFunc func(reader jxlio.BitReader, frame Framer, streamIndex int, channelCount int, ecStart int, channelArray []ModularChannel) (ModularStreamer, error)

type NewLFCoefficientsWithReaderFunc func(reader jxlio.BitReader, parent *LFGroup, frame Framer, lfBuffer []image.ImageBuffer, modularStreamFunc NewModularStreamFunc) (*LFCoefficients, error)

type LFCoefficients struct {
	dequantLFCoeff [][][]float32
	lfIndex        [][]int32
	frame          Framer
}

func NewLFCoefficientsWithReader(reader jxlio.BitReader, parent *LFGroup, frame Framer, lfBuffer []image.ImageBuffer, modularStreamFunc NewModularStreamFunc) (*LFCoefficients, error) {
	lf := &LFCoefficients{}

	lf.frame = frame
	lf.lfIndex = util.MakeMatrix2D[int32](parent.size.Height, parent.size.Width)
	header := frame.getFrameHeader()
	adapativeSmoothing := (header.Flags & (SKIP_ADAPTIVE_LF_SMOOTHING | USE_LF_FRAME)) == 0
	info := make([]ModularChannel, 3)
	dequantLFCoeff := util.MakeMatrix3D[float32](3, 0, 0)
	subSampled := header.jpegUpsamplingY[0] != 0 || header.jpegUpsamplingY[1] != 0 || header.jpegUpsamplingY[2] != 0 ||
		header.jpegUpsamplingX[0] != 0 || header.jpegUpsamplingX[1] != 0 || header.jpegUpsamplingX[2] != 0
	if adapativeSmoothing && subSampled {
		return nil, errors.New("Adaptive smoothing is incompatible with subsampling")
	}
	for i := 0; i < 3; i++ {
		sizeY := parent.size.Height >> header.jpegUpsamplingY[i]
		sizeX := parent.size.Width >> header.jpegUpsamplingX[i]
		info[cMap[i]] = *NewModularChannelWithAllParams(int32(sizeY), int32(sizeX), header.jpegUpsamplingY[i], header.jpegUpsamplingX[i], false)
		dequantLFCoeff[i] = util.MakeMatrix2D[float32, uint32](sizeY, sizeX)
	}

	if (header.Flags & USE_LF_FRAME) != 0 {
		pos := frame.getLFGroupLocation(parent.lfGroupID)
		pY := pos.Y << 8
		pX := pos.X << 8
		lf.dequantLFCoeff = dequantLFCoeff
		for c := 0; c < 3; c++ {
			lfBuffer[c].CastToFloatIfInt(^(^0 << frame.getGlobalMetadata().BitDepth.BitsPerSample))
			b := lfBuffer[c].FloatBuffer
			for y := int32(0); y < int32(len(dequantLFCoeff[c])); y++ {
				for x, d := pX, 0; x < pX+int32(len(dequantLFCoeff[c][y])); x, d = x+1, d+1 {
					dequantLFCoeff[c][y][d] = b[pY+y][x]
				}
			}
		}
	}

	extraPrecision, err := reader.ReadBits(2)
	if err != nil {
		return nil, err
	}

	// use function to pass in func to create ModularStream... just easier to test.
	lfQuantStream, err := modularStreamFunc(reader, frame, int(1+parent.lfGroupID), len(info), 0, info)
	if err != nil {
		return nil, err
	}

	err = lfQuantStream.decodeChannels(reader, false)
	if err != nil {
		return nil, err
	}
	lfQuant := lfQuantStream.getDecodedBuffer()
	scaledDequant := frame.getLFGlobal().scaledDequant
	for i := 0; i < 3; i++ {
		c := cMap[i]
		xx := 1 << extraPrecision
		sd := scaledDequant[i] / float32(xx)
		for y := 0; y < len(lfQuant[c]); y++ {
			dq := dequantLFCoeff[i][y]
			q := lfQuant[c][y]
			for x := 0; x < len(lfQuant[c][y]); x++ {
				dq[x] = float32(q[x]) * sd
			}
		}
	}

	if !subSampled {

		// TOOD(kpfaulkner) investigate what this really does.
		lfc := frame.getLFGlobal().lfChanCorr
		kX := lfc.baseCorrelationX + float32(lfc.xFactorLF-128)/float32(lfc.colorFactor)
		kB := lfc.baseCorrelationB + float32(lfc.bFactorLF-128)/float32(lfc.colorFactor)
		dqLFY := dequantLFCoeff[1]
		dqLFX := dequantLFCoeff[0]
		dqLFB := dequantLFCoeff[2]
		for y := 0; y < len(dqLFY); y++ {
			dqLFYy := dqLFY[y]
			dqLFXy := dqLFX[y]
			dqLFBY := dqLFB[y]
			for x := 0; x < len(dqLFYy); x++ {
				dqLFXy[x] += kX * dqLFYy[x]
				dqLFBY[x] += kB * dqLFYy[x]
			}
		}
	}

	if adapativeSmoothing {
		lf.dequantLFCoeff = adaptiveSmooth(dequantLFCoeff, scaledDequant)
	} else {
		lf.dequantLFCoeff = dequantLFCoeff
	}

	err = lf.populatedLFIndex(parent, lfQuant)
	return lf, nil
}

func (c *LFCoefficients) populatedLFIndex(parent *LFGroup, lfQuant [][][]int32) error {
	hfctx := c.frame.getLFGlobal().hfBlockCtx
	for y := uint32(0); y < parent.size.Height; y++ {
		for x := uint32(0); x < parent.size.Width; x++ {
			c.lfIndex[y][x] = c.getLFIndex(lfQuant, hfctx, y, x)
		}
	}
	return nil
}

func (c *LFCoefficients) getLFIndex(lfQuant [][][]int32, hfctx *HFBlockContext, y uint32, x uint32) int32 {
	index := make([]int, 3)
	header := c.frame.getFrameHeader()
	for i := 0; i < 3; i++ {
		sy := y >> header.jpegUpsamplingY[i]
		sx := x >> header.jpegUpsamplingX[i]
		hft := hfctx.lfThresholds[i]
		for j := 0; j < len(hft); j++ {
			if lfQuant[cMap[i]][sy][sx] > hft[j] {
				index[i]++
			}
		}
	}

	lfIndex := index[0]
	lfIndex *= len(hfctx.lfThresholds[2]) + 1
	lfIndex += index[2]
	lfIndex *= len(hfctx.lfThresholds[1]) + 1
	lfIndex += index[1]
	return int32(lfIndex)
}

func adaptiveSmooth(coeff [][][]float32, scaledDequant []float32) [][][]float32 {
	weighted := make([][][]float32, 3)
	gap := make([][]float32, len(coeff[0]))
	dequantLFCoeff := make([][][]float32, 3)
	for i := 0; i < 3; i++ {
		co := coeff[i]
		weighted[i] = make([][]float32, len(co))
		sd := scaledDequant[i]
		for y := 01; y < len(co)-1; y++ {
			coy := co[y]
			coym := co[y-1]
			coyp := co[y+1]
			if gap[y] == nil {
				gap[y] = make([]float32, len(coy))
				for x := 0; x < len(gap[y]); x++ {
					gap[y][x] = 0.5
				}
			}
			gy := gap[y]
			weighted[i][y] = make([]float32, len(coy))
			wy := weighted[i][y]
			for x := 01; x < len(coy)-1; x++ {
				sample := coy[x]
				adjacent := coy[x-1] + coy[x+1] + coym[x] + coyp[x]
				diag := coym[x-1] + coym[x+1] + coyp[x-1] + coyp[x+1]
				wy[x] = 0.05226273532324128*sample + 0.20345139757231578*adjacent + 0.0334829185968739*diag
				g := float32(math.Abs(float64(sample-wy[x])) * float64(sd))
				if g > gy[x] {
					gy[x] = g
				}
			}
		}
	}

	for y := 0; y < len(gap); y++ {
		if gap[y] == nil {
			continue
		}
		gy := gap[y]
		for x := 0; x < len(gy); x++ {
			gy[x] = util.Max[float32](0.0, 3.0-4.0*gy[x])
		}
	}

	for i := 0; i < 3; i++ {
		co := coeff[i]
		dequantLFCoeff[i] = make([][]float32, len(co))
		dqi := dequantLFCoeff[i]
		wi := weighted[i]
		for y := 0; y < len(co); y++ {
			coy := co[y]
			dqi[y] = make([]float32, len(coy))
			dqy := dqi[y]
			gy := gap[y]
			wiy := wi[y]
			if y == 0 || y+1 == len(co) {
				copy(dqy, coy)
				continue
			}
			for x := 0; x < len(coy); x++ {
				if x == 0 || x+1 == len(coy) {
					dqy[x] = coy[x]
					continue
				}
				dqy[x] = (coy[x]-wiy[x])*gy[x] + wiy[x]
			}

		}
	}
	return dequantLFCoeff
}
