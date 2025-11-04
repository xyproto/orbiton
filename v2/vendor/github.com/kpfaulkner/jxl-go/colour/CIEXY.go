package colour

import (
	"errors"
	"math"

	"github.com/kpfaulkner/jxl-go/util"
)

type CIEXY struct {
	X float32
	Y float32
}

func NewCIEXY(x float32, y float32) *CIEXY {
	cxy := &CIEXY{}
	cxy.X = x
	cxy.Y = y
	return cxy
}

func (cxy *CIEXY) Matches(b *CIEXY) bool {
	if b == nil {
		return false
	}

	return math.Abs(float64(cxy.X-b.X))+math.Abs(float64(cxy.Y-b.Y)) < 0.0001
}

func AdaptWhitePoint(targetWP *CIEXY, currentWP *CIEXY) ([][]float32, error) {
	if targetWP == nil {
		targetWP = CM_WP_D50
	}
	if currentWP == nil {
		currentWP = CM_WP_D65
	}

	wCurrent, err := GetXYZ(*currentWP)
	if err != nil {
		return nil, err
	}
	lmsCurrent, err := util.MatrixVectorMultiply(BRADFORD, wCurrent)
	if err != nil {
		return nil, err
	}

	wTarget, err := GetXYZ(*targetWP)
	if err != nil {
		return nil, err
	}
	lmsTarget, err := util.MatrixVectorMultiply(BRADFORD, wTarget)
	if err != nil {
		return nil, err
	}

	if !isLMSValid(lmsCurrent) {
		return nil, errors.New("Invalid LMS")
	}

	a := util.MakeMatrix2D[float32](3, 3)
	for i := 0; i < 3; i++ {
		a[i][i] = lmsTarget[i] / lmsCurrent[i]
	}

	return util.MatrixMultiply(BRADFORD_INVERSE, a, BRADFORD)

}

func isLMSValid(lms []float32) bool {
	for i := 0; i < len(lms); i++ {
		if math.Abs(float64(lms[i])) < 1e-8 {
			return false
		}
	}
	return true
}
