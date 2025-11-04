package frame

import (
	"errors"
	"fmt"

	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

var (
	AFV_BASIS = [][]float32{
		{0.25, 0.25, 0.25, 0.25, 0.25, 0.25, 0.25, 0.25, 0.25, 0.25, 0.25, 0.25, 0.25, 0.25, 0.25, 0.25},
		{0.876902929799142, 0.2206518106944235, -0.10140050393753763, -0.1014005039375375, 0.2206518106944236, -0.10140050393753777, -0.10140050393753772, -0.10140050393753763,
			-0.10140050393753758, -0.10140050393753769, -0.1014005039375375, -0.10140050393753768, -0.10140050393753768,
			-0.10140050393753759, -0.10140050393753763, -0.10140050393753741},
		{0.0, 0.0, 0.40670075830260755, 0.44444816619734445, 0.0, 0.0, 0.19574399372042936, 0.2929100136981264, -0.40670075830260716,
			-0.19574399372042872, 0.0, 0.11379074460448091, -0.44444816619734384, -0.29291001369812636,
			-0.1137907446044814, 0.0},
		{0.0, 0.0, -0.21255748058288748, 0.3085497062849767, 0.0, 0.4706702258572536, -0.1621205195722993,
			0.0, -0.21255748058287047, -0.16212051957228327, -0.47067022585725277, -0.1464291867126764,
			0.3085497062849487, 0.0, -0.14642918671266536, 0.4251149611657548},
		{0.0, -0.7071067811865474, 0.0, 0.0, 0.7071067811865476, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
		{-0.4105377591765233, 0.6235485373547691, -0.06435071657946274, -0.06435071657946266, 0.6235485373547694,
			-0.06435071657946284, -0.0643507165794628, -0.06435071657946274, -0.06435071657946272, -0.06435071657946279,
			-0.06435071657946266, -0.06435071657946277, -0.06435071657946277, -0.06435071657946273, -0.06435071657946274,
			-0.0643507165794626},
		{0.0, 0.0, -0.4517556589999482, 0.15854503551840063, 0.0, -0.04038515160822202,
			0.0074182263792423875, 0.39351034269210167, -0.45175565899994635, 0.007418226379244351, 0.1107416575309343,
			0.08298163094882051, 0.15854503551839705, 0.3935103426921022, 0.0829816309488214, -0.45175565899994796},
		{0.0, 0.0, -0.304684750724869, 0.5112616136591823, 0.0, 0.0, -0.290480129728998, -0.06578701549142804,
			0.304684750724884, 0.2904801297290076, 0.0, -0.23889773523344604, -0.5112616136592012, 0.06578701549142545,
			0.23889773523345467, 0.0},
		{0.0, 0.0, 0.3017929516615495, 0.25792362796341184, 0.0, 0.16272340142866204,
			0.09520022653475037, 0.0, 0.3017929516615503, 0.09520022653475055, -0.16272340142866173, -0.35312385449816297,
			0.25792362796341295, 0.0, -0.3531238544981624, -0.6035859033230976},
		{0.0, 0.0, 0.40824829046386274, 0.0, 0.0,
			0.0, 0.0, -0.4082482904638628, -0.4082482904638635, 0.0, 0.0, -0.40824829046386296, 0.0, 0.4082482904638634,
			0.408248290463863, 0.0},
		{0.0, 0.0, 0.1747866975480809, 0.0812611176717539, 0.0, 0.0, -0.3675398009862027,
			-0.307882213957909, -0.17478669754808135, 0.3675398009862011, 0.0, 0.4826689115059883, -0.08126111767175039,
			0.30788221395790305, -0.48266891150598584, 0.0},
		{0.0, 0.0, -0.21105601049335784, 0.18567180916109802, 0.0, 0.0,
			0.49215859013738733, -0.38525013709251915, 0.21105601049335806, -0.49215859013738905, 0.0, 0.17419412659916217,
			-0.18567180916109904, 0.3852501370925211, -0.1741941265991621, 0.0}, {0.0, 0.0, -0.14266084808807264,
			-0.3416446842253372, 0.0, 0.7367497537172237, 0.24627107722075148, -0.08574019035519306, -0.14266084808807344,
			0.24627107722075137, 0.14883399227113567, -0.04768680350229251, -0.3416446842253373, -0.08574019035519267,
			-0.047686803502292804, -0.14266084808807242},
		{0.0, 0.0, -0.13813540350758585, 0.3302282550303788, 0.0,
			0.08755115000587084, -0.07946706605909573, -0.4613374887461511, -0.13813540350758294, -0.07946706605910261,
			0.49724647109535086, 0.12538059448563663, 0.3302282550303805, -0.4613374887461554, 0.12538059448564315,
			-0.13813540350758452},
		{0.0, 0.0, -0.17437602599651067, 0.0702790691196284, 0.0, -0.2921026642334881,
			0.3623817333531167, 0.0, -0.1743760259965108, 0.36238173335311646, 0.29210266423348785, -0.4326608024727445,
			0.07027906911962818, 0.0, -0.4326608024727457, 0.34875205199302267},
		{0.0, 0.0, 0.11354987314994337,
			-0.07417504595810355, 0.0, 0.19402893032594343, -0.435190496523228, 0.21918684838857466, 0.11354987314994257,
			-0.4351904965232251, 0.5550443808910661, -0.25468277124066463, -0.07417504595810233, 0.2191868483885728,
			-0.25468277124066413, 0.1135498731499429}}
)

type PassGroup struct {
	modularPassGroupBuffer [][][]int32
	modularStream          ModularStreamer
	frame                  *Frame
	groupID                uint32
	passID                 uint32
	hfCoefficients         *HFCoefficients
	lfg                    *LFGroup
}

func NewPassGroupWithReader(reader jxlio.BitReader, frame *Frame, pass uint32, group uint32, replacedChannels []ModularChannel) (*PassGroup, error) {

	pg := &PassGroup{}
	pg.frame = frame
	pg.groupID = group
	pg.passID = pass
	if frame.Header.Encoding == VARDCT {
		coeff, err := NewHFCoefficientsWithReader(reader, frame, pass, group)
		if err != nil {
			return nil, err
		}

		// have confirmed that QuantizedCoeffs and DequantizedCoeffs are the same
		// as JXLatte at least. Have not checked other members in HFCoefficients.
		pg.hfCoefficients = coeff
	} else {
		pg.hfCoefficients = nil
	}

	stream, err := NewModularStreamWithStreamIndex(reader, frame, int(18+3*frame.numLFGroups+frame.numGroups*pass+group), replacedChannels)
	if err != nil {
		return nil, err
	}

	pg.modularStream = stream
	err = stream.decodeChannels(reader, false)
	if err != nil {
		return nil, err
	}

	pg.lfg = frame.getLFGroupForGroup(int32(group))

	return pg, nil
}

func (g *PassGroup) invertVarDCT(frameBuffer [][][]float32, prev *PassGroup) error {

	header := g.frame.Header
	if prev != nil {
		panic("not implemented")
	}

	// differs from jxlatte
	if err := g.hfCoefficients.bakeDequantizedCoeffs(); err != nil {
		return err
	}

	groupLocation := g.frame.getGroupLocation(int32(g.groupID))
	groupLocation.Y <<= 8
	groupLocation.X <<= 8

	coeffs := g.hfCoefficients.dequantHFCoeff
	scratchBlock := util.MakeMatrix3D[float32](5, 256, 256)
	for i := 0; i < len(g.hfCoefficients.blocks); i++ {
		posInLFG := g.hfCoefficients.blocks[i]
		if posInLFG == nil {
			continue
		}
		tt := g.lfg.hfMetadata.dctSelect[posInLFG.Y][posInLFG.X]
		groupY := posInLFG.Y - g.hfCoefficients.groupPos.Y
		groupX := posInLFG.X - g.hfCoefficients.groupPos.X
		for c := 0; c < 3; c++ {
			sGroupY := groupY >> header.jpegUpsamplingY[c]
			sGroupX := groupX >> header.jpegUpsamplingX[c]
			if sGroupY<<header.jpegUpsamplingY[c] != groupY ||
				sGroupX<<header.jpegUpsamplingX[c] != groupX {
				continue
			}
			ppg := util.Point{
				X: sGroupX << 3,
				Y: sGroupY << 3,
			}
			ppf := util.Point{
				X: ppg.X + (groupLocation.X >> header.jpegUpsamplingX[c]),
				Y: ppg.Y + (groupLocation.Y >> header.jpegUpsamplingY[c]),
			}

			lfs := make([]float32, 2)
			var coeff0 float32
			var coeff1 float32
			switch tt.transformMethod {
			case METHOD_DCT:
				if err := util.InverseDCT2D(coeffs[c], frameBuffer[c], ppg, ppf, tt.getPixelSize(), scratchBlock[0], scratchBlock[1], false); err != nil {
					return err
				}
				break
			case METHOD_DCT8_4:
				coeff0 = coeffs[c][ppg.Y][ppg.X]
				coeff1 = coeffs[c][ppg.Y+1][ppg.X]
				lfs[0] = coeff0 + coeff1
				lfs[1] = coeff0 - coeff1
				for x := int32(0); x < 2; x++ {
					scratchBlock[0][0][0] = lfs[x]
					for iy := int32(0); iy < 4; iy++ {
						startX := int32(0)
						if iy == 0 {
							startX = 1
						}
						for ix := startX; ix < 8; ix++ {
							scratchBlock[0][iy][ix] = coeffs[c][ppg.Y+x+iy*2][ppg.X+ix]
						}
					}
					ppf2 := util.Point{
						X: ppf.X,
						Y: ppf.Y,
					}
					ppf2.X += x << 2
					if err := util.InverseDCT2D(scratchBlock[0], frameBuffer[c], util.ZERO, ppf2,
						util.Dimension{Height: 4, Width: 8}, scratchBlock[1], scratchBlock[2], true); err != nil {
						return err
					}
				}
				break

			case METHOD_DCT4_8:
				coeff0 = coeffs[c][ppg.Y][ppg.X]
				coeff1 = coeffs[c][ppg.Y+1][ppg.X]
				lfs[0] = coeff0 + coeff1
				lfs[1] = coeff0 - coeff1
				for y := int32(0); y < 2; y++ {
					scratchBlock[0][0][0] = lfs[y]
					for iy := int32(0); iy < 4; iy++ {
						startX := int32(0)
						if iy == 0 {
							startX = 1
						}
						for ix := startX; ix < 8; ix++ {
							scratchBlock[0][iy][ix] = coeffs[c][ppg.Y+y+iy*2][ppg.X+ix]
						}
					}
					ppf2 := util.Point{
						X: ppf.X,
						Y: ppf.Y,
					}
					ppf2.Y += y << 2
					if err := util.InverseDCT2D(scratchBlock[0], frameBuffer[c], util.ZERO, ppf2,
						util.Dimension{Height: 4, Width: 8}, scratchBlock[1], scratchBlock[2], false); err != nil {
						return err
					}
				}
				break

			case METHOD_AFV:
				//displayBuffer("before", frameBuffer[c])
				// FIXME(kpfaulkner) there is some bug in here compared to JXLatte, but
				// have yet to figure it out.
				if err := g.invertAFV(coeffs[c], frameBuffer[c], tt, ppg, ppf, scratchBlock); err != nil {
					return err
				}
				//displayBuffer("after", frameBuffer[c])
				break
			case METHOD_DCT2:
				g.auxDCT2(coeffs[c], scratchBlock[0], ppg, util.ZERO, 2)
				g.auxDCT2(scratchBlock[0], scratchBlock[1], util.ZERO, util.ZERO, 4)
				g.auxDCT2(scratchBlock[1], frameBuffer[c], util.ZERO, ppf, 8)
				break
			case METHOD_HORNUSS:
				g.auxDCT2(coeffs[c], scratchBlock[1], ppg, util.ZERO, 2)
				for y := int32(0); y < 2; y++ {
					for x := int32(0); x < 2; x++ {
						blockLF := scratchBlock[1][y][x]
						residual := float32(0.0)
						for iy := int32(0); iy < 4; iy++ {
							ixTemp := int32(0)
							if iy == 0 {
								ixTemp = 1
							}
							for ix := ixTemp; ix < 4; ix++ {
								residual += coeffs[c][ppg.Y+y+iy*2][ppg.X+x+ix*2]
							}
						}
						scratchBlock[0][4*y+1][4*x+1] = blockLF - residual*0.0625
						for iy := int32(0); iy < 4; iy++ {
							for ix := int32(0); ix < 4; ix++ {
								if ix == 1 && iy == 1 {
									continue
								}
								scratchBlock[0][4*y+iy][x*4+ix] = coeffs[c][ppg.Y+y+iy*2][ppg.X+x+ix*2] + scratchBlock[0][4*y+1][4*x+1]
							}
						}
						scratchBlock[0][4*y][4*x] = coeffs[c][ppg.Y+y+2][ppg.X+x+2] + scratchBlock[0][4*y+1][4*x+1]
					}
				}
				layBlock(scratchBlock[0], frameBuffer[c], util.ZERO, ppf, tt.getPixelSize())
			case METHOD_DCT4:
				panic("not implemented")
			default:
				return errors.New("transform not implemented")
			}
		}
	}
	return nil
}

func layBlock(block [][]float32, buffer [][]float32, inPos util.Point, outPos util.Point, inSize util.Dimension) {
	for y := int32(0); y < int32(inSize.Height); y++ {

		// Make sure to specify end position in X slice. ie the outPos.X + inSize.Width etc.
		// otherwise some images end up with black unpopulated sections.
		copy(buffer[y+outPos.Y][outPos.X:outPos.X+int32(inSize.Width)], block[y+inPos.Y][inPos.X:inPos.X+int32(inSize.Width)])
	}
}

func (g *PassGroup) invertAFV(coeffs [][]float32, buffer [][]float32, tt *TransformType, ppg util.Point, ppf util.Point, scratchBlock [][][]float32) error {

	// some debugging logic here... there's a bug in here somewhere :/
	if false {
		fmt.Printf("invertAFV coeffs:\n")
		for y := 0; y < len(coeffs); y++ {
			total := float32(0.0)
			for x := 0; x < len(coeffs[y]); x++ {
				//fmt.Printf("%0.10f ", coeffs[i][j])
				total += coeffs[y][x]
			}
			if total != 0.0 {
				// super inefficient... but dont care.
				fmt.Printf("coord y=%d non zero %0.10f\n", y, total)
				for x := 0; x < len(coeffs[y]); x++ {
					//fmt.Printf("%0.10f ", coeffs[y][x])
				}
				//fmt.Printf("\n")
			}
		}
		fmt.Printf("==========\n")
	}

	scratchBlock[0][0][0] = (coeffs[ppg.Y][ppg.X] + coeffs[ppg.Y+1][ppg.X] + coeffs[ppg.Y][ppg.X+1]) * 4.0
	for iy := int32(0); iy < 4; iy++ {
		startX := int32(0)
		if iy == 0 {
			startX = 1
		}
		for ix := startX; ix < 4; ix++ {
			scratchBlock[0][iy][ix] = coeffs[ppg.Y+iy*2][ppg.X+ix*2]
		}
	}

	var flipX int32
	var flipY int32
	if tt == AFV2 || tt == AFV3 {
		flipY = 1
	}
	if tt == AFV1 || tt == AFV3 {
		flipX = 1
	}
	totalSample := float32(0)
	for iy := 0; iy < 4; iy++ {
		for ix := 0; ix < 4; ix++ {
			sample := float32(0.0)
			for j := 0; j < 16; j++ {
				jy := j >> 2
				jx := j & 0b11
				sample += scratchBlock[0][jy][jx] * AFV_BASIS[j][iy*4+ix]
			}
			scratchBlock[1][iy][ix] = sample
			totalSample += sample
		}
	}

	for iy := int32(0); iy < 4; iy++ {
		for ix := int32(0); ix < 4; ix++ {
			xpos := ix
			ypos := iy
			if flipY == 1 {
				ypos = 3 - iy
			}
			if flipX == 1 {
				xpos = 3 - ix
			}
			buffer[ppf.Y+flipY*4+iy][ppf.X+flipX*4+ix] = scratchBlock[1][ypos][xpos]
		}
	}

	scratchBlock[0][0][0] = coeffs[ppg.Y][ppg.X] + coeffs[ppg.Y+1][ppg.X] - coeffs[ppg.Y][ppg.X+1]
	for iy := int32(0); iy < 4; iy++ {
		startX := int32(0)
		if iy == 0 {
			startX = 1
		}
		for ix := startX; ix < 4; ix++ {
			scratchBlock[0][iy][ix] = coeffs[ppg.Y+iy*2][ppg.X+ix*2+1]
		}
	}

	if err := util.InverseDCT2D(scratchBlock[0], scratchBlock[1], util.ZERO, util.ZERO,
		util.Dimension{4, 4}, scratchBlock[2], scratchBlock[3], false); err != nil {
		return err
	}

	for iy := int32(0); iy < 4; iy++ {
		for ix := int32(0); ix < 4; ix++ {
			xx := int32(4)
			if flipX == 1 {
				xx = 0
			}
			buffer[ppf.Y+flipY*4+iy][ppf.X+xx+ix] = scratchBlock[1][ix][iy]
		}
	}

	scratchBlock[0][0][0] = coeffs[ppg.Y][ppg.X] - coeffs[ppg.Y+1][ppg.X]
	for iy := int32(0); iy < 4; iy++ {
		startX := int32(0)
		if iy == 0 {
			startX = 1
		}
		for ix := startX; ix < 8; ix++ {
			scratchBlock[0][iy][ix] = coeffs[ppg.Y+1+iy*2][ppg.X+ix]
		}
	}

	if err := util.InverseDCT2D(scratchBlock[0], scratchBlock[1], util.ZERO, util.ZERO, util.Dimension{Height: 4, Width: 8},
		scratchBlock[2], scratchBlock[3], false); err != nil {
		return err
	}

	for iy := int32(0); iy < 4; iy++ {

		for ix := int32(0); ix < 8; ix++ {
			yy := int32(4)
			if flipY == 1 {
				yy = 0
			}
			buffer[ppf.Y+yy+iy][ppf.X+ix] = scratchBlock[1][iy][ix]
		}
	}
	return nil
}

func (g *PassGroup) auxDCT2(coeffs [][]float32, result [][]float32, p util.Point, ps util.Point, s int32) {
	g.layBlock(coeffs, result, p, ps, util.Dimension{Height: 8, Width: 8})

	num := s / 2
	for iy := int32(0); iy < num; iy++ {
		for ix := int32(0); ix < num; ix++ {
			c00 := coeffs[p.Y+iy][p.X+ix]
			c01 := coeffs[p.Y+iy][p.X+ix+num]
			c10 := coeffs[p.Y+iy+num][p.X+ix]
			c11 := coeffs[p.Y+iy+num][p.X+ix+num]
			r00 := c00 + c01 + c10 + c11
			r01 := c00 + c01 - c10 - c11
			r10 := c00 - c01 + c10 - c11
			r11 := c00 - c01 - c10 + c11
			result[ps.Y+iy*2][ps.X+ix*2] = r00
			result[ps.Y+iy*2][ps.X+ix*2+1] = r01
			result[ps.Y+iy*2+1][ps.X+ix*2] = r10
			result[ps.Y+iy*2+1][ps.X+ix*2+1] = r11
		}
	}
}

func (g *PassGroup) layBlock(block [][]float32, buffer [][]float32, inPos util.Point, outPos util.Point, inSize util.Dimension) {
	for y := int32(0); y < int32(inSize.Height); y++ {
		copy(buffer[y+outPos.Y][outPos.X:outPos.X+int32(inSize.Width)], block[y+inPos.Y][inPos.X:inPos.X+int32(inSize.Width)])
	}
}
