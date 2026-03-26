package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"github.com/xyproto/burnfont"
	"github.com/xyproto/vt"
)

const minimapCols = 20

// envKitty is true when TERM=xterm-kitty
var envKitty = os.Getenv("TERM") == "xterm-kitty"

// kittyDeleteImages sends the Kitty graphics protocol command to delete all placed images.
func kittyDeleteImages() {
	fmt.Fprintf(os.Stdout, "\033_Ga=d,d=A,q=2\033\\")
}

// renderEditorToImage renders the current in-memory editor content with syntax
// highlighting and returns an *image.RGBA. The image is imgWidth pixels wide with
// height proportional to the number of lines.
func (e *Editor) renderEditorToImage(theme Theme) *image.RGBA {
	const (
		imgWidth   = minimapCols * 8 // 8px per char column
		charWidth  = 8
		lineHeight = 10
		marginLeft = 2
		marginTop  = 2
		marginBot  = 2
		tabWidth   = 4
	)

	data := []byte(e.String())
	cfg := textConfigFromTheme(theme)
	taggedText, err := AsText(data, e.mode, func(c *TextConfig) { *c = cfg })
	if err != nil {
		taggedText = data
	}

	coloredText := tout.DarkTags(string(taggedText))
	charAttrs := make([]vt.CharAttribute, len([]rune(coloredText)))
	n := tout.ExtractToSlice(coloredText, &charAttrs)
	charAttrs = charAttrs[:n]

	type styledRune struct {
		r  rune
		fg vt.AttributeColor
	}
	var lines [][]styledRune
	cur := []styledRune{}
	for _, ca := range charAttrs {
		switch ca.R {
		case '\n':
			lines = append(lines, cur)
			cur = []styledRune{}
		case '\r':
		case '\t':
			for i := 0; i < tabWidth; i++ {
				cur = append(cur, styledRune{' ', ca.A})
			}
		default:
			cur = append(cur, styledRune{ca.R, ca.A})
		}
	}
	if len(cur) > 0 {
		lines = append(lines, cur)
	}

	imgHeight := len(lines)*lineHeight + marginTop + marginBot
	if imgHeight < lineHeight+marginTop+marginBot {
		imgHeight = lineHeight + marginTop + marginBot
	}

	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(img, img.Bounds(), &image.Uniform{renderBgAttrToColor(theme.Background)}, image.Point{}, draw.Src)

	defaultFg := renderAttrToColor(theme.Foreground)
	maxCharsPerLine := (imgWidth - marginLeft) / charWidth

	for lineIdx, line := range lines {
		y := marginTop + lineIdx*lineHeight
		x := marginLeft
		for i, sr := range line {
			if i >= maxCharsPerLine {
				break
			}
			fg := defaultFg
			if sr.fg != 0 {
				fg = renderAttrToColor(sr.fg)
			}
			burnfont.Draw(img, sr.r, x, y, fg.R, fg.G, fg.B)
			x += charWidth
		}
	}
	return img
}

// addViewportIndicator draws a semi-transparent overlay rectangle on the base
// image to indicate the visible viewport region.
func addViewportIndicator(base *image.RGBA, offsetY, visibleLines, totalLines int) {
	if totalLines <= 0 || visibleLines <= 0 {
		return
	}
	h := base.Bounds().Dy()
	w := base.Bounds().Dx()

	topFrac := float64(offsetY) / float64(totalLines)
	botFrac := float64(offsetY+visibleLines) / float64(totalLines)
	if botFrac > 1.0 {
		botFrac = 1.0
	}

	y0 := int(topFrac * float64(h))
	y1 := int(botFrac * float64(h))
	if y1 <= y0 {
		y1 = y0 + 1
	}
	if y1 > h {
		y1 = h
	}

	overlay := color.NRGBA{R: 255, G: 255, B: 255, A: 48}
	rect := image.Rect(0, y0, w, y1)
	draw.Draw(base, rect, &image.Uniform{overlay}, image.Point{}, draw.Over)
}

// sendMinimapKitty transmits a PNG image to the Kitty graphics protocol,
// positioned at (col, row) (1-indexed terminal coordinates) with the given
// cell dimensions.
func sendMinimapKitty(pngBytes []byte, col, row, cols, rows uint) {
	encoded := base64.StdEncoding.EncodeToString(pngBytes)
	fmt.Fprintf(os.Stdout, "\033[%d;%dH", row, col)

	const chunkSize = 4096
	total := len(encoded)
	for i := 0; i < total; i += chunkSize {
		end := i + chunkSize
		if end > total {
			end = total
		}
		chunk := encoded[i:end]
		isFirst := i == 0
		isLast := end >= total
		switch {
		case isFirst && isLast:
			fmt.Fprintf(os.Stdout, "\033_Ga=T,f=100,q=2,c=%d,r=%d;%s\033\\", cols, rows, chunk)
		case isFirst:
			fmt.Fprintf(os.Stdout, "\033_Ga=T,f=100,q=2,m=1,c=%d,r=%d;%s\033\\", cols, rows, chunk)
		case isLast:
			fmt.Fprintf(os.Stdout, "\033_Gm=0;%s\033\\", chunk)
		default:
			fmt.Fprintf(os.Stdout, "\033_Gm=1;%s\033\\", chunk)
		}
	}
}

// DrawMinimap renders and displays the minimap on the right side of the terminal.
// It uses a cached base image and only re-renders when the content (line count) changes.
// The viewport indicator is always redrawn to reflect the current scroll position.
func (e *Editor) DrawMinimap(c *vt.Canvas, theme Theme) {
	if !envKitty {
		return
	}

	termW := c.W() + minimapCols // full terminal width (c.W() is the editor area)
	termH := c.H()

	// The minimap occupies the rightmost minimapCols columns.
	col := termW - minimapCols + 1 // 1-indexed
	row := uint(1)
	rows := termH
	cols := uint(minimapCols)

	totalLines := e.Len()
	if totalLines == 0 {
		return
	}

	// Re-render the base image only when content has changed.
	if e.minimapCacheLines != totalLines || e.minimapCacheImg == nil {
		e.minimapCacheImg = e.renderEditorToImage(theme)
		e.minimapCacheLines = totalLines
	}

	// Copy the base image and draw the viewport indicator on the copy.
	bounds := e.minimapCacheImg.Bounds()
	composed := image.NewRGBA(bounds)
	draw.Draw(composed, bounds, e.minimapCacheImg, bounds.Min, draw.Src)

	offsetY := e.pos.OffsetY()
	visibleLines := int(termH)
	addViewportIndicator(composed, offsetY, visibleLines, totalLines)

	// Encode to PNG.
	var buf bytes.Buffer
	if err := png.Encode(&buf, composed); err != nil {
		return
	}

	// Compute display rows to fit the image height into available terminal rows.
	_, cellH := terminalCellPixels()
	imgH := uint(bounds.Dy())
	dispRows := rows
	if cellH > 0 {
		imgRows := (imgH + cellH - 1) / cellH
		if imgRows < rows {
			dispRows = imgRows
		}
	}

	sendMinimapKitty(buf.Bytes(), col, row, cols, dispRows)
}
