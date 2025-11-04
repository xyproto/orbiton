package bundle

import (
	"errors"
	"math"

	"github.com/kpfaulkner/jxl-go/jxlio"
)

type Extensions struct {
	Payloads      [64][]byte
	ExtensionsKey uint64
}

func NewExtensions() *Extensions {
	ex := &Extensions{}
	ex.ExtensionsKey = 0
	return ex
}

func NewExtensionsWithReader(reader jxlio.BitReader) (*Extensions, error) {
	ex := &Extensions{}
	var err error
	if ex.ExtensionsKey, err = reader.ReadU64(); err != nil {
		return nil, err
	}

	var length uint64
	for i := uint64(0); i < 64; i++ {
		if (1<<i)&ex.ExtensionsKey != 0 {
			if length, err = reader.ReadU64(); err != nil {
				return nil, err
			}
			if length > math.MaxUint32 {
				return nil, errors.New("Large Extensions unsupported")
			}
			ex.Payloads[i] = make([]byte, length)
		}
	}
	for i := 0; i < 64; i++ {
		if len(ex.Payloads[i]) > 0 {
			for j := 0; j < len(ex.Payloads[i]); j++ {
				if bits, err := reader.ReadBits(8); err != nil {
					return nil, err
				} else {
					ex.Payloads[i][j] = byte(bits)
				}
			}
		}
	}
	return ex, nil
}
