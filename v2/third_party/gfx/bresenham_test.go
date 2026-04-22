package gfx

func ExampleDrawLineBresenham() {
	dst := NewPaletted(10, 5, Palette1Bit, ColorWhite)

	DrawLineBresenham(dst, V(1, 1), V(8, 3), ColorBlack)

	for y := 0; y < dst.Bounds().Dy(); y++ {
		for x := 0; x < dst.Bounds().Dx(); x++ {
			if dst.Index(x, y) == 0 {
				Printf("▓▓")
			} else {
				Printf("░░")
			}
		}
		Printf("\n")
	}

	// Output:
	//
	// ░░░░░░░░░░░░░░░░░░░░
	// ░░▓▓▓▓░░░░░░░░░░░░░░
	// ░░░░░░▓▓▓▓▓▓▓▓░░░░░░
	// ░░░░░░░░░░░░░░▓▓▓▓░░
	// ░░░░░░░░░░░░░░░░░░░░
	//
}

func ExampleDrawLineBresenham_steep() {
	dst := NewPaletted(10, 5, Palette1Bit, ColorWhite)

	DrawLineBresenham(dst, V(7, 3), V(6, 1), ColorBlack)

	for y := 0; y < dst.Bounds().Dy(); y++ {
		for x := 0; x < dst.Bounds().Dx(); x++ {
			if dst.Index(x, y) == 0 {
				Printf("▓▓")
			} else {
				Printf("░░")
			}
		}
		Printf("\n")
	}

	// Output:
	//
	// ░░░░░░░░░░░░░░░░░░░░
	// ░░░░░░░░░░░░▓▓░░░░░░
	// ░░░░░░░░░░░░▓▓░░░░░░
	// ░░░░░░░░░░░░░░▓▓░░░░
	// ░░░░░░░░░░░░░░░░░░░░
	//
}
