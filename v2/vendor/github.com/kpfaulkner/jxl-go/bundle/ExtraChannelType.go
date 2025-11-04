package bundle

const (
	ALPHA              = 0
	DEPTH              = 1
	SPOT_COLOR         = 2
	SELECTION_MASK     = 3
	CMYK_BLACK         = 4
	COLOR_FILTER_ARRAY = 5
	THERMAL            = 6
	NON_OPTIONAL       = 15
	OPTIONAL           = 16
)

func ValidateExtraChannel(ec int32) bool {
	return ec >= 0 && ec <= 6 || ec == 15 || ec == 16
}
