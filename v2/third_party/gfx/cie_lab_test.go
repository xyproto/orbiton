package gfx

func ExampleCIELab() {
	var (
		rgba   = ColorRGBA(255, 0, 0, 255)
		xyz    = ColorToXYZ(rgba)
		hunter = xyz.HunterLab(XYZReference2.D65)
		cieLab = xyz.CIELab(XYZReference2.D65)
	)

	Dump(
		"RGBA",
		rgba,
		"XYZ",
		xyz,
		"Hunter",
		hunter,
		"CIE-L*ab",
		cieLab,
	)
}
