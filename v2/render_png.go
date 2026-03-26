package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"github.com/xyproto/burnfont"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

// renderAttrToColor converts a vt.AttributeColor (foreground ANSI code) to a color.NRGBA.
// Only the lower 16 bits (primary color) are used.
func renderAttrToColor(ac vt.AttributeColor) color.NRGBA {
	switch uint32(ac) & 0xFFFF {
	case 30:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	case 31:
		return color.NRGBA{R: 205, G: 0, B: 0, A: 255}
	case 32:
		return color.NRGBA{R: 0, G: 205, B: 0, A: 255}
	case 33:
		return color.NRGBA{R: 205, G: 205, B: 0, A: 255}
	case 34:
		return color.NRGBA{R: 0, G: 0, B: 238, A: 255}
	case 35:
		return color.NRGBA{R: 205, G: 0, B: 205, A: 255}
	case 36:
		return color.NRGBA{R: 0, G: 205, B: 205, A: 255}
	case 37:
		return color.NRGBA{R: 229, G: 229, B: 229, A: 255}
	case 90:
		return color.NRGBA{R: 127, G: 127, B: 127, A: 255}
	case 91:
		return color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	case 92:
		return color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	case 93:
		return color.NRGBA{R: 255, G: 255, B: 0, A: 255}
	case 94:
		return color.NRGBA{R: 0, G: 0, B: 255, A: 255}
	case 95:
		return color.NRGBA{R: 255, G: 0, B: 255, A: 255}
	case 96:
		return color.NRGBA{R: 0, G: 255, B: 255, A: 255}
	case 97:
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	default:
		return color.NRGBA{R: 229, G: 229, B: 229, A: 255}
	}
}

// renderBgAttrToColor converts a vt.AttributeColor (background ANSI code) to a color.NRGBA.
func renderBgAttrToColor(ac vt.AttributeColor) color.NRGBA {
	switch uint32(ac) & 0xFFFF {
	case 40:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	case 41:
		return color.NRGBA{R: 205, G: 0, B: 0, A: 255}
	case 42:
		return color.NRGBA{R: 0, G: 205, B: 0, A: 255}
	case 43:
		return color.NRGBA{R: 205, G: 205, B: 0, A: 255}
	case 44:
		return color.NRGBA{R: 0, G: 0, B: 238, A: 255}
	case 45:
		return color.NRGBA{R: 205, G: 0, B: 205, A: 255}
	case 46:
		return color.NRGBA{R: 0, G: 205, B: 205, A: 255}
	case 47:
		return color.NRGBA{R: 245, G: 245, B: 245, A: 255}
	default:
		// BackgroundDefault (49) and anything else: use a near-black dark background
		return color.NRGBA{R: 0x1e, G: 0x1e, B: 0x1e, A: 255}
	}
}

// textConfigFromTheme builds a TextConfig using the syntax color names from the given Theme.
func textConfigFromTheme(theme Theme) TextConfig {
	return TextConfig{
		AndOr:         theme.AndOr,
		AngleBracket:  theme.AngleBracket,
		AssemblyEnd:   theme.AssemblyEnd,
		Class:         theme.Class,
		Comment:       theme.Comment,
		Decimal:       theme.Decimal,
		Dollar:        theme.Dollar,
		Keyword:       theme.Keyword,
		Literal:       theme.Literal,
		Mut:           theme.Mut,
		Plaintext:     theme.Plaintext,
		Private:       theme.Private,
		Protected:     theme.Protected,
		Public:        theme.Public,
		Punctuation:   theme.Punctuation,
		Self:          theme.Self,
		Star:          theme.Star,
		Static:        theme.Static,
		String:        theme.String,
		Tag:           theme.Tag,
		TextAttrName:  theme.TextAttrName,
		TextAttrValue: theme.TextAttrValue,
		TextTag:       theme.TextTag,
		Type:          theme.Type,
		Whitespace:    theme.Whitespace,
	}
}

// renderFileToPNG renders a source code file with syntax highlighting using the
// given Orbiton theme, and saves the result to outputFilename.
// The output image is 256 pixels wide with a height that depends on the number of lines.
func renderFileToPNG(filename, outputFilename string, theme Theme) error {
	const (
		imgWidth     = 256
		charWidth    = 8
		lineHeight   = 10
		marginLeft   = 4
		marginTop    = 4
		marginBottom = 4
		tabWidth     = 4
	)

	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	detectedMode := mode.SimpleDetectBytes(data)
	cfg := textConfigFromTheme(theme)

	taggedText, err := AsText(data, detectedMode, func(c *TextConfig) { *c = cfg })
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
	currentLine := []styledRune{}
	for _, ca := range charAttrs {
		switch ca.R {
		case '\n':
			lines = append(lines, currentLine)
			currentLine = []styledRune{}
		case '\r':
			// skip carriage returns
		case '\t':
			for i := 0; i < tabWidth; i++ {
				currentLine = append(currentLine, styledRune{' ', ca.A})
			}
		default:
			currentLine = append(currentLine, styledRune{ca.R, ca.A})
		}
	}
	if len(currentLine) > 0 {
		lines = append(lines, currentLine)
	}

	imgHeight := len(lines)*lineHeight + marginTop + marginBottom
	if imgHeight < lineHeight+marginTop+marginBottom {
		imgHeight = lineHeight + marginTop + marginBottom
	}

	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

	// Fill background using the theme's background color
	draw.Draw(img, img.Bounds(), &image.Uniform{renderBgAttrToColor(theme.Background)}, image.Point{}, draw.Src)

	// Default foreground for uncolored characters (whitespace) uses the theme's foreground
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

	f, err := os.Create(outputFilename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
