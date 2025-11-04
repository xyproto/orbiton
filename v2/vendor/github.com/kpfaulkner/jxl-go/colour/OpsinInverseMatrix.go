package colour

import (
	"errors"
	"slices"

	"github.com/kpfaulkner/jxl-go/jxlio"
	"github.com/kpfaulkner/jxl-go/util"
)

var (
	DEFAULT_MATRIX = [][]float32{
		{11.031566901960783, -9.866943921568629, -0.16462299647058826},
		{-3.254147380392157, 4.418770392156863, -0.16462299647058826},
		{-3.6588512862745097, 2.7129230470588235, 1.9459282392156863}}
	DEFAULT_OPSIN_BIAS              = []float32{-0.0037930732552754493, -0.0037930732552754493, -0.0037930732552754493}
	DEFAULT_QUANT_BIAS              = []float32{1.0 - 0.05465007330715401, 1.0 - 0.07005449891748593, 1.0 - 0.049935103337343655}
	DEFAULT_QBIAS_NUMERATOR float32 = 0.145
)

type OpsinInverseMatrix struct {
	Matrix             [][]float32
	OpsinBias          []float32
	QuantBias          []float32
	CbrtOpsinBias      []float32
	Primaries          CIEPrimaries
	WhitePoint         CIEXY
	QuantBiasNumerator float32
}

func NewOpsinInverseMatrix() *OpsinInverseMatrix {
	return NewOpsinInverseMatrixAllParams(*CM_PRI_SRGB, *CM_WP_D65, DEFAULT_MATRIX, DEFAULT_OPSIN_BIAS, DEFAULT_QUANT_BIAS, DEFAULT_QBIAS_NUMERATOR)
}

func NewOpsinInverseMatrixAllParams(
	primaries CIEPrimaries,
	whitePoint CIEXY,
	matrix [][]float32,
	opsinBias []float32,
	quantBias []float32,
	quantBiasNumerator float32) *OpsinInverseMatrix {

	oim := &OpsinInverseMatrix{}
	oim.Matrix = matrix
	oim.OpsinBias = opsinBias
	oim.QuantBias = quantBias
	oim.QuantBiasNumerator = quantBiasNumerator
	oim.Primaries = primaries
	oim.WhitePoint = whitePoint
	oim.bakeCbrtBias()
	return oim
}

func NewOpsinInverseMatrixWithReader(reader jxlio.BitReader) (*OpsinInverseMatrix, error) {
	oim := &OpsinInverseMatrix{}
	var err error
	var useMatrix bool
	if useMatrix, err = reader.ReadBool(); err != nil {
		return nil, err
	}
	if useMatrix {
		oim.Matrix = DEFAULT_MATRIX
		oim.OpsinBias = DEFAULT_OPSIN_BIAS
		oim.QuantBias = DEFAULT_QUANT_BIAS
		oim.QuantBiasNumerator = DEFAULT_QBIAS_NUMERATOR
	} else {
		oim.Matrix = util.MakeMatrix2D[float32](3, 3)
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				if oim.Matrix[i][j], err = reader.ReadF16(); err != nil {
					return nil, err
				}
			}
		}
		oim.OpsinBias = make([]float32, 3)
		for i := 0; i < 3; i++ {
			if oim.OpsinBias[i], err = reader.ReadF16(); err != nil {
				return nil, err
			}
		}
		oim.QuantBias = make([]float32, 3)
		for i := 0; i < 3; i++ {
			if oim.QuantBias[i], err = reader.ReadF16(); err != nil {
				return nil, err
			}
		}
		if oim.QuantBiasNumerator, err = reader.ReadF16(); err != nil {
			return nil, err
		}
	}
	oim.Primaries = *CM_PRI_SRGB
	oim.WhitePoint = *CM_WP_D65
	oim.bakeCbrtBias()

	return oim, nil
}

func (oim *OpsinInverseMatrix) bakeCbrtBias() {
	oim.CbrtOpsinBias = make([]float32, 3)
	for c := 0; c < 3; c++ {
		oim.CbrtOpsinBias[c] = util.SignedPow(oim.OpsinBias[c], 1.0/3.0)
	}
}

func (oim *OpsinInverseMatrix) GetMatrix(prim *CIEPrimaries, white *CIEXY) (*OpsinInverseMatrix, error) {
	conversion, err := GetConversionMatrix(*prim, *white, oim.Primaries, oim.WhitePoint)
	if err != nil {
		return nil, err
	}
	matrix, err := util.MatrixMultiply(conversion, oim.Matrix)
	if err != nil {
		return nil, err
	}

	return NewOpsinInverseMatrixAllParams(*prim, *white, matrix, oim.OpsinBias, oim.QuantBias, oim.QuantBiasNumerator), nil
}

func (oim *OpsinInverseMatrix) InvertXYB(buffer [][][]float32, intensityTarget float32) error {

	if len(buffer) < 3 {
		return errors.New("Can only XYB on 3 channels")
	}
	itScale := 255.0 / intensityTarget
	for y := 0; y < len(buffer[0]); y++ {
		for x := 0; x < len(buffer[0][y]); x++ {
			gammaL := buffer[1][y][x] + buffer[0][y][x] - oim.CbrtOpsinBias[0]
			gammaM := buffer[1][y][x] - buffer[0][y][x] - oim.CbrtOpsinBias[1]
			gammaS := buffer[2][y][x] - oim.CbrtOpsinBias[2]
			mixL := gammaL*gammaL*gammaL + oim.OpsinBias[0]
			mixM := gammaM*gammaM*gammaM + oim.OpsinBias[1]
			mixS := gammaS*gammaS*gammaS + oim.OpsinBias[2]
			for c := 0; c < 3; c++ {
				buffer[c][y][x] = (mixL*oim.Matrix[c][0] + mixM*oim.Matrix[c][1] + mixS*oim.Matrix[c][2]) * itScale
			}
		}
	}
	return nil
}

// Matches determines if values are equal. Simplistic but will do for now.
func (oim *OpsinInverseMatrix) Matches(other OpsinInverseMatrix) bool {

	if !util.CompareMatrix2D(oim.Matrix, other.Matrix, func(a float32, b float32) bool { return a == b }) {
		return false
	}

	if slices.Compare(oim.OpsinBias, other.OpsinBias) != 0 {
		return false
	}

	if slices.Compare(oim.QuantBias, other.QuantBias) != 0 {
		return false
	}

	if oim.QuantBiasNumerator != other.QuantBiasNumerator {
		return false
	}

	if !oim.Primaries.Red.Matches(other.Primaries.Red) {
		return false
	}

	if !oim.Primaries.Green.Matches(other.Primaries.Green) {
		return false
	}

	if !oim.Primaries.Blue.Matches(other.Primaries.Blue) {
		return false
	}

	if !oim.WhitePoint.Matches(&other.WhitePoint) {
		return false
	}

	return true
}
