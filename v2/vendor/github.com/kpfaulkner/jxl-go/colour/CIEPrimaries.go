package colour

type CIEPrimaries struct {
	Red   *CIEXY
	Green *CIEXY
	Blue  *CIEXY
}

func NewCIEPrimaries(red *CIEXY, green *CIEXY, blue *CIEXY) *CIEPrimaries {
	cp := CIEPrimaries{}
	cp.Red = red
	cp.Green = green
	cp.Blue = blue
	return &cp
}

func (cp *CIEPrimaries) Matches(b *CIEPrimaries) bool {

	if b == nil {
		return false
	}
	return cp.Red.Matches(b.Red) && cp.Green.Matches(b.Green) && cp.Blue.Matches(b.Blue)
}
