# gfx

[![Build status](https://github.com/peterhellberg/gfx/actions/workflows/test.yml/badge.svg?branch=master)](https://github.com/peterhellberg/gfx/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/peterhellberg/gfx?style=flat)](https://goreportcard.com/report/github.com/peterhellberg/gfx)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://pkg.go.dev/github.com/peterhellberg/gfx)
[![License MIT](https://img.shields.io/badge/license-MIT-lightgrey.svg?style=flat)](https://github.com/peterhellberg/gfx#license-mit)

Convenience package for dealing with graphics in my pixel drawing experiments.

#### :warning: NO STABILITY GUARANTEES :warning:

## Triangles

Triangles can be drawn to an image using a `*gfx.DrawTarget`.

![gfx-triangles](examples/gfx-example-triangles/gfx-example-triangles.png)

[embedmd]:# (examples/gfx-example-triangles/gfx-example-triangles.go)
```go
package main

import "github.com/peterhellberg/gfx"

var p = gfx.PaletteFamicube

func main() {
	n := 50
	m := gfx.NewPaletted(900, 270, p, p.Color(n+7))
	t := gfx.NewDrawTarget(m)

	t.MakeTriangles(&gfx.TrianglesData{
		vx(114, 16, n+1), vx(56, 142, n+2), vx(352, 142, n+3),
		vx(350, 142, n+4), vx(500, 50, n+5), vx(640, 236, n+6),
		vx(640, 70, n+8), vx(820, 160, n+9), vx(670, 236, n+10),
	}).Draw()

	gfx.SavePNG("gfx-example-triangles.png", m)
}

func vx(x, y float64, n int) gfx.Vertex {
	return gfx.Vertex{Position: gfx.V(x, y), Color: p.Color(n)}
}
```


## Polygons

A `gfx.Polygon` is represented by a list of vectors.
There is also `gfx.Polyline` which is a slice of polygons forming a line.

![gfx-example-polygon](examples/gfx-example-polygon/gfx-example-polygon.png)

[embedmd]:# (examples/gfx-example-polygon/gfx-example-polygon.go)
```go
package main

import "github.com/peterhellberg/gfx"

var edg32 = gfx.PaletteEDG32

func main() {
	m := gfx.NewNRGBA(gfx.IR(0, 0, 1024, 256))
	p := gfx.Polygon{
		{80, 40},
		{440, 60},
		{700, 200},
		{250, 230},
		{310, 140},
	}

	p.EachPixel(m, func(x, y int) {
		pv := gfx.IV(x, y)
		l := pv.To(p.Rect().Center()).Len()

		gfx.Mix(m, x, y, edg32.Color(int(l/18)%32))
	})

	for n, v := range p {
		c := edg32.Color(n * 4)

		gfx.DrawCircle(m, v, 15, 8, gfx.ColorWithAlpha(c, 96))
		gfx.DrawCircle(m, v, 16, 1, c)
	}

	gfx.SavePNG("gfx-example-polygon.png", m)
}
```


## Blocks

You can draw (isometric) blocks using the `gfx.Blocks` and `gfx.Block` types.

![gfx-example-blocks](examples/gfx-example-blocks/gfx-example-blocks.png)

[embedmd]:# (examples/gfx-example-blocks/gfx-example-blocks.go)
```go
package main

import "github.com/peterhellberg/gfx"

func main() {
	var (
		dst    = gfx.NewPaletted(898, 330, gfx.PaletteGo, gfx.PaletteGo[14])
		rect   = gfx.BoundsToRect(dst.Bounds())
		origin = rect.Center().ScaledXY(gfx.V(1.5, -2.5)).Vec3(0.55)
		blocks gfx.Blocks
	)

	for i, bc := range gfx.BlockColorsGo {
		var (
			f    = float64(i) + 0.5
			v    = f * 11
			pos  = gfx.V3(290+(v*3), 8.5*v, 9*(f+2))
			size = gfx.V3(90, 90, 90)
		)

		blocks.AddNewBlock(pos, size, bc)
	}

	blocks.Draw(dst, origin)

	gfx.SavePNG("gfx-example-blocks.png", dst)
}
```

## Signed Distance Functions

The `gfx.SignedDistance` type allows you to use basic [signed distance functions](http://jamie-wong.com/2016/07/15/ray-marching-signed-distance-functions/) (and operations) to produce some interesting graphics.

![gfx-example-sdf](examples/gfx-example-sdf/gfx-example-sdf.png)

[embedmd]:# (examples/gfx-example-sdf/gfx-example-sdf.go)
```go
package main

import "github.com/peterhellberg/gfx"

func main() {
	c := gfx.PaletteEDG36.Color
	m := gfx.NewImage(1024, 256, c(5))

	gfx.EachPixel(m.Bounds(), func(x, y int) {
		sd := gfx.SignedDistance{gfx.IV(x, y)}

		if d := sd.OpRepeat(gfx.V(128, 128), func(sd gfx.SignedDistance) float64 {
			return sd.OpSubtraction(sd.Circle(50), sd.Line(gfx.V(0, 0), gfx.V(64, 64)))
		}); d < 40 {
			m.Set(x, y, c(int(gfx.MathAbs(d/5))))
		}
	})

	gfx.SavePNG("gfx-example-sdf.png", m)
}
```

## Domain Coloring

You can use the `CmplxPhaseAt` method on a `gfx.Palette` to do [domain coloring](https://en.wikipedia.org/wiki/Domain_coloring).

![gfx-example-domain-coloring](examples/gfx-example-domain-coloring/gfx-example-domain-coloring.png)

[embedmd]:# (examples/gfx-example-domain-coloring/gfx-example-domain-coloring.go)
```go
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
```

## Animation

There is rudimentary support for making animations using `gfx.Animation`, the animations can then be encoded into GIF.

![gfx-example-animation](examples/gfx-example-animation/gfx-example-animation.gif)

[embedmd]:# (examples/gfx-example-animation/gfx-example-animation.go)
```go
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
```

## Line drawing

### DrawInt functions

Drawing functions based on [TinyDraw](https://github.com/tinygo-org/tinydraw),
which in turn is based on the [Adafruit GFX library](https://github.com/adafruit/Adafruit-GFX-Library).

![gfx-example-draw-int](examples/gfx-example-draw-int/gfx-example-draw-int.png)

[embedmd]:# (examples/gfx-example-draw-int/gfx-example-draw-int.go)
```go
package main

import "github.com/peterhellberg/gfx"

func main() {
	m := gfx.NewImage(160, 128, gfx.ColorTransparent)

	p := gfx.PaletteNight16

	gfx.DrawIntLine(m, 10, 10, 94, 10, p.Color(0))
	gfx.DrawIntLine(m, 94, 16, 10, 16, p.Color(1))
	gfx.DrawIntLine(m, 10, 20, 10, 118, p.Color(2))
	gfx.DrawIntLine(m, 16, 118, 16, 20, p.Color(4))

	gfx.DrawIntLine(m, 40, 40, 80, 80, p.Color(5))
	gfx.DrawIntLine(m, 40, 40, 80, 70, p.Color(6))
	gfx.DrawIntLine(m, 40, 40, 80, 60, p.Color(7))
	gfx.DrawIntLine(m, 40, 40, 80, 50, p.Color(8))
	gfx.DrawIntLine(m, 40, 40, 80, 40, p.Color(9))

	gfx.DrawIntLine(m, 100, 100, 40, 100, p.Color(10))
	gfx.DrawIntLine(m, 100, 100, 40, 90, p.Color(11))
	gfx.DrawIntLine(m, 100, 100, 40, 80, p.Color(12))
	gfx.DrawIntLine(m, 100, 100, 40, 70, p.Color(13))
	gfx.DrawIntLine(m, 100, 100, 40, 60, p.Color(14))
	gfx.DrawIntLine(m, 100, 100, 40, 50, p.Color(15))

	gfx.DrawIntRectangle(m, 30, 106, 120, 20, p.Color(14))
	gfx.DrawIntFilledRectangle(m, 34, 110, 112, 12, p.Color(8))

	gfx.DrawIntCircle(m, 120, 30, 20, p.Color(5))
	gfx.DrawIntFilledCircle(m, 120, 30, 16, p.Color(4))

	gfx.DrawIntTriangle(m, 120, 102, 100, 80, 152, 46, p.Color(9))
	gfx.DrawIntFilledTriangle(m, 119, 98, 105, 80, 144, 54, p.Color(6))

	s := gfx.NewScaledImage(m, 6)

	gfx.SavePNG("gfx-example-draw-int.png", s)
}
```

### Bresenham's line algorithm

`gfx.DrawLineBresenham` draws a line using [Bresenham's line algorithm](http://en.wikipedia.org/wiki/Bresenham's_line_algorithm).

![gfx-example-bresenham-line](examples/gfx-example-bresenham-line/gfx-example-bresenham-line.png)

[embedmd]:# (examples/gfx-example-bresenham-line/gfx-example-bresenham-line.go)
```go
package main

import "github.com/peterhellberg/gfx"

var (
	red   = gfx.BlockColorRed.Medium
	green = gfx.BlockColorGreen.Medium
	blue  = gfx.BlockColorBlue.Medium
)

func main() {
	m := gfx.NewImage(32, 16, gfx.ColorTransparent)

	gfx.DrawLineBresenham(m, gfx.V(2, 2), gfx.V(2, 14), red)
	gfx.DrawLineBresenham(m, gfx.V(6, 2), gfx.V(32, 2), green)
	gfx.DrawLineBresenham(m, gfx.V(6, 6), gfx.V(30, 14), blue)

	s := gfx.NewScaledImage(m, 16)

	gfx.SavePNG("gfx-example-bresenham-line.png", s)
}
```

## Geometry and Transformation

The (2D) geometry and transformation types are based on those found in <https://github.com/faiface/pixel> (but indended for use without Pixel)

### 2D types

#### Vec

`gfx.Vec` is a 2D vector type with X and Y components.

#### Rect

`gfx.Rect` is a 2D rectangle aligned with the axes of the coordinate system. It is defined by two `gfx.Vec`, Min and Max.

#### Matrix

`gfx.Matrix` is a 2x3 affine matrix that can be used for all kinds of spatial transforms, such as movement, scaling and rotations.

![gfx-readme-examples-matrix](https://user-images.githubusercontent.com/565124/51478881-f8e69a00-1d8c-11e9-92c5-270c767dfc06.gif)

[embedmd]:# (examples/gfx-example-matrix/gfx-example-matrix.go)
```go
package main

import "github.com/peterhellberg/gfx"

var en4 = gfx.PaletteEN4

func main() {
	a := &gfx.Animation{Delay: 10}

	c := gfx.V(128, 128)

	p := gfx.Polygon{
		{50, 50},
		{50, 206},
		{128, 96},
		{206, 206},
		{206, 50},
	}

	for d := 0.0; d < 360; d += 2 {
		m := gfx.NewPaletted(256, 256, en4, en4.Color(3))

		matrix := gfx.IM.RotatedDegrees(c, d)

		gfx.DrawPolygon(m, p.Project(matrix), 0, en4.Color(2))
		gfx.DrawPolygon(m, p.Project(matrix.Scaled(c, 0.5)), 0, en4.Color(1))

		gfx.DrawCircleFilled(m, c, 5, en4.Color(0))

		a.AddPalettedImage(m)
	}

	a.SaveGIF("/tmp/gfx-readme-examples-matrix.gif")
}
```

### 3D types

#### Vec3

`gfx.Vec3` is a 3D vector type with X, Y and Z components.

#### Box

`gfx.Box` is a 3D box. It is defined by two `gfx.Vec3`, Min and Max

## Errors

The `gfx.Error` type is a string that implements the `error` interface.

> If you are using [Ebiten](https://github.com/hajimehoshi/ebiten) then you can return the provided `gfx.ErrDone` error to exit its run loop.

## HTTP

You can use `gfx.GetPNG` to download and decode a PNG given an URL.

## Log

I find that it is fairly common for me to do some logging driven development
when experimenting with graphical effects, so I've included `gfx.Log`,
`gfx.Dump`, `gfx.Printf` and `gfx.Sprintf` in this package.

## Math

I have included a few functions that call functions in the `math` package.

There is also `gfx.Sign`, `gfx.Clamp` and `gfx.Lerp` functions for `float64`.

## Cmplx

I have included a few functions that call functions in the `cmplx` package.

## Reading files

It is fairly common to read files in my experiments, so I've included `gfx.ReadFile` and `gfx.ReadJSON` in this package.

## Resizing images

You can use `gfx.ResizeImage` to resize an image. (nearest neighbor, mainly useful for pixelated graphics)

## Noise

Different types of noise is often used in procedural generation.

### SimplexNoise

SimplexNoise is a speed-improved simplex noise algorithm for 2D, 3D and 4D.

![gfx-example-simplex](examples/gfx-example-simplex/gfx-example-simplex.png)

[embedmd]:# (examples/gfx-example-simplex/gfx-example-simplex.go)
```go
package main

import "github.com/peterhellberg/gfx"

func main() {
	sn := gfx.NewSimplexNoise(17)

	dst := gfx.NewImage(1024, 256)

	gfx.EachImageVec(dst, gfx.ZV, func(u gfx.Vec) {
		n := sn.Noise2D(u.X/900, u.Y/900)
		c := gfx.PaletteSplendor128.At(n / 2)

		gfx.SetVec(dst, u, c)
	})

	gfx.SavePNG("gfx-example-simplex.png", dst)
}
```

## Colors

You can construct new colors using `gfx.ColorRGBA`, `gfx.ColorNRGBA`, `gfx.ColorGray`, `gfx.ColorGray16` and `gfx.ColorWithAlpha`.

There is also a `gfx.LerpColors` function that performs linear interpolation between two colors.

### Default colors

There are a few default colors in this package, convenient when you just want to experiment,
for more ambitious projects I suggest creating a `gfx.Palette` (or even use one of the included palettes).


| Variable               | Color
|------------------------|---------------------------------------------------------
| `gfx.ColorBlack`       | ![gfx.ColorBlack](examples/gfx-colors/gfx-ColorBlack.png)
| `gfx.ColorWhite`       | ![gfx.ColorWhite](examples/gfx-colors/gfx-ColorWhite.png)
| `gfx.ColorTransparent` | ![gfx.ColorTransparent](examples/gfx-colors/gfx-ColorTransparent.png)
| `gfx.ColorOpaque`      | ![gfx.ColorOpaque](examples/gfx-colors/gfx-ColorOpaque.png)
| `gfx.ColorRed`         | ![gfx.ColorRed](examples/gfx-colors/gfx-ColorRed.png)
| `gfx.ColorGreen`       | ![gfx.ColorGreen](examples/gfx-colors/gfx-ColorGreen.png)
| `gfx.ColorBlue`        | ![gfx.ColorBlue](examples/gfx-colors/gfx-ColorBlue.png)
| `gfx.ColorCyan`        | ![gfx.ColorCyan](examples/gfx-colors/gfx-ColorCyan.png)
| `gfx.ColorMagenta`     | ![gfx.ColorMagenta](examples/gfx-colors/gfx-ColorMagenta.png)
| `gfx.ColorYellow`      | ![gfx.ColorYellow](examples/gfx-colors/gfx-ColorYellow.png)

### Block colors

Each `gfx.BlockColor` consists of a `Dark`, `Medium` and `Light` shade of the same color.


| Variable                     | Block Color
|------------------------------|---------------------------------------------------------
| `gfx.BlockColorYellow`       | ![gfx.BlockColorYellow](examples/gfx-colors/gfx-BlockColorYellow.png)
| `gfx.BlockColorOrange`       | ![gfx.BlockColorOrange](examples/gfx-colors/gfx-BlockColorOrange.png)
| `gfx.BlockColorBrown`        | ![gfx.BlockColorBrown](examples/gfx-colors/gfx-BlockColorBrown.png)
| `gfx.BlockColorGreen`        | ![gfx.BlockColorGreen](examples/gfx-colors/gfx-BlockColorGreen.png)
| `gfx.BlockColorBlue`         | ![gfx.BlockColorBlue](examples/gfx-colors/gfx-BlockColorBlue.png)
| `gfx.BlockColorPurple`       | ![gfx.BlockColorPurple](examples/gfx-colors/gfx-BlockColorPurple.png)
| `gfx.BlockColorRed`          | ![gfx.BlockColorRed](examples/gfx-colors/gfx-BlockColorRed.png)
| `gfx.BlockColorWhite`        | ![gfx.BlockColorWhite](examples/gfx-colors/gfx-BlockColorWhite.png)
| `gfx.BlockColorBlack`        | ![gfx.BlockColorBlack](examples/gfx-colors/gfx-BlockColorBlack.png)
| `gfx.BlockColorGoGopherBlue` | ![gfx.BlockColorGoGopherBlue](examples/gfx-colors/gfx-BlockColorGoGopherBlue.png)
| `gfx.BlockColorGoLightBlue`  | ![gfx.BlockColorGoLightBlue](examples/gfx-colors/gfx-BlockColorGoLightBlue.png)
| `gfx.BlockColorGoAqua`       | ![gfx.BlockColorGoAqua](examples/gfx-colors/gfx-BlockColorGoAqua.png)
| `gfx.BlockColorGoFuchsia`    | ![gfx.BlockColorGoFuchsia](examples/gfx-colors/gfx-BlockColorGoFuchsia.png)
| `gfx.BlockColorGoBlack`      | ![gfx.BlockColorGoBlack](examples/gfx-colors/gfx-BlockColorGoBlack.png)
| `gfx.BlockColorGoYellow`     | ![gfx.BlockColorGoYellow](examples/gfx-colors/gfx-BlockColorGoYellow.png)


### Palettes

There are a number of palettes in the `gfx` package,
most of them are found in the [Lospec Palette List](https://lospec.com/palette-list/).


| Variable                   | Colors | Lospec Palette
|----------------------------|-------:| -----------------------------------------------------
| `gfx.Palette1Bit`          |      2 | ![Palette1Bit](examples/gfx-palettes/gfx-Palette1Bit.png)
| `gfx.Palette2BitGrayScale` |      4 | ![Palette2BitGrayScale](examples/gfx-palettes/gfx-Palette2BitGrayScale.png)
| `gfx.PaletteEN4`           |      4 | ![PaletteEN4](examples/gfx-palettes/gfx-PaletteEN4.png)
| `gfx.PaletteARQ4`          |      4 | ![PaletteARQ4](examples/gfx-palettes/gfx-PaletteARQ4.png)
| `gfx.PaletteInk`           |      5 | ![PaletteInk](examples/gfx-palettes/gfx-PaletteInk.png)
| `gfx.Palette3Bit`          |      8 | ![Palette3Bit](examples/gfx-palettes/gfx-Palette3Bit.png)
| `gfx.PaletteEDG8`          |      8 | ![PaletteEDG8](examples/gfx-palettes/gfx-PaletteEDG8.png)
| `gfx.PaletteAmmo8`         |      8 | ![PaletteAmmo8](examples/gfx-palettes/gfx-PaletteAmmo8.png)
| `gfx.PaletteNYX8`          |      8 | ![PaletteNYX8](examples/gfx-palettes/gfx-PaletteNYX8.png)
| `gfx.Palette15PDX`         |     15 | ![Palette15PDX](examples/gfx-palettes/gfx-Palette15PDX.png)
| `gfx.PaletteCGA`           |     16 | ![PaletteCGA](examples/gfx-palettes/gfx-PaletteCGA.png)
| `gfx.PalettePICO8`         |     16 | ![PalettePICO8](examples/gfx-palettes/gfx-PalettePICO8.png)
| `gfx.PaletteNight16`       |     16 | ![PaletteNight16](examples/gfx-palettes/gfx-PaletteNight16.png)
| `gfx.PaletteAAP16`         |     16 | ![PaletteAAP16](examples/gfx-palettes/gfx-PaletteAAP16.png)
| `gfx.PaletteArne16`        |     16 | ![PaletteArne16](examples/gfx-palettes/gfx-PaletteArne16.png)
| `gfx.PaletteEDG16`         |     16 | ![PaletteEDG16](examples/gfx-palettes/gfx-PaletteEDG16.png)
| `gfx.Palette20PDX`         |     20 | ![Palette20PDX](examples/gfx-palettes/gfx-Palette20PDX.png)
| `gfx.PaletteTango`         |     27 | ![PaletteTango](examples/gfx-palettes/gfx-PaletteTango.png)
| `gfx.PaletteEDG32`         |     32 | ![PaletteEDG32](examples/gfx-palettes/gfx-PaletteEDG32.png)
| `gfx.PaletteEDG36`         |     36 | ![PaletteEDG36](examples/gfx-palettes/gfx-PaletteEDG36.png)
| `gfx.PaletteEDG64`         |     64 | ![PaletteEDG64](examples/gfx-palettes/gfx-PaletteEDG64.png)
| `gfx.PaletteAAP64`         |     64 | ![PaletteAAP64](examples/gfx-palettes/gfx-PaletteAAP64.png)
| `gfx.PaletteFamicube`      |     64 | ![PaletteFamicube](examples/gfx-palettes/gfx-PaletteFamicube.png)
| `gfx.PaletteSplendor128`   |    128 | ![PaletteSplendor128](examples/gfx-palettes/gfx-PaletteSplendor128.png)

The palette images were generated like this:

[embedmd]:# (examples/gfx-palettes/gfx-palettes.go)
```go
package main

import "github.com/peterhellberg/gfx"

func main() {
	for size, paletteLookup := range gfx.PalettesByNumberOfColors {
		for name, palette := range paletteLookup {
			dst := gfx.NewImage(size, 1)

			for x, c := range palette {
				dst.Set(x, 0, c)
			}

			filename := gfx.Sprintf("gfx-Palette%s.png", name)

			gfx.SavePNG(filename, gfx.NewResizedImage(dst, 1120, 96))
		}
	}
}
```


<img src="https://assets.c7.se/svg/viking-gopher.svg" align="right" width="30%" height="300">

## License (MIT)

Copyright (c) 2019-2024 [Peter Hellberg](https://c7.se)

> Permission is hereby granted, free of charge, to any person obtaining
> a copy of this software and associated documentation files (the
> "Software"), to deal in the Software without restriction, including
> without limitation the rights to use, copy, modify, merge, publish,
> distribute, sublicense, and/or sell copies of the Software, and to
> permit persons to whom the Software is furnished to do so, subject to
> the following conditions:
>
> The above copyright notice and this permission notice shall be
> included in all copies or substantial portions of the Software.

> THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
> EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
> MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
> NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
> LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
> OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
> WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
