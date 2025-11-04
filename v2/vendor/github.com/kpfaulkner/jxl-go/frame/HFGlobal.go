package frame

import (
	"errors"
	"fmt"
	"math"

	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

var (
	defaultParams []DCTParam

	dct4x4params = [][]float64{
		{2200, 0.0, 0.0, 0.0},
		{392, 0.0, 0.0, 0.0},
		{112, -0.25, -0.25, -0.5}}

	dct4x8params = [][]float64{
		{2198.050556016380522, -0.96269623020744692, -0.76194253026666783, -0.6551140670773547},
		{764.3655248643528689, -0.92630200888366945, -0.9675229603596517, -0.27845290869168118},
		{527.107573587542228, -1.4594385811273854, -1.450082094097871593, -1.5843722511996204}}

	afvFreqs = []float64{0, 0, 0.8517778890324296, 5.37778436506804,
		0, 0, 4.734747904497923, 5.449245381693219, 1.6598270267479331, 4, 7.275749096817861,
		10.423227632456525, 2.662932286148962, 7.630657783650829, 8.962388608184032, 12.97166202570235}

	seqA = []float64{-1.025, -0.78, -0.65012, -0.19041574084286472, -0.20819395464, -0.421064, -0.32733845535848671}
	seqB = []float64{-0.3041958212306401, -0.3633036457487539, -0.35660379990111464, -0.3443074455424403, -0.33699592683512467, -0.30180866526242109, -0.27321683125358037}
	seqC = []float64{-1.2, -1.2, -0.8, -0.7, -0.7, -0.4, -0.5}
)

type HFGlobal struct {
	params       []DCTParam
	weights      [][][][]float32
	numHFPresets int32
}

func init() {
	setupDefaultParams()
}

func setupDefaultParams() {
	defaultParams = make([]DCTParam, 17)
	defaultParams[0] = DCTParam{
		dctParam: [][]float64{
			{3150.0, 0.0, -0.4, -0.4, -0.4, -2.0},
			{560.0, 0.0, -0.3, -0.3, -0.3, -0.3},
			{512.0, -2.0, -1.0, 0.0, -1.0, -2.0}},
		param:       nil,
		mode:        MODE_DCT,
		denominator: 1,
		params4x4:   nil,
	}
	defaultParams[1] = DCTParam{
		dctParam: nil,
		param: [][]float32{
			{280.0, 3160.0, 3160.0},
			{60.0, 864.0, 864.0},
			{18.0, 200.0, 200.0}},
		mode:        MODE_HORNUSS,
		denominator: 1,
		params4x4:   nil,
	}
	defaultParams[2] = DCTParam{
		dctParam: nil,
		param: [][]float32{
			{3840.0, 2560.0, 1280.0, 640.0, 480.0, 300.0},
			{960.0, 640.0, 320.0, 180.0, 140.0, 120.0},
			{640.0, 320.0, 128.0, 64.0, 32.0, 16.0}},
		mode:        MODE_DCT2,
		denominator: 1,
		params4x4:   nil,
	}
	defaultParams[3] = DCTParam{
		dctParam: dct4x4params,
		param: [][]float32{
			{1.0, 1.0},
			{1.0, 1.0},
			{1.0, 1.0}},
		mode:        MODE_DCT4,
		denominator: 1,
		params4x4:   dct4x4params,
	}
	defaultParams[4] = DCTParam{
		dctParam: [][]float64{
			{8996.8725711814115328, -1.3000777393353804, -0.49424529824571225, -0.439093774457103443, -0.6350101832695744, -0.90177264050827612, -1.6162099239887414},
			{3191.48366296844234752, -0.67424582104194355, -0.80745813428471001, -0.44925837484843441, -0.35865440981033403, -0.31322389111877305, -0.37615025315725483},
			{1157.50408145487200256, -2.0531423165804414, -1.4, -0.50687130033378396, -0.42708730624733904, -1.4856834539296244, -4.9209142884401604}},
		param:       nil,
		mode:        MODE_DCT,
		denominator: 1,
		params4x4:   nil,
	}
	defaultParams[5] = DCTParam{
		dctParam: [][]float64{
			{15718.40830982518931456, -1.025, -0.98, -0.9012, -0.4, -0.48819395464, -0.421064, -0.27},
			{7305.7636810695983104, -0.8041958212306401, -0.7633036457487539, -0.55660379990111464, -0.49785304658857626, -0.43699592683512467, -0.40180866526242109, -0.27321683125358037},
			{3803.53173721215041536, -3.060733579805728, -2.0413270132490346, -2.0235650159727417, -0.5495389509954993, -0.4, -0.4, -0.3}},
		param:       nil,
		mode:        MODE_DCT,
		denominator: 1,
		params4x4:   nil,
	}
	defaultParams[6] = DCTParam{
		dctParam: [][]float64{
			{7240.7734393502, -0.7, -0.7, -0.2, -0.2, -0.2, -0.5},
			{1448.15468787004, -0.5, -0.5, -0.5, -0.2, -0.2, -0.2},
			{506.854140754517, -1.4, -0.2, -0.5, -0.5, -1.5, -3.6}},
		param:       nil,
		mode:        MODE_DCT,
		denominator: 1,
		params4x4:   nil,
	}
	defaultParams[7] = DCTParam{
		dctParam: [][]float64{
			{16283.2494710648897, -1.7812845336559429, -1.6309059012653515, -1.0382179034313539, -0.85, -0.7, -0.9, -1.2360638576849587},
			{5089.15750884921511936, -0.320049391452786891, -0.35362849922161446, -0.30340000000000003, -0.61, -0.5, -0.5, -0.6},
			{3397.77603275308720128, -0.321327362693153371, -0.34507619223117997, -0.70340000000000003, -0.9, -1.0, -1.0, -1.1754605576265209}},
		param:       nil,
		mode:        MODE_DCT,
		denominator: 1,
		params4x4:   nil,
	}
	defaultParams[8] = DCTParam{
		dctParam: [][]float64{
			{13844.97076442300573, -0.97113799999999995, -0.658, -0.42026, -0.22712, -0.2206, -0.226, -0.6},
			{4798.964084220744293, -0.61125308982767057, -0.83770786552491361, -0.79014862079498627, -0.2692727459704829, -0.38272769465388551, -0.22924222653091453, -0.20719098826199578},
			{1807.236946760964614, -1.2, -1.2, -0.7, -0.7, -0.7, -0.4, -0.5}},
		param:       nil,
		mode:        MODE_DCT,
		denominator: 1,
		params4x4:   nil,
	}
	defaultParams[9] = DCTParam{
		dctParam:    dct4x8params,
		param:       [][]float32{{1.0}, {1.0}, {1.0}},
		mode:        MODE_DCT4_8,
		denominator: 1,
		params4x4:   nil,
	}
	defaultParams[10] = DCTParam{
		dctParam: dct4x8params,
		param: [][]float32{
			{3072, 3072, 256, 256, 256, 414, 0.0, 0.0, 0.0},
			{1024, 1024, 50.0, 50.0, 50.0, 58, 0.0, 0.0, 0.0},
			{384, 384, 12.0, 12.0, 12.0, 22, -0.25, -0.25, -0.25}},
		mode:        MODE_AFV,
		denominator: 1,
		params4x4:   dct4x4params,
	}
	defaultParams[11] = DCTParam{
		dctParam: [][]float64{
			append([]float64{23966.1665298448605}, seqA[:]...),
			append([]float64{8380.19148390090414}, seqB[:]...),
			append([]float64{4493.02378009847706}, seqC[:]...),
		},
		param:       nil,
		mode:        MODE_DCT,
		denominator: 1,
		params4x4:   nil,
	}
	defaultParams[12] = DCTParam{
		dctParam: [][]float64{
			append([]float64{15358.89804933239925}, seqA[:]...),
			append([]float64{5597.360516150652990}, seqB[:]...),
			append([]float64{2919.961618960011210}, seqC[:]...),
		},
		param:       nil,
		mode:        MODE_DCT,
		denominator: 1,
		params4x4:   nil,
	}
	defaultParams[13] = DCTParam{
		dctParam: [][]float64{
			append([]float64{47932.3330596897210}, seqA[:]...),
			append([]float64{16760.38296780180828}, seqB[:]...),
			append([]float64{8986.04756019695412}, seqC[:]...),
		},
		param:       nil,
		mode:        MODE_DCT,
		denominator: 1,
		params4x4:   nil,
	}
	defaultParams[14] = DCTParam{
		dctParam: [][]float64{
			append([]float64{30717.796098664792}, seqA[:]...),
			append([]float64{11194.72103230130598}, seqB[:]...),
			append([]float64{5839.92323792002242}, seqC[:]...),
		},
		param:       nil,
		mode:        MODE_DCT,
		denominator: 1,
		params4x4:   nil,
	}
	defaultParams[15] = DCTParam{
		dctParam: [][]float64{
			append([]float64{95864.6661193794420}, seqA[:]...),
			append([]float64{33520.76593560361656}, seqB[:]...),
			append([]float64{17972.09512039390824}, seqC[:]...),
		},
		param:       nil,
		mode:        MODE_DCT,
		denominator: 1,
		params4x4:   nil,
	}
	defaultParams[16] = DCTParam{
		dctParam: [][]float64{
			append([]float64{61435.5921973295970}, seqA[:]...),
			append([]float64{24209.44206460261196}, seqB[:]...),
			append([]float64{12979.84647584004484}, seqC[:]...),
		},
		param:       nil,
		mode:        MODE_DCT,
		denominator: 1,
		params4x4:   nil,
	}
}

func NewHFGlobalWithReader(reader jxlio.BitReader, frame *Frame) (*HFGlobal, error) {
	hf := &HFGlobal{}

	quantAllDefault, err := reader.ReadBool()
	if err != nil {
		return nil, err
	}
	if quantAllDefault {
		hf.params = defaultParams
	} else {
		hf.params = make([]DCTParam, 17)
		for i := int32(0); i < 17; i++ {
			if err := hf.setupDCTParam(reader, frame, i); err != nil {
				return nil, err
			}
		}
	}

	hf.weights = util.MakeMatrix4D[float32](17, 3, 0, 0)
	for i := 0; i < 17; i++ {
		if err := hf.generateWeights(i); err != nil {
			return nil, err
		}
	}

	//hf.totalWeights()
	numPresets, err := reader.ReadBits(uint32(util.CeilLog1p(frame.numGroups - 1)))
	if err != nil {
		return nil, err
	}
	hf.numHFPresets = 1 + int32(numPresets)
	return hf, nil
}

func (hfg *HFGlobal) getHFPresets() int32 {
	return hfg.numHFPresets
}

func (hfg *HFGlobal) totalWeights() {

	var total float32 = 0
	for index := 0; index < 17; index++ {
		for c := 0; c < len(hfg.weights[index]); c++ {
			for y := 0; y < len(hfg.weights[index][c]); y++ {
				for x := 0; x < len(hfg.weights[index][c][y]); x++ {
					total += hfg.weights[index][c][y][x]
				}
			}
		}
	}
	fmt.Printf("total weight %f\n", total)
}

func (hfg *HFGlobal) displayWeights() {
	for index := 0; index < 17; index++ {
		for c := 0; c < len(hfg.weights[index]); c++ {
			for y := 0; y < len(hfg.weights[index][c]); y++ {
				for x := 0; x < len(hfg.weights[index][c][y]); x++ {
					print(hfg.weights[index][c][y][x], " ")
				}
				println()
			}
			println()
		}
	}
}

func (hfg *HFGlobal) displaySpecificWeights(index int, c int, y int) {
	for x := 0; x < len(hfg.weights[index][c][y]); x++ {
		fmt.Printf("%f ", hfg.weights[index][c][y][x])
	}
	fmt.Printf("\n")

}

func (hfg *HFGlobal) setupDCTParam(reader jxlio.BitReader, frame *Frame, index int32) error {
	encodingMode, err := reader.ReadBits(3)
	if err != nil {
		return err
	}
	_, err = validateIndex(index, int32(encodingMode))
	if err != nil {
		return err
	}

	switch encodingMode {
	case MODE_LIBRARY:
		hfg.params[index] = defaultParams[index]
		break
	case MODE_HORNUSS:
		m := util.MakeMatrix2D[float32](3, 3)
		for y := int32(0); y < 3; y++ {
			for x := int32(0); x < 3; x++ {
				mm, err := reader.ReadF16()
				if err != nil {
					return err
				}
				m[y][x] = 64.0 * mm
			}
		}
		hfg.params[index] = DCTParam{dctParam: nil, param: m, mode: MODE_HORNUSS, denominator: 1, params4x4: nil}
		break
	case MODE_DCT2:
		m := util.MakeMatrix2D[float32](3, 6)
		for y := int32(0); y < 3; y++ {
			for x := int32(0); x < 6; x++ {
				mm, err := reader.ReadF16()
				if err != nil {
					return err
				}
				m[y][x] = 64.0 * mm
			}
		}
		hfg.params[index] = DCTParam{dctParam: nil, param: m, mode: MODE_DCT2, denominator: 1, params4x4: nil}
		break

	case MODE_DCT4:
		m := util.MakeMatrix2D[float32](3, 2)
		for y := int32(0); y < 3; y++ {
			for x := int32(0); x < 2; x++ {
				mm, err := reader.ReadF16()
				if err != nil {
					return err
				}
				m[y][x] = 64.0 * mm
			}
		}
		dctParam, err := hfg.readDCTParams(reader)
		if err != nil {
			return err
		}
		hfg.params[index] = DCTParam{dctParam: dctParam, param: m, mode: MODE_DCT4, denominator: 1, params4x4: nil}
		break

	case MODE_DCT:
		dctParam, err := hfg.readDCTParams(reader)
		if err != nil {
			return err
		}
		hfg.params[index] = DCTParam{dctParam: dctParam, param: nil, mode: MODE_DCT, denominator: 1, params4x4: nil}
		break
	case MODE_RAW:
		den, err := reader.ReadF16()
		if err != nil {
			return err
		}

		var tt *TransformType
		if tt, err = getHorizontalTransformType(index); err != nil {
			return err
		}
		info := make([]ModularChannel, 3)
		info[0] = *NewModularChannelWithAllParams(tt.matrixHeight, tt.matrixWidth, 0, 0, false)
		info[1] = *NewModularChannelWithAllParams(tt.matrixHeight, tt.matrixWidth, 0, 0, false)
		info[2] = *NewModularChannelWithAllParams(tt.matrixHeight, tt.matrixWidth, 0, 0, false)

		stream, err := NewModularStreamWithStreamIndex(reader, frame, int(1+3*int32(frame.numLFGroups)+index), info)
		if err != nil {
			return err
		}
		if err = stream.decodeChannels(reader, false); err != nil {
			return err
		}
		m := util.MakeMatrix2D[float32](3, tt.matrixWidth*tt.matrixHeight)
		b := stream.getDecodedBuffer()
		for c := 0; c < 3; c++ {
			for y := 0; y < len(b[c]); y++ {
				for x := 0; x < len(b[c][y]); x++ {
					m[c][y*int(tt.matrixWidth)+x] = float32(b[c][y][x])
				}
			}
		}
		hfg.params[index] = DCTParam{dctParam: nil, param: m, mode: MODE_RAW, denominator: den, params4x4: nil}
		break
	case MODE_AFV:
		m := util.MakeMatrix2D[float32](3, 9)
		for y := int32(0); y < 3; y++ {
			for x := int32(0); x < 9; x++ {
				mm, err := reader.ReadF16()
				if err != nil {
					return err
				}
				m[y][x] = mm
				if x < 6 {
					m[y][x] *= 64.0
				}
			}
		}
		var d [][]float64
		if d, err = hfg.readDCTParams(reader); err != nil {
			return err
		}
		var f [][]float64
		if f, err = hfg.readDCTParams(reader); err != nil {
			return err
		}
		hfg.params[index] = DCTParam{dctParam: d, param: m, mode: MODE_AFV, denominator: 1, params4x4: f}
		break
	default:
		return errors.New("Invalid encoding mode")
	}
	return nil
}

func (hfg *HFGlobal) readDCTParams(reader jxlio.BitReader) ([][]float64, error) {

	var numParams uint64
	var err error
	if numParams, err = reader.ReadBits(4); err != nil {
		return nil, err
	}
	numParams++
	vals := util.MakeMatrix2D[float64](3, numParams)

	for c := 0; c < 3; c++ {
		for i := 0; i < int(numParams); i++ {
			var v float32
			if v, err = reader.ReadF16(); err != nil {
				return nil, err
			}
			vals[c][i] = float64(v)
		}
		vals[c][0] *= 64
	}
	return vals, nil
}

func (hfg *HFGlobal) generateWeights(index int) error {

	var tt *TransformType
	var err error
	if tt, err = getHorizontalTransformType(int32(index)); err != nil {
		return err
	}

	for c := 0; c < 3; c++ {
		var w [][]float32
		switch hfg.params[index].mode {
		case MODE_DCT:
			hfg.weights[index][c] = hfg.getDCTQuantWeights(tt.matrixHeight, tt.matrixWidth, hfg.params[index].dctParam[c])
			break
		case MODE_DCT4:
			hfg.weights[index][c] = util.MakeMatrix2D[float32](8, 8)
			w = hfg.getDCTQuantWeights(4, 4, hfg.params[index].dctParam[c])
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					hfg.weights[index][c][y][x] = w[y/2][x/2]
				}
			}
			hfg.weights[index][c][1][0] /= hfg.params[index].param[c][0]
			hfg.weights[index][c][0][1] /= hfg.params[index].param[c][0]
			hfg.weights[index][c][1][1] /= hfg.params[index].param[c][1]
			break
		case MODE_DCT2:
			w = util.MakeMatrix2D[float32](8, 8)
			w[0][0] = 1
			w[0][1] = hfg.params[index].param[c][0]
			w[1][0] = hfg.params[index].param[c][0]
			w[1][1] = hfg.params[index].param[c][1]
			for y := 0; y < 2; y++ {
				for x := 0; x < 2; x++ {
					w[y][x+2] = hfg.params[index].param[c][2]
					w[x+2][y] = hfg.params[index].param[c][2]
					w[y+2][x+2] = hfg.params[index].param[c][3]
				}
			}
			for y := 0; y < 4; y++ {
				for x := 0; x < 4; x++ {
					w[y][x+4] = hfg.params[index].param[c][4]
					w[x+4][y] = hfg.params[index].param[c][4]
					w[y+4][x+4] = hfg.params[index].param[c][5]
				}
			}
			hfg.weights[index][c] = w
			break
		case MODE_HORNUSS:
			w = util.MakeMatrix2D[float32](8, 8)
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					w[y][x] = hfg.params[index].param[c][0]
				}
			}
			w[1][1] = hfg.params[index].param[c][2]
			w[0][1] = hfg.params[index].param[c][1]
			w[1][0] = hfg.params[index].param[c][1]
			w[0][0] = 1.0
			hfg.weights[index][c] = w
			break
		case MODE_DCT4_8:
			hfg.weights[index][c] = util.MakeMatrix2D[float32](8, 8)
			w = hfg.getDCTQuantWeights(4, 8, hfg.params[index].dctParam[c])
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					hfg.weights[index][c][y][x] = w[y/2][x]
				}
			}
			hfg.weights[index][c][1][0] /= hfg.params[index].param[c][0]
			break
		case MODE_AFV:
			afv, err := hfg.getAFVTransformWeights(index, c)
			if err != nil {
				return err
			}
			hfg.weights[index][c] = afv
			break
		case MODE_RAW:
			hfg.weights[index][c] = util.MakeMatrix2D[float32](tt.matrixHeight, tt.matrixWidth)
			for y := int32(0); y < tt.matrixHeight; y++ {
				for x := int32(0); x < tt.matrixWidth; x++ {
					hfg.weights[index][c][y][x] = hfg.params[index].param[c][y*tt.matrixWidth+x] * hfg.params[index].denominator
				}
			}
			break
		default:
			return errors.New("Invalid mode")
		}
	}
	if hfg.params[index].mode != MODE_RAW {
		for c := 0; c < 3; c++ {
			for y := int32(0); y < tt.matrixHeight; y++ {
				for x := int32(0); x < tt.matrixWidth; x++ {
					if hfg.weights[index][c][y][x] < 0 || math.IsInf(float64(hfg.weights[index][c][y][x]), 0) {
						return errors.New("Invalid weight")
					}
					hfg.weights[index][c][y][x] = 1.0 / hfg.weights[index][c][y][x]
				}
			}
		}
	}
	return nil
}

func quantMult(v float32) float32 {
	if v >= 0 {
		return 1 + v
	}

	return 1 / (1 - v)
}

func (hfg *HFGlobal) getDCTQuantWeights(height int32, width int32, params []float64) [][]float32 {

	bands := make([]float32, len(params))
	bands[0] = float32(params[0])
	for i := 1; i < len(bands); i++ {
		bands[i] = bands[i-1] * quantMult(float32(params[i]))
	}

	weights := util.MakeMatrix2D[float32](height, width)
	scale := float32(len(bands)-1) / (math.Sqrt2 + 1e-6)
	for y := int32(0); y < height; y++ {
		dy := float32(y) * scale / float32(height-1)
		dy2 := dy * dy
		for x := int32(0); x < width; x++ {
			dx := float32(x) * scale / float32(width-1)
			dist := float32(math.Sqrt(float64(dx*dx + dy2)))
			weights[y][x] = interpolate(dist, bands)
		}
	}
	return weights
}

func interpolate(scaledPos float32, bands []float32) float32 {
	l := len(bands) - 1
	if l == 0 {
		return bands[0]
	}

	scaledIndex := int(scaledPos)
	if scaledIndex+1 > l {
		return bands[l]
	}
	fracIndex := float64(scaledPos) - float64(scaledIndex)
	a := bands[scaledIndex]
	b := bands[scaledIndex+1]
	first := float64(b / a)
	second := fracIndex
	//fmt.Printf("first %f second %f\n", first, second)
	return float32(a) * float32(math.Pow(first, second))
}

func (hfg *HFGlobal) getAFVTransformWeights(index int, c int) ([][]float32, error) {

	weights4x8 := hfg.getDCTQuantWeights(4, 8, hfg.params[index].dctParam[c])
	weights4x4 := hfg.getDCTQuantWeights(4, 4, hfg.params[index].params4x4[c])

	low := 0.8517778890324296
	high := 12.97166202570235

	bands := make([]float32, 4)
	bands[0] = hfg.params[index].param[c][5]
	if bands[0] < 0 {
		return nil, errors.New("Invalid band")
	}
	for i := 1; i < 4; i++ {
		bands[i] = bands[i-1] * quantMult(hfg.params[index].param[c][i+5])
		if bands[i] < 0 {
			return nil, errors.New("Negative band value")
		}
	}
	weight := util.MakeMatrix2D[float32](8, 8)
	weight[0][0] = 1
	weight[1][0] = hfg.params[index].param[c][0]
	weight[0][1] = hfg.params[index].param[c][1]
	weight[2][0] = hfg.params[index].param[c][2]
	weight[0][2] = hfg.params[index].param[c][3]
	weight[2][2] = hfg.params[index].param[c][4]

	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if x < 2 && y < 2 {
				continue
			}
			pos := (afvFreqs[y*4+x] - low) / (high - low)
			weight[2*x][2*y] = interpolate(float32(pos), bands)
		}
		for x := 0; x < 8; x++ {
			if x == 0 && y == 0 {
				continue
			}
			weight[2*y+1][x] = weights4x8[y][x]
		}
		for x := 0; x < 4; x++ {
			if x == 0 && y == 0 {
				continue
			}
			weight[2*y][2*x+1] = weights4x4[y][x]
		}
	}

	return weight, nil
}
