package bundle

import (
	"errors"

	"github.com/kpfaulkner/jxl-go/jxlio"
)

type AnimationHeader struct {
	HaveTimeCodes bool
}

func NewAnimationHeader(reader jxlio.BitReader) (*AnimationHeader, error) {
	return nil, errors.New("Animation not implemented")
}
