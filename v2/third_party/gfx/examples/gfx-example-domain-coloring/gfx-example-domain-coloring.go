package main

import "github.com/peterhellberg/gfx"

const (
	w, h        = 1800, 540
	fovY        = 1.9
	aspectRatio = float64(w) / float64(h)
	centerReal  = 0
	centerImag  = 0
	ahc         = aspectRatio*fovY/2.0 + centerReal
	hfc         = fovY/2.0 + centerImag
)

func pixelCoordinates(px, py int) gfx.Vec {
	return gfx.V(
		((float64(px)/(w-1))*2-1)*ahc,
		((float64(h-py-1)/(h-1))*2-1)*hfc,
	)
}

func main() {
	var (
		p  = gfx.PaletteEN4
		p0 = pixelCoordinates(0, 0)
		p1 = pixelCoordinates(w-1, h-1)
		y  = p0.Y
		d  = gfx.V((p1.X-p0.X)/(w-1), (p1.Y-p0.Y)/(h-1))
		m  = gfx.NewImage(w, h)
	)

	for py := 0; py < h; py++ {
		x := p0.X

		for px := 0; px < w; px++ {
			cc := p.CmplxPhaseAt(gfx.CmplxCos(gfx.CmplxSin(0.42 / complex(y*x, x*x))))

			m.Set(px, py, cc)

			x += d.X
		}

		y += d.Y
	}

	gfx.SavePNG("gfx-example-domain-coloring.png", m)
}
