package colour

import (
	"errors"

	"github.com/kpfaulkner/jxl-go/jxlio"
)

type ColourEncodingBundle struct {
	Prim            *CIEPrimaries
	White           *CIEXY
	ColourEncoding  int32
	WhitePoint      int32
	Primaries       int32
	Tf              int32
	RenderingIntent int32
	UseIccProfile   bool
}

func NewColourEncodingBundle() (*ColourEncodingBundle, error) {
	ceb := &ColourEncodingBundle{}
	ceb.UseIccProfile = false
	ceb.ColourEncoding = CE_RGB
	ceb.WhitePoint = WP_D65
	ceb.White = getWhitePoint(ceb.WhitePoint)
	ceb.Primaries = PRI_SRGB
	ceb.Prim = GetPrimaries(ceb.Primaries)
	ceb.Tf = TF_SRGB
	ceb.RenderingIntent = RI_RELATIVE
	return ceb, nil
}

func NewColourEncodingBundleWithReader(reader jxlio.BitReader) (*ColourEncodingBundle, error) {
	ceb := &ColourEncodingBundle{}
	var allDefault bool
	var err error
	if allDefault, err = reader.ReadBool(); err != nil {
		return nil, err
	}

	if !allDefault {
		if ceb.UseIccProfile, err = reader.ReadBool(); err != nil {
			return nil, err
		}
	}

	if !allDefault {
		if ceb.ColourEncoding, err = reader.ReadEnum(); err != nil {
			return nil, err
		}
	} else {
		ceb.ColourEncoding = CE_RGB
	}

	if !ValidateColourEncoding(ceb.ColourEncoding) {
		return nil, errors.New("Invalid ColorSpace enum")
	}

	if !allDefault && !ceb.UseIccProfile && ceb.ColourEncoding != CE_XYB {
		if ceb.WhitePoint, err = reader.ReadEnum(); err != nil {
			return nil, err
		}
	} else {
		ceb.WhitePoint = WP_D65
	}

	if !ValidateWhitePoint(ceb.WhitePoint) {
		return nil, errors.New("Invalid WhitePoint enum")
	}

	if ceb.WhitePoint == WP_CUSTOM {
		white, err := NewCustomXY(reader)
		if err != nil {
			return nil, err
		}
		ceb.White = &white.CIEXY
	} else {
		ceb.White = getWhitePoint(ceb.WhitePoint)
	}

	if !allDefault && !ceb.UseIccProfile && ceb.ColourEncoding != CE_XYB && ceb.ColourEncoding != CE_GRAY {
		if ceb.Primaries, err = reader.ReadEnum(); err != nil {
			return nil, err
		}
	} else {
		ceb.Primaries = PRI_SRGB
	}

	if !ValidatePrimaries(ceb.Primaries) {
		return nil, errors.New("Invalid Primaries enum")
	}

	if ceb.Primaries == PRI_CUSTOM {
		pRed, err := NewCustomXY(reader)
		if err != nil {
			return nil, err
		}
		pGreen, err := NewCustomXY(reader)
		if err != nil {
			return nil, err
		}
		pBlue, err := NewCustomXY(reader)
		if err != nil {
			return nil, err
		}
		ceb.Prim = NewCIEPrimaries(&pRed.CIEXY, &pGreen.CIEXY, &pBlue.CIEXY)
	} else {
		ceb.Prim = GetPrimaries(ceb.Primaries)
	}

	if !allDefault && !ceb.UseIccProfile {
		var useGamma bool
		if useGamma, err = reader.ReadBool(); err != nil {
			return nil, err
		}
		if useGamma {
			if tf, err := reader.ReadBits(24); err != nil {
				return nil, err
			} else {
				ceb.Tf = int32(tf)
			}
		} else {
			var tfEnum int32
			if tfEnum, err = reader.ReadEnum(); err != nil {
				return nil, err
			}
			ceb.Tf = (1 << 24) + tfEnum
		}
		if !ValidateTransfer(ceb.Tf) {
			return nil, errors.New("Illegal transfer function")
		}
		if ceb.RenderingIntent, err = reader.ReadEnum(); err != nil {
			return nil, err
		}
		if !ValidateRenderingIntent(ceb.RenderingIntent) {
			return nil, errors.New("Invalid RenderingIntent enum")
		}
	} else {
		ceb.Tf = TF_SRGB
		ceb.RenderingIntent = RI_RELATIVE
	}

	return ceb, nil
}

func getWhitePoint(whitePoint int32) *CIEXY {
	switch whitePoint {
	case WP_D65:
		return NewCIEXY(0.3127, 0.3290)
	case WP_E:
		return NewCIEXY(1/3, 1/3)
	case WP_DCI:
		return NewCIEXY(0.314, 0.351)
	case WP_D50:
		return NewCIEXY(0.34567, 0.34567)
	}
	return nil
}
