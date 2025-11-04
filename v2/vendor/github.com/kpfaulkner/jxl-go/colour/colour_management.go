package colour

import (
	"errors"

	"github.com/kpfaulkner/jxl-go/util"
)

var (
	CM_PRI_SRGB   = GetPrimaries(PRI_SRGB)
	CM_PRI_BT2100 = GetPrimaries(PRI_BT2100)
	CM_PRI_P3     = GetPrimaries(PRI_P3)

	CM_WP_D65 = GetWhitePoint(WP_D65)
	CM_WP_D50 = GetWhitePoint(WP_D50)

	BRADFORD = [][]float32{
		{0.8951, 0.2664, -0.1614},
		{-0.7502, 1.7135, 0.0367},
		{0.0389, -0.0685, 1.0296},
	}

	BRADFORD_INVERSE = util.InvertMatrix3x3(BRADFORD)
)

type TransferFunction interface {
	ToLinear(input float64) float64
	FromLinear(input float64) float64
}

func GetConversionMatrix(targetPrim CIEPrimaries, targetWP CIEXY, currentPrim CIEPrimaries, currentWP CIEXY) ([][]float32, error) {

	if targetPrim.Matches(&currentPrim) && targetWP.Matches(&currentWP) {
		return util.MatrixIdentity(3), nil
	}

	var whitePointConv [][]float32
	var err error
	if !targetWP.Matches(&currentWP) {
		whitePointConv, err = AdaptWhitePoint(&targetWP, &currentWP)
		if err != nil {
			return nil, err
		}
	}
	forward, err := primariesToXYZ(&currentPrim, &currentWP)
	if err != nil {
		return nil, err
	}

	t, err := primariesToXYZ(&targetPrim, &targetWP)
	if err != nil {
		return nil, err
	}
	reverse := util.InvertMatrix3x3(t)
	res, err := util.MatrixMultiply(reverse, whitePointConv, forward)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func primariesToXYZ(primaries *CIEPrimaries, wp *CIEXY) ([][]float32, error) {
	if primaries == nil {
		return nil, nil
	}

	if wp == nil {
		wp = CM_WP_D50
	}
	if wp.X < 0 || wp.X > 1 || wp.Y <= 0 || wp.Y > 1 {
		return nil, errors.New("invalid argument")
	}
	r, errR := GetXYZ(*primaries.Red)
	g, errG := GetXYZ(*primaries.Green)
	b, errB := GetXYZ(*primaries.Blue)
	if errR != nil || errG != nil || errB != nil {
		return nil, errors.New("invalid argument")
	}
	primariesTr := [][]float32{r, g, b}
	primariesMatrix := util.TransposeMatrix(primariesTr, *util.NewPoint(3, 3))
	inversePrimaries := util.InvertMatrix3x3(primariesMatrix)
	w, err := GetXYZ(*wp)
	if err != nil {
		return nil, err
	}
	xyz, err := util.MatrixVectorMultiply(inversePrimaries, w)
	if err != nil {
		return nil, err
	}
	a := [][]float32{{xyz[0], 0, 0}, {0, xyz[1], 0}, {0, 0, xyz[2]}}
	res, err := util.MatrixMatrixMultiply(primariesMatrix, a)
	if err != nil {
		return nil, err
	}
	return res, nil

}

func validateXY(xy CIEXY) error {
	if xy.X < 0 || xy.X > 1 || xy.Y <= 0 || xy.Y > 1 {
		return errors.New("Invalid argument")
	}
	return nil
}

func GetXYZ(xy CIEXY) ([]float32, error) {
	if err := validateXY(xy); err != nil {
		return nil, err
	}
	invY := 1.0 / xy.Y
	return []float32{xy.X * invY, 1.0, (1.0 - xy.X - xy.Y) * invY}, nil
}

func GetTransferFunction(transfer int32) (TransferFunction, error) {

	switch transfer {
	case TF_LINEAR:
		return LinearTransferFunction{}, nil
	case TF_SRGB:
		return SRGBTransferFunction{}, nil
	case TF_PQ:
		return PQTransferFunction{}, nil
	case TF_BT709:
		return BT709TransferFunction{}, nil
	case TF_DCI:
		return NewGammaTransferFunction(transfer), nil
	case TF_HLG:
		return nil, errors.New("Not implemented")
	}

	if transfer < (1 << 24) {
		return NewGammaTransferFunction(transfer), nil
	}

	return nil, errors.New("Invalid transfer function")
}
