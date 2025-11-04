package colour

const (
	PRI_SRGB   int32 = 1
	PRI_CUSTOM int32 = 2
	PRI_BT2100 int32 = 9
	PRI_P3     int32 = 11

	WP_D50    int32 = -1
	WP_D65    int32 = 1
	WP_CUSTOM int32 = 2
	WP_E      int32 = 10
	WP_DCI    int32 = 11

	CE_RGB     int32 = 0
	CE_GRAY    int32 = 1
	CE_XYB     int32 = 2
	CE_UNKNOWN int32 = 3

	RI_PERCEPTUAL int32 = 0
	RI_RELATIVE   int32 = 1
	RI_SATURATION int32 = 2
	RI_ABSOLUTE   int32 = 3

	TF_BT709   int32 = 1 + (1 << 24)
	TF_UNKNOWN int32 = 2 + (1 << 24)
	TF_LINEAR  int32 = 8 + (1 << 24)
	TF_SRGB    int32 = 13 + (1 << 24)
	TF_PQ      int32 = 16 + (1 << 24)
	TF_DCI     int32 = 17 + (1 << 24)
	TF_HLG     int32 = 18 + (1 << 24)
)

func ValidateColourEncoding(colourEncoding int32) bool {
	return colourEncoding >= 0 && colourEncoding <= 3
}

func ValidateWhitePoint(whitePoint int32) bool {
	return whitePoint == WP_D65 || whitePoint == WP_CUSTOM || whitePoint == WP_E || whitePoint == WP_DCI
}

func ValidatePrimaries(primaries int32) bool {
	return primaries == PRI_SRGB || primaries == PRI_CUSTOM || primaries == PRI_BT2100 || primaries == PRI_P3
}

func ValidateRenderingIntent(renderingIntent int32) bool {
	return renderingIntent >= 0 && renderingIntent <= 3
}

func ValidateTransfer(transfer int32) bool {

	if transfer < 0 {
		return false
	} else if transfer <= 10_000_000 {
		return true
	} else if transfer < (1 << 24) {
		return false
	} else {
		return transfer == TF_BT709 ||
			transfer == TF_UNKNOWN ||
			transfer == TF_LINEAR ||
			transfer == TF_SRGB ||
			transfer == TF_PQ ||
			transfer == TF_DCI ||
			transfer == TF_HLG
	}

}

func GetPrimaries(primaries int32) *CIEPrimaries {
	switch primaries {
	case PRI_SRGB:
		return NewCIEPrimaries(
			NewCIEXY(0.639998686, 0.330010138),
			NewCIEXY(0.300003784, 0.600003357),
			NewCIEXY(0.150002046, 0.059997204))
	case PRI_BT2100:
		return NewCIEPrimaries(
			NewCIEXY(0.708, 0.292),
			NewCIEXY(0.170, 0.797),
			NewCIEXY(0.131, 0.046))
	case PRI_P3:
		return NewCIEPrimaries(
			NewCIEXY(0.680, 0.320),
			NewCIEXY(0.265, 0.690),
			NewCIEXY(0.150, 0.060))

	}

	return nil
}

func GetWhitePoint(whitePoint int32) *CIEXY {
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
