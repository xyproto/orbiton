package frame

import "github.com/kpfaulkner/jxl-go/util"

type DCTParam struct {
	dctParam    [][]float64
	param       [][]float32
	mode        int32
	denominator float32
	params4x4   [][]float64
}

// Equals compares 2 DCTParams structs. Slightly concerned about having to
// compare all the multi-dimensional slices. Investigated slices.EqualFunc, Equal, compare etc.
// but unsure about getting those working for multi-dimensional. So will just do naively for now
// and measure later.
// TODO(kpfaulkner) do some measuring around performance here.
func (dct DCTParam) Equals(other DCTParam) bool {
	if dct.mode != other.mode {
		return false
	}
	if dct.denominator != other.denominator {
		return false
	}
	if len(dct.dctParam) != len(other.dctParam) {
		return false
	}
	if len(dct.param) != len(other.param) {
		return false
	}

	// FIXME(kpfaulkner) not keen on == for float64...  need to double check
	if !util.CompareMatrix2D(dct.dctParam, other.dctParam, func(a float64, b float64) bool {
		return a == b
	}) {
		return false
	}

	if !util.CompareMatrix2D(dct.param, other.param, func(a float32, b float32) bool {
		return a == b
	}) {
		return false
	}

	if !util.CompareMatrix2D(dct.params4x4, other.params4x4, func(a float64, b float64) bool {
		return a == b
	}) {
		return false
	}
	return true
}

func NewDCTParam() *DCTParam {
	return &DCTParam{
		dctParam:    [][]float64{},
		param:       [][]float32{},
		params4x4:   [][]float64{},
		mode:        0,
		denominator: 0,
	}
}
