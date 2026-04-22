package main

import "github.com/peterhellberg/gfx"

func main() {
	a := &gfx.Animation{}
	p := gfx.PaletteEDG36

	var fireflower = []uint8{
		0, 1, 1, 1, 1, 1, 1, 0,
		1, 1, 2, 2, 2, 2, 1, 1,
		1, 2, 3, 3, 3, 3, 2, 1,
		1, 1, 2, 2, 2, 2, 1, 1,
		0, 1, 1, 1, 1, 1, 1, 0,
		0, 0, 0, 4, 4, 0, 0, 0,
		0, 0, 0, 4, 4, 0, 0, 0,
		4, 4, 0, 4, 4, 0, 4, 4,
		0, 4, 0, 4, 4, 0, 4, 0,
		0, 4, 4, 4, 4, 4, 4, 0,
		0, 0, 4, 4, 4, 4, 0, 0,
	}

	for i := 0; i < len(p)-4; i++ {
		t := gfx.NewTile(p[i:i+4], 8, fireflower)

		a.AddPalettedImage(gfx.NewScaledPalettedImage(t, 20))
	}

	a.SaveGIF("gfx-example-animation.gif")
}
