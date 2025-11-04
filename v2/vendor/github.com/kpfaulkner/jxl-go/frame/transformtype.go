package frame

import (
	"errors"
	"fmt"

	"github.com/kpfaulkner/jxl-go/util"
)

const (
	MODE_LIBRARY = 0
	MODE_HORNUSS = 1
	MODE_DCT2    = 2
	MODE_DCT4    = 3
	MODE_DCT4_8  = 4
	MODE_AFV     = 5
	MODE_DCT     = 6
	MODE_RAW     = 7

	METHOD_DCT     = 0
	METHOD_DCT2    = 1
	METHOD_DCT4    = 2
	METHOD_HORNUSS = 3
	METHOD_DCT8_4  = 4
	METHOD_DCT4_8  = 5
	METHOD_AFV     = 6
)

var (
	SCALE_F = []float64{1.0000000000000000000, 1.0003954307206444720, 1.0015830492063566798,
		1.0035668445359847378, 1.0063534990068075448, 1.0099524393750471170,
		1.0143759095929498827, 1.0196390660646908181, 1.0257600967811994622,
		1.0327603660498609462, 1.0406645869479269795, 1.0495010240726261235,
		1.0593017296818027804, 1.0701028169146909598, 1.0819447744633102634,
		1.0948728278735071820, 1.1089373535928257701, 1.1241943530045446156,
		1.1407059950032801390, 1.1585412372562662921, 1.1777765381971696030,
		1.1984966740821024139, 1.2207956782314713353, 1.2447779229495839992,
		1.2705593687655135089, 1.2982690107340108228, 1.3280505578212198723,
		1.3600643892400108061, 1.3944898413648201160, 1.4315278911623840964,
		1.4714043176060183528, 1.5143734423313919909}
)

type TransformType struct {
	parameterIndex  int32
	dctSelectWidth  int32
	dctSelectHeight int32
	name            string
	ttType          int32
	matrixWidth     int32
	matrixHeight    int32
	orderID         int32
	transformMethod int32
	llfScale        [][]float32
	pixelHeight     int32
	pixelWidth      int32
}

var (
	DCT8       = NewTransformType("DCT 8x8", 0, 0, 0, 0, 8, 8)
	HORNUSS    = NewTransformType("Hornuss", 1, 1, 1, 3, 8, 8)
	DCT2       = NewTransformType("DCT 2x2", 2, 2, 1, 1, 8, 8)
	DCT4       = NewTransformType("DCT 4x4", 3, 3, 1, 2, 8, 8)
	DCT16      = NewTransformType("DCT 16x16", 4, 4, 2, 0, 16, 16)
	DCT32      = NewTransformType("DCT 32x32", 5, 5, 3, 0, 32, 32)
	DCT16_8    = NewTransformType("DCT 16x8", 6, 6, 4, 0, 16, 8)
	DCT8_16    = NewTransformType("DCT 8x16", 7, 6, 4, 0, 8, 16)
	DCT32_8    = NewTransformType("DCT 32x8", 8, 7, 5, 0, 32, 8)
	DCT8_32    = NewTransformType("DCT 8x32", 9, 7, 5, 0, 8, 32)
	DCT32_16   = NewTransformType("DCT 32x16", 10, 8, 6, 0, 32, 16)
	DCT16_32   = NewTransformType("DCT 16x32", 11, 8, 6, 0, 16, 32)
	DCT4_8     = NewTransformType("DCT 4x8", 12, 9, 1, 5, 8, 8)
	DCT8_4     = NewTransformType("DCT 8x4", 13, 9, 1, 4, 8, 8)
	AFV0       = NewTransformType("AFV0", 14, 10, 1, 6, 8, 8)
	AFV1       = NewTransformType("AFV1", 15, 10, 1, 6, 8, 8)
	AFV2       = NewTransformType("AFV2", 16, 10, 1, 6, 8, 8)
	AFV3       = NewTransformType("AFV3", 17, 10, 1, 6, 8, 8)
	DCT64      = NewTransformType("DCT 64x64", 18, 11, 7, 0, 64, 64)
	DCT64_32   = NewTransformType("DCT 64x32", 19, 12, 8, 0, 64, 32)
	DCT32_64   = NewTransformType("DCT 32x64", 20, 12, 8, 0, 32, 64)
	DCT128     = NewTransformType("DCT 128x128", 21, 13, 9, 0, 128, 128)
	DCT128_64  = NewTransformType("DCT 128x64", 22, 14, 10, 0, 128, 64)
	DCT64_128  = NewTransformType("DCT 64x128", 23, 14, 10, 0, 64, 128)
	DCT256     = NewTransformType("DCT 256x256", 24, 15, 11, 0, 256, 256)
	DCT256_128 = NewTransformType("DCT 256x128", 25, 16, 12, 0, 256, 128)
	DCT128_256 = NewTransformType("DCT 128x256", 26, 16, 12, 0, 128, 256)

	allDCT = []TransformType{*DCT8, *HORNUSS, *DCT2, *DCT4, *DCT16, *DCT32, *DCT16_8, *DCT8_16, *DCT32_8, *DCT8_32, *DCT32_16, *DCT16_32, *DCT4_8, *DCT8_4, *AFV0, *AFV1, *AFV2, *AFV3, *DCT64, *DCT64_32, *DCT32_64, *DCT128, *DCT128_64, *DCT64_128, *DCT256, *DCT256_128, *DCT128_256}
)

func NewTransformType(name string, transType int32, parameterIndex int32, orderID int32, transformMethod int32, pixelHeight int32, pixelWidth int32) *TransformType {

	dctSelectWidth := pixelWidth >> 3
	dctSelectHeight := pixelHeight >> 3
	yll := util.CeilLog2(dctSelectHeight)
	xll := util.CeilLog2(dctSelectWidth)
	tt := &TransformType{
		name:            name,
		ttType:          transType,
		parameterIndex:  parameterIndex,
		pixelHeight:     pixelHeight,
		pixelWidth:      pixelWidth,
		dctSelectWidth:  dctSelectWidth,
		dctSelectHeight: dctSelectHeight,
		orderID:         orderID,
		matrixWidth:     util.Max[int32](pixelHeight, pixelWidth),
		matrixHeight:    util.Min[int32](pixelHeight, pixelWidth),

		transformMethod: transformMethod,
		llfScale:        util.MakeMatrix2D[float32](dctSelectHeight, dctSelectWidth),
	}
	for y := int32(0); y < dctSelectHeight; y++ {
		for x := int32(0); x < dctSelectWidth; x++ {
			tt.llfScale[y][x] = float32(scaleF(y, int32(yll)) * scaleF(x, int32(xll)))
		}
	}

	return tt
}

func (tt TransformType) isVertical() bool {
	return tt.pixelHeight > tt.pixelWidth
}

func (tt TransformType) flip() bool {
	return tt.pixelHeight > tt.pixelWidth || tt.transformMethod == METHOD_DCT && tt.pixelHeight == tt.pixelWidth
}

func (tt TransformType) getPixelSize() util.Dimension {
	return util.Dimension{
		Width:  uint32(tt.pixelWidth),
		Height: uint32(tt.pixelHeight),
	}

}

func (tt TransformType) getDctSelectSize() util.Dimension {
	return util.Dimension{
		Width:  uint32(tt.dctSelectWidth),
		Height: uint32(tt.dctSelectHeight),
	}
}

func scaleF(c int32, b int32) float64 {
	//piSize := math.Pi * c
	//return (1.0 / math.Cos(piSize/(2*b)) * math.Cos(piSize/b) * math.Cos(2.0*piSize/(2.0*b)))
	return SCALE_F[c<<(5-b)]
}

func validateIndex(index int32, mode int32) (bool, error) {
	if mode < 0 || mode > 7 {
		return false, errors.New("Invalid mode")
	}
	if mode == MODE_LIBRARY || mode == MODE_DCT || mode == MODE_RAW {
		return true, nil
	}

	if index >= 0 && index <= 3 || index == 9 || index == 10 {
		return true, nil
	}

	return false, errors.New(fmt.Sprintf("Invalid index %d for mode %d", index, mode))

}

func getHorizontalTransformType(index int32) (*TransformType, error) {

	for _, tt := range allDCT {
		if tt.parameterIndex == index && !tt.isVertical() {
			return &tt, nil
		}
	}

	return nil, errors.New("Unable to find horizontal transform type")
}

func getByOrderID(orderID int32) TransformType {
	return orderLookup[orderID]
}
