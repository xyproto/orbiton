package imagepreview

import (
	"image"
	"image/color"
)

// ScaleNearestNeighbor scales src to dstW x dstH using nearest-neighbor
// interpolation, producing sharp pixels instead of a blurry bilinear upscale.
func ScaleNearestNeighbor(src image.Image, dstW, dstH int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	for y := range dstH {
		srcY := bounds.Min.Y + y*srcH/dstH
		for x := range dstW {
			srcX := bounds.Min.X + x*srcW/dstW
			r, g, b, a := src.At(srcX, srcY).RGBA()
			dst.SetRGBA(x, y, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)})
		}
	}
	return dst
}

// AspectRatioCells computes the display size in terminal cells that best fits
// the given image (imgW x imgH pixels) inside the available area
// (availCols x availRows cells) while preserving the image's pixel aspect ratio.
func AspectRatioCells(imgW, imgH, availCols, availRows uint) (cols, rows uint) {
	if imgW == 0 || imgH == 0 || availCols == 0 || availRows == 0 {
		return availCols, availRows
	}
	cellW, cellH := TerminalCellPixels()
	if cellW == 0 || cellH == 0 {
		return availCols, availRows
	}
	paneW := availCols * cellW
	paneH := availRows * cellH
	if imgW*paneH > imgH*paneW {
		// Wider than pane: fit to width, reduce rows.
		cols = availCols
		pixH := paneW * imgH / imgW
		rows = min((pixH+cellH-1)/cellH, availRows)
	} else {
		// Taller than pane: fit to height, reduce cols.
		rows = availRows
		pixW := paneH * imgW / imgH
		cols = min((pixW+cellW-1)/cellW, availCols)
	}
	if cols == 0 {
		cols = 1
	}
	if rows == 0 {
		rows = 1
	}
	return
}
