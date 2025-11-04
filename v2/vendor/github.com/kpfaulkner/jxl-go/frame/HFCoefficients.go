package frame

import (
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/kpfaulkner/jxl-go/entropy"
	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

var (
	coeffFreqCtx = []int32{
		-1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
		15, 15, 16, 16, 17, 17, 18, 18, 19, 19, 20, 20, 21, 21, 22, 22,
		23, 23, 23, 23, 24, 24, 24, 24, 25, 25, 25, 25, 26, 26, 26, 26,
		27, 27, 27, 27, 28, 28, 28, 28, 29, 29, 29, 29, 30, 30, 30, 30}
	coeffNumNonzeroCtx = []int32{
		-1, 0, 31, 62, 62, 93, 93, 93, 93, 123, 123, 123, 123, 152, 152,
		152, 152, 152, 152, 152, 152, 180, 180, 180, 180, 180, 180, 180,
		180, 180, 180, 180, 180, 206, 206, 206, 206, 206, 206, 206, 206,
		206, 206, 206, 206, 206, 206, 206, 206, 206, 206, 206, 206, 206,
		206, 206, 206, 206, 206, 206, 206, 206, 206, 206}
)

type HFCoefficients struct {
	hfPreset        int32
	groupID         uint32
	frame           Framer
	hfctx           *HFBlockContext
	lfg             *LFGroup
	stream          *entropy.EntropyStream
	quantizedCoeffs [][][]int32
	dequantHFCoeff  [][][]float32
	groupPos        util.Point
	blocks          []*util.Point
}

func NewHFCoefficientsWithReader(reader jxlio.BitReader, frame Framer, pass uint32, group uint32) (*HFCoefficients, error) {
	hf := &HFCoefficients{}

	if frame == nil || reader == nil {
		return nil, errors.New("frame or reader is nil")
	}

	hfPreset, err := reader.ReadBits(uint32(util.CeilLog1p(frame.getHFGlobal().numHFPresets - 1)))
	if err != nil {
		return nil, err
	}
	hf.hfPreset = int32(hfPreset)
	hf.groupID = group
	hf.frame = frame
	hf.hfctx = frame.getLFGlobal().hfBlockCtx
	hf.lfg = frame.getLFGroupForGroup(int32(group))
	offset := 495 * hf.hfctx.numClusters * hf.hfPreset
	header := frame.getFrameHeader()
	shift := header.passes.shift[pass]
	hfPass := hf.frame.getPasses()[pass].hfPass
	size, err := hf.frame.getGroupSize(int32(hf.groupID))
	if err != nil {
		return nil, err
	}
	nonZeros := util.MakeMatrix3D[int32](3, 32, 32)
	hf.stream = entropy.NewEntropyStreamWithStream(hfPass.contextStream)
	hf.quantizedCoeffs = util.MakeMatrix3D[int32](3, 0, 0)
	hf.dequantHFCoeff = util.MakeMatrix3D[float32](3, 0, 0)

	for c := 0; c < 3; c++ {
		sY := size.Height >> header.jpegUpsamplingY[c]
		sX := size.Width >> header.jpegUpsamplingX[c]
		hf.quantizedCoeffs[c] = util.MakeMatrix2D[int32](sY, sX)
		hf.dequantHFCoeff[c] = util.MakeMatrix2D[float32](sY, sX)
	}

	hf.groupPos = hf.frame.groupPosInLFGroup(hf.lfg.lfGroupID, hf.groupID)
	hf.groupPos.Y <<= 5
	hf.groupPos.X <<= 5
	hf.blocks = make([]*util.Point, len(hf.lfg.hfMetadata.blockList))
	for i := 0; i < len(hf.lfg.hfMetadata.blockList); i++ {
		posInLfg := hf.lfg.hfMetadata.blockList[i]
		groupY := posInLfg.Y - hf.groupPos.Y
		groupX := posInLfg.X - hf.groupPos.X
		if groupY < 0 || groupX < 0 || groupY >= 32 || groupX >= 32 {
			continue
		}
		hf.blocks[i] = &posInLfg
		tt := hf.lfg.hfMetadata.dctSelect[posInLfg.Y][posInLfg.X]
		flip := tt.flip()
		hfMult := hf.lfg.hfMetadata.hfMultiplier[posInLfg.Y][posInLfg.X]
		lfIndex := hf.lfg.lfCoeff.lfIndex[posInLfg.Y][posInLfg.X]
		numBlocks := tt.dctSelectHeight * tt.dctSelectWidth
		for _, c := range cMap {
			sGroupY := groupY >> header.jpegUpsamplingY[c]
			sGroupX := groupX >> header.jpegUpsamplingX[c]
			if groupY != sGroupY<<header.jpegUpsamplingY[c] || groupX != sGroupX<<header.jpegUpsamplingX[c] {
				continue
			}

			pixelGroupY := sGroupY << 3
			pixelGroupX := sGroupX << 3
			predicted := getPredictedNonZeros(nonZeros, c, sGroupY, sGroupX)
			blockCtx := hf.getBlockContext(c, tt.orderID, hfMult, lfIndex)
			nonZeroCtx := offset + hf.getNonZeroContext(predicted, blockCtx)
			nonZero, err := hf.stream.ReadSymbol(reader, int(nonZeroCtx))
			if err != nil {
				return nil, err
			}
			nz := nonZeros[c]
			for iy := int32(0); iy < tt.dctSelectHeight; iy++ {
				for ix := int32(0); ix < tt.dctSelectWidth; ix++ {
					nz[sGroupY+iy][sGroupX+ix] = (nonZero + numBlocks - 1) / numBlocks
				}
			}

			// TODO(kpfaulkner) check this...  taken from JXLatte
			if nonZero <= 0 {
				continue
			}
			orderSize := int32(len(hfPass.order[tt.orderID][c]))
			ucoeff := make([]int32, orderSize-numBlocks)
			histCtx := offset + 458*blockCtx + 37*hf.hfctx.numClusters
			for k := int32(0); k < int32(len(ucoeff)); k++ {
				var prev int32
				if k == 0 {
					if nonZero > orderSize/16 {
						prev = 0
					} else {
						prev = 1
					}
				} else {
					if ucoeff[k-1] != 0 {
						prev = 1
					} else {
						prev = 0
					}
				}
				ctx := histCtx + hf.getCoefficientContext(k+numBlocks, nonZero, numBlocks, prev)
				uc, err := hf.stream.ReadSymbol(reader, int(ctx))
				if err != nil {
					return nil, err
				}
				ucoeff[k] = uc
				order := hfPass.order[tt.orderID][c][k+numBlocks]
				posY := pixelGroupY
				posX := pixelGroupX
				if flip {
					posY += order.X
					posX += order.Y
				} else {
					posY += order.Y
					posX += order.X
				}

				hf.quantizedCoeffs[c][posY][posX] = jxlio.UnpackSigned(uint32(ucoeff[k])) << shift
				if ucoeff[k] != 0 {
					nonZero--
					if nonZero == 0 {
						break
					}
				}
			}

			if nonZero != 0 {
				return nil, errors.New("nonZero != 0")
			}
		}

	}
	if !hf.stream.ValidateFinalState() {
		return nil, errors.New(fmt.Sprintf("Illegal final state in passgroup pass %d : group %d", pass, group))
	}
	return hf, nil
}

func (hf *HFCoefficients) DisplayHFCoefficients() {
	hf.displayQuantizedCoeffs()
	hf.displayDequantHFCoeff()
}

func (hf *HFCoefficients) displayQuantizedCoeffs() {
	grandTotal := int32(0)
	for c := 0; c < 3; c++ {
		for y := 0; y < len(hf.quantizedCoeffs[c]); y++ {
			total := int32(0)
			for x := 0; x < len(hf.quantizedCoeffs[c][y]); x++ {
				total += hf.quantizedCoeffs[c][y][x]
			}
			grandTotal += total
		}
	}
}

func (hf *HFCoefficients) displayDequantHFCoeff() {
	for c := 0; c < 3; c++ {
		for y := 0; y < len(hf.dequantHFCoeff[c]); y++ {
			total := float32(0)
			for x := 0; x < len(hf.dequantHFCoeff[c][y]); x++ {
				total += hf.dequantHFCoeff[c][y][x]
			}
		}
	}
}

func (hf *HFCoefficients) getBlockContext(c int, orderID int32, hfMult int32, lfIndex int32) int32 {

	var idx int
	if c < 2 {
		idx = 1 - c
	} else {
		idx = c
	}
	idx = idx*13 + int(orderID)
	idx *= len(hf.hfctx.qfThresholds) + 1
	for _, t := range hf.hfctx.qfThresholds {
		if hfMult > t {
			idx++
		}
	}
	idx *= int(hf.hfctx.numLFContexts)
	return int32(hf.hfctx.clusterMap[int32(idx)+lfIndex])
}

func (hf *HFCoefficients) getNonZeroContext(predicted int32, ctx int32) int32 {

	if predicted > 64 {
		predicted = 64
	}
	if predicted < 8 {
		return ctx + hf.hfctx.numClusters*predicted
	}
	return ctx + hf.hfctx.numClusters*(4+predicted/2)
}

func (hf *HFCoefficients) getCoefficientContext(k int32, nonZeros int32, numBlocks int32, prev int32) int32 {
	nonZeros = (nonZeros + numBlocks - 1) / numBlocks
	k /= numBlocks
	return (coeffNumNonzeroCtx[nonZeros]+coeffFreqCtx[k])*2 + prev
}

func (hf *HFCoefficients) bakeDequantizedCoeffs() error {

	if err := hf.dequantizeHFCoefficients(); err != nil {
		return err
	}

	if err := hf.chromaFromLuma(); err != nil {
		return err
	}

	if err := hf.finalizeLLF(); err != nil {
		return err
	}

	return nil
}

func (hf *HFCoefficients) dequantizeHFCoefficients() error {
	matrix := hf.frame.getGlobalMetadata().OpsinInverseMatrix
	header := hf.frame.getFrameHeader()
	globalScale := 65536.0 / float32(hf.frame.getLFGlobal().globalScale)
	scaleFactor := [3]float32{
		globalScale * float32(math.Pow(0.8, float64(header.xqmScale-2))),
		globalScale,
		globalScale * float32(math.Pow(0.8, float64(header.bqmScale-2))),
	}
	weights := hf.frame.getHFGlobal().weights
	qbclut := [][]float32{
		{-matrix.QuantBias[0], 0.0, matrix.QuantBias[0]},
		{-matrix.QuantBias[1], 0.0, matrix.QuantBias[1]},
		{-matrix.QuantBias[2], 0.0, matrix.QuantBias[2]},
	}

	for i := 0; i < len(hf.blocks); i++ {
		pos := hf.blocks[i]
		if pos == nil {
			continue
		}

		tt := hf.lfg.hfMetadata.dctSelect[pos.Y][pos.X]
		groupY := pos.Y - hf.groupPos.Y
		groupX := pos.X - hf.groupPos.X
		flip := tt.flip()
		w2 := weights[tt.parameterIndex]
		for c := 0; c < 3; c++ {
			sGroupY := groupY >> header.jpegUpsamplingY[c]
			sGroupX := groupX >> header.jpegUpsamplingX[c]
			if groupY != sGroupY<<header.jpegUpsamplingY[c] ||
				groupX != sGroupX<<header.jpegUpsamplingX[c] {
				continue
			}

			w3 := w2[c]
			sfc := scaleFactor[c] / float32(hf.lfg.hfMetadata.hfMultiplier[pos.Y][pos.X])
			pixelGroupY := sGroupY << 3
			pixelGroupX := sGroupX << 3
			qbc := qbclut[c]
			for y := int32(0); y < tt.pixelHeight; y++ {
				for x := int32(0); x < tt.pixelWidth; x++ {
					if y < tt.dctSelectHeight && x < tt.dctSelectWidth {
						continue
					}
					pY := pixelGroupY + y
					pX := pixelGroupX + x
					coeff := hf.quantizedCoeffs[c][pY][pX]
					var quant float32
					if coeff > -2 && coeff < 2 {
						quant = qbc[coeff+1]
					} else {
						quant = float32(coeff) - matrix.QuantBiasNumerator/float32(coeff)
					}
					var wy int32
					if flip {
						wy = x
					} else {
						wy = y
					}
					wx := x ^ y ^ wy
					hf.dequantHFCoeff[c][pY][pX] = quant * sfc * w3[wy][wx]
				}
			}
		}
	}
	return nil
}

func (hf *HFCoefficients) chromaFromLuma() error {

	header := hf.frame.getFrameHeader()
	xMatch := slices.ContainsFunc(header.jpegUpsamplingX, func(x int32) bool { return x != 0 })
	yMatch := slices.ContainsFunc(header.jpegUpsamplingY, func(y int32) bool { return y != 0 })
	if xMatch || yMatch {
		return nil
	}

	lfc := hf.frame.getLFGlobal().lfChanCorr
	xFactorHF := hf.lfg.hfMetadata.hfStreamBuffer[0]
	bFactorHF := hf.lfg.hfMetadata.hfStreamBuffer[1]
	xFactors := util.MakeMatrix2D[float32](len(xFactorHF), len(xFactorHF[0]))
	bFactors := util.MakeMatrix2D[float32](len(bFactorHF), len(bFactorHF[0]))

	for i := 0; i < len(hf.blocks); i++ {
		pos := hf.blocks[i]
		if pos == nil {
			continue
		}
		tt := hf.lfg.hfMetadata.dctSelect[pos.Y][pos.X]
		pPosY := pos.Y << 3
		pPosX := pos.X << 3
		for iy := int32(0); iy < tt.pixelHeight; iy++ {
			y := pPosY + iy
			fy := y >> 6
			by := fy<<6 == y
			xF := xFactors[fy]
			bF := bFactors[fy]
			hfX := xFactorHF[fy]
			hfB := bFactorHF[fy]
			for ix := int32(0); ix < tt.pixelWidth; ix++ {
				x := pPosX + ix
				fx := x >> 6
				var kX float32
				var kB float32
				if by && fx<<6 == x {
					kX = lfc.baseCorrelationX + float32(hfX[fx])/float32(lfc.colorFactor)
					kB = lfc.baseCorrelationB + float32(hfB[fx])/float32(lfc.colorFactor)
					xF[fx] = kX
					bF[fx] = kB
				} else {
					kX = xF[fx]
					kB = bF[fx]
				}
				dequantY := hf.dequantHFCoeff[1][y&0xFF][x&0xFF]
				hf.dequantHFCoeff[0][y&0xFF][x&0xFF] += kX * dequantY
				hf.dequantHFCoeff[2][y&0xFF][x&0xFF] += kB * dequantY
			}
		}
	}

	return nil
}

func (hf *HFCoefficients) finalizeLLF() error {

	scratchBlock := util.MakeMatrix3D[float32](2, 32, 32)
	header := hf.frame.getFrameHeader()
	for i := 0; i < len(hf.blocks); i++ {
		posInLfg := hf.blocks[i]
		if posInLfg == nil {
			continue
		}
		tt := hf.lfg.hfMetadata.dctSelect[posInLfg.Y][posInLfg.X]
		groupY := posInLfg.Y - hf.groupPos.Y
		groupX := posInLfg.X - hf.groupPos.X
		for c := 0; c < 3; c++ {
			sGroupY := groupY >> header.jpegUpsamplingY[c]
			sGroupX := groupX >> header.jpegUpsamplingX[c]
			if groupY != sGroupY<<header.jpegUpsamplingY[c] ||
				groupX != sGroupX<<header.jpegUpsamplingX[c] {
				continue
			}
			pixelGroupY := sGroupY << 3
			pixelGroupX := sGroupX << 3
			sLfgY := posInLfg.Y >> header.jpegUpsamplingY[c]
			sLfgX := posInLfg.X >> header.jpegUpsamplingX[c]

			dqlf := hf.lfg.lfCoeff.dequantLFCoeff[c]
			dq := hf.dequantHFCoeff[c]
			if err := util.ForwardDCT2D(dqlf, dq, util.Point{X: sLfgX, Y: sLfgY},
				util.Point{X: pixelGroupX, Y: pixelGroupY},
				tt.getDctSelectSize(), scratchBlock[0], scratchBlock[1], false); err != nil {
				return err
			}

			for y := int32(0); y < tt.dctSelectHeight; y++ {
				dqy := dq[y+pixelGroupY]

				llfy := tt.llfScale[y]
				for x := int32(0); x < tt.dctSelectWidth; x++ {
					dqy[x+pixelGroupX] *= llfy[x]
				}
			}
		}
	}

	return nil
}

func getPredictedNonZeros(nonZeros [][][]int32, c int, y int32, x int32) int32 {
	if x == 0 && y == 0 {
		return 32
	}
	if x == 0 {
		return nonZeros[c][y-1][0]
	}

	if y == 0 {
		return nonZeros[c][0][x-1]
	}

	return (nonZeros[c][y-1][x] + nonZeros[c][y][x-1] + 1) >> 1
}
