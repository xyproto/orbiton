package imagepreview

import (
	"image"
	"image/color"
	"os"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/palgen"
	"github.com/xyproto/vt"
	"golang.org/x/image/draw"
)

const (
	// BlockRune is the UTF-8 block character used for text-based image rendering.
	BlockRune = '▒'

	// ASCIIRune is the ASCII fallback character used for text-based image rendering.
	ASCIIRune = '#'
)

var envNoColor = env.Bool("NO_COLOR")

// DrawOnCanvas draws the given image onto a VT100 Canvas using the basic 16-color palette.
// The drawRune parameter specifies the character used for each pixel
// (typically BlockRune or ASCIIRune).
func DrawOnCanvas(canvas *vt.Canvas, m image.Image, drawRune rune) error {
	img, err := palgen.ConvertBasic(m)
	if err != nil {
		return err
	}
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if IsVT {
				c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
				vc := vt.White // default
				average := (c.R + c.G + c.B) / 3.0
				if found, ok := PaletteColorMap[[3]uint8{average, average, average}]; ok {
					vc = found
				}
				if envNoColor {
					vc = vt.Default
				}
				canvas.PlotColor(uint(x), uint(y), vc, drawRune)
			} else {
				c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
				vc := vt.White // default
				if found, ok := PaletteColorMap[[3]uint8{c.R, c.G, c.B}]; ok {
					vc = found
				}
				if envNoColor {
					vc = vt.Default
				}
				canvas.PlotColor(uint(x), uint(y), vc, drawRune)
			}
		}
	}
	return nil
}

// DrawTextImage renders an image file into a region of a VT100 Canvas using colored
// characters. col and row specify the top-left corner (0-indexed canvas coordinates);
// cols and rows specify the available area in terminal cells. The drawRune parameter
// specifies the character used for each pixel (typically BlockRune or ASCIIRune).
func DrawTextImage(canvas *vt.Canvas, path string, col, row, cols, rows uint, drawRune rune) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return
	}

	bounds := img.Bounds()
	imgW := float64(bounds.Dx())
	imgH := float64(bounds.Dy())
	if imgW == 0 || imgH == 0 {
		return
	}

	width := int(cols)
	height := int(rows)

	// Terminal cells are taller than they are wide (typically ~2:1).
	// Account for this so the image is not stretched or squished.
	cW, cH := TerminalCellPixels()
	cellRatio := float64(cH) / float64(cW) // e.g. 2.0 for 8x16 cells

	// Given the available height, how wide should the image be?
	targetW := int(float64(height) * (imgW / imgH) * cellRatio)
	// Given the available width, how tall should the image be?
	targetH := int(float64(width) * (imgH / imgW) / cellRatio)

	if targetW < width {
		width = targetW
	} else if targetH < height {
		height = targetH
	}

	if width <= 0 || height <= 0 {
		return
	}

	resizedImage := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(resizedImage, resizedImage.Rect, img, bounds, draw.Over, nil)

	indexedImg, err := palgen.ConvertBasic(resizedImage)
	if err != nil {
		return
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if IsVT {
				c := color.NRGBAModel.Convert(indexedImg.At(x, y)).(color.NRGBA)
				vc := vt.White // default
				average := (c.R + c.G + c.B) / 3.0
				if found, ok := PaletteColorMap[[3]uint8{average, average, average}]; ok {
					vc = found
				}
				if envNoColor {
					vc = vt.Default
				}
				canvas.PlotColor(col+uint(x), row+uint(y), vc, drawRune)
			} else {
				c := color.NRGBAModel.Convert(indexedImg.At(x, y)).(color.NRGBA)
				vc := vt.White // default
				if found, ok := PaletteColorMap[[3]uint8{c.R, c.G, c.B}]; ok {
					vc = found
				}
				if envNoColor {
					vc = vt.Default
				}
				canvas.PlotColor(col+uint(x), row+uint(y), vc, drawRune)
			}
		}
	}
}
