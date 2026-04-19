package main

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"github.com/xyproto/imagepreview"
	"github.com/xyproto/vt"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed fonts/Vollkorn-Regular.ttf
var vollkornRegularTTF []byte

//go:embed fonts/Vollkorn-Italic.ttf
var vollkornItalicTTF []byte

//go:embed fonts/Montserrat-Bold.ttf
var montserratBoldTTF []byte

// Margin ratios for book mode (fraction of pixel dimensions).
const (
	bookMarginLeft   = 0.10
	bookMarginRight  = 0.05
	bookMarginTop    = 0.02
	bookMarginBottom = 0.02
)

// bookFontSet caches Vollkorn body faces, Montserrat Bold header faces and a small
// Montserrat Bold face for the status bar, all derived from the same base pixel size.
type bookFontSet struct {
	regular       font.Face
	italic        font.Face
	h1            font.Face
	h2            font.Face
	h3            font.Face
	h4            font.Face
	h5            font.Face
	statusBar     font.Face
	baseSize      float64
	h1Size        float64
	h2Size        float64
	h3Size        float64
	h4Size        float64
	h5Size        float64
	statusBarSize float64
}

func (fs *bookFontSet) headerForLevel(level int) font.Face {
	switch level {
	case 1:
		return fs.h1
	case 2:
		return fs.h2
	case 3:
		return fs.h3
	case 4:
		return fs.h4
	default:
		return fs.h5
	}
}

func (fs *bookFontSet) headerSizeForLevel(level int) float64 {
	switch level {
	case 1:
		return fs.h1Size
	case 2:
		return fs.h2Size
	case 3:
		return fs.h3Size
	case 4:
		return fs.h4Size
	default:
		return fs.h5Size
	}
}

var (
	bookFontMu              sync.Mutex
	bookFontCache           *bookFontSet
	parsedVollkornRegular   *opentype.Font
	parsedVollkornItalic    *opentype.Font
	parsedMontserratBold    *opentype.Font
	parseFontsOnce          sync.Once
	parseFontsErr           error
	bookContentCache        *image.RGBA
	bookContentCacheW       int
	bookContentCacheH       int
	bookContentCacheOffsetY int
	bookStatusMsg           string
	bookStatusMsgMu         sync.Mutex
)

func parsedFonts() error {
	parseFontsOnce.Do(func() {
		var err error
		if parsedVollkornRegular, err = opentype.Parse(vollkornRegularTTF); err != nil {
			parseFontsErr = err
			return
		}
		if parsedVollkornItalic, err = opentype.Parse(vollkornItalicTTF); err != nil {
			parseFontsErr = err
			return
		}
		if parsedMontserratBold, err = opentype.Parse(montserratBoldTTF); err != nil {
			parseFontsErr = err
		}
	})
	return parseFontsErr
}

// bookGraphicalMode returns true when book mode uses the Kitty/iTerm2 graphics
// protocol for rendering (terminal supports pixel-level image display).
func (e *Editor) bookGraphicalMode() bool {
	return e.bookMode.Load() && imagepreview.HasGraphics
}

// bookTextMode returns true when book mode uses VT100/xterm text rendering
// (vt100, vt220, xterm, xterm-color, xterm-256color, linux terminals).
func (e *Editor) bookTextMode() bool {
	return e.bookMode.Load() && !imagepreview.HasGraphics
}

// bookModeSetStatusMsg atomically sets the temporary status bar message.
func bookModeSetStatusMsg(msg string) {
	bookStatusMsgMu.Lock()
	bookStatusMsg = msg
	bookStatusMsgMu.Unlock()
}

// bookModeGetStatusMsg atomically reads the temporary status bar message.
func bookModeGetStatusMsg() string {
	bookStatusMsgMu.Lock()
	defer bookStatusMsgMu.Unlock()
	return bookStatusMsg
}

func newFace(f *opentype.Font, size float64) (font.Face, error) {
	// TODO: Find better DPI values
	const defaultDPI = 72
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     defaultDPI,
		Hinting: font.HintingFull,
	})
}

// bookFaces returns a font set for the requested base pixel size, using a
// cached set when the size hasn't changed.
func bookFaces(pixelSize float64) (*bookFontSet, error) {
	bookFontMu.Lock()
	defer bookFontMu.Unlock()
	if bookFontCache != nil && bookFontCache.baseSize == pixelSize {
		return bookFontCache, nil
	}
	if err := parsedFonts(); err != nil {
		return nil, err
	}
	reg, err := newFace(parsedVollkornRegular, pixelSize)
	if err != nil {
		return nil, err
	}
	ita, err := newFace(parsedVollkornItalic, pixelSize)
	if err != nil {
		return nil, err
	}
	h1Size := pixelSize * 2.1
	h2Size := pixelSize * 1.9
	h3Size := pixelSize * 1.6
	h4Size := pixelSize * 1.3
	h5Size := pixelSize * 1.1
	h1, err := newFace(parsedMontserratBold, h1Size)
	if err != nil {
		return nil, err
	}
	h2, err := newFace(parsedMontserratBold, h2Size)
	if err != nil {
		return nil, err
	}
	h3, err := newFace(parsedMontserratBold, h3Size)
	if err != nil {
		return nil, err
	}
	h4, err := newFace(parsedMontserratBold, h4Size)
	if err != nil {
		return nil, err
	}
	h5, err := newFace(parsedMontserratBold, h5Size)
	if err != nil {
		return nil, err
	}
	statusBarSize := pixelSize * 0.75
	if statusBarSize < 8 {
		statusBarSize = 8
	}
	sb, err := newFace(parsedMontserratBold, statusBarSize)
	if err != nil {
		return nil, err
	}
	bookFontCache = &bookFontSet{
		regular:       reg,
		italic:        ita,
		h1:            h1,
		h2:            h2,
		h3:            h3,
		h4:            h4,
		h5:            h5,
		statusBar:     sb,
		h1Size:        h1Size,
		h2Size:        h2Size,
		h3Size:        h3Size,
		h4Size:        h4Size,
		h5Size:        h5Size,
		statusBarSize: statusBarSize,
		baseSize:      pixelSize,
	}
	return bookFontCache, nil
}

// faceAscent returns the ascent in pixels, with fallback to a font-size estimate.
func faceAscent(face font.Face, fontSizePx float64) int {
	a := face.Metrics().Ascent.Round()
	if a <= 0 {
		a = int(fontSizePx*0.8 + 0.5)
	}
	if a <= 0 {
		a = 10
	}
	return a
}

// ── Inline Markdown parsing ────────────────────────────────────────────────

// textSegment is one styled run of text within a body line.
type textSegment struct {
	text      string
	bold      bool
	italic    bool
	underline bool
}

// parseLineSegments converts a line with Markdown-like inline markers into
// styled segments. Markers consumed (not rendered):
//
//	***text***     bold + italic
//	**text**       bold
//	*text*         italic
//	__text__       underline
//	`code`         italic (code style; backticks stripped)
//	[text](url)    underlined link text (URL discarded)
//	![alt](url)    skipped entirely (image)
func parseLineSegments(line string) []textSegment {
	flush := func(segs []textSegment, cur *strings.Builder, bold, italic, underline bool) []textSegment {
		if cur.Len() > 0 {
			segs = append(segs, textSegment{text: cur.String(), bold: bold, italic: italic, underline: underline})
			cur.Reset()
		}
		return segs
	}
	type state struct{ bold, italic, underline bool }
	var (
		segs  []textSegment
		cur   strings.Builder
		st    state
		runes = []rune(line)
	)
	for i := 0; i < len(runes); {
		// images
		if runes[i] == '!' && i+1 < len(runes) && runes[i+1] == '[' {
			j := i + 2
			for j < len(runes) && runes[j] != ']' {
				j++
			}
			if j < len(runes) && j+1 < len(runes) && runes[j+1] == '(' {
				k := j + 2
				for k < len(runes) && runes[k] != ')' {
					k++
				}
				if k < len(runes) {
					segs = flush(segs, &cur, st.bold, st.italic, st.underline)
					i = k + 1
					continue
				}
			}
			cur.WriteRune(runes[i])
			i++
			continue
		}
		// [text](url) links are rendered as underlined text
		if runes[i] == '[' {
			j := i + 1
			for j < len(runes) && runes[j] != ']' {
				j++
			}
			if j < len(runes) && j+1 < len(runes) && runes[j+1] == '(' {
				k := j + 2
				for k < len(runes) && runes[k] != ')' {
					k++
				}
				if k < len(runes) {
					segs = flush(segs, &cur, st.bold, st.italic, st.underline)
					linkText := string(runes[i+1 : j])
					segs = append(segs, textSegment{text: linkText, bold: st.bold, italic: st.italic, underline: true})
					i = k + 1
					continue
				}
			}
			cur.WriteRune(runes[i])
			i++
			continue
		}
		// inline code is rendered as italic text with backticks stripped
		if runes[i] == '`' {
			j := i + 1
			for j < len(runes) && runes[j] != '`' {
				j++
			}
			if j < len(runes) {
				segs = flush(segs, &cur, st.bold, st.italic, st.underline)
				codeText := string(runes[i+1 : j])
				segs = append(segs, textSegment{text: codeText, bold: st.bold, italic: true, underline: st.underline})
				i = j + 1
				continue
			}
			cur.WriteRune(runes[i])
			i++
			continue
		}
		if i+2 < len(runes) && runes[i] == '*' && runes[i+1] == '*' && runes[i+2] == '*' {
			segs = flush(segs, &cur, st.bold, st.italic, st.underline)
			st.bold = !st.bold
			st.italic = !st.italic
			i += 3
			continue
		}
		if i+1 < len(runes) && runes[i] == '*' && runes[i+1] == '*' {
			segs = flush(segs, &cur, st.bold, st.italic, st.underline)
			st.bold = !st.bold
			i += 2
			continue
		}
		if runes[i] == '*' {
			segs = flush(segs, &cur, st.bold, st.italic, st.underline)
			st.italic = !st.italic
			i++
			continue
		}
		if i+1 < len(runes) && runes[i] == '_' && runes[i+1] == '_' {
			segs = flush(segs, &cur, st.bold, st.italic, st.underline)
			st.underline = !st.underline
			i += 2
			continue
		}
		cur.WriteRune(runes[i])
		i++
	}
	segs = flush(segs, &cur, st.bold, st.italic, st.underline)
	return segs
}

type lineKind int

const (
	lineKindBody      lineKind = iota
	lineKindHeader             // level stored in parsedLine.headerLevel
	lineKindBullet             // - or * list item
	lineKindNumbered           // 1. 2. ... list item
	lineKindUnchecked          // - [ ] item
	lineKindChecked            // - [x] item
	lineKindRule               // --- horizontal rule
	lineKindBlank
	lineKindImage // standalone ![alt](path) image
)

type parsedLine struct {
	kind        lineKind
	headerLevel int    // 1, 2, 3, 4, 5 for lineKindHeader
	indent      int    // leading spaces & 2 (list nesting depth)
	prefix      string // rendered prefix, e.g. "• ", "1. ", "☐ ", "☑ "
	body        string // text after the prefix, may contain inline markers
}

// isHorizontalRule reports whether the line is a Markdown thematic break
// (three or more - or = characters, optionally separated by spaces).
// '*' is intentionally excluded: it is overloaded for bold/italic markers
// and list bullets, so "***" or "****" must never become a horizontal rule.
func isHorizontalRule(line string) bool {
	stripped := strings.TrimSpace(line)
	if len(stripped) < 3 {
		return false
	}
	r := rune(stripped[0])
	if r != '-' && r != '=' {
		return false
	}
	count := 0
	for _, c := range stripped {
		if c == r {
			count++
		} else if c != ' ' {
			return false
		}
	}
	return count >= 3
}

// parseBookLine converts a raw Markdown line into a parsedLine.
func parseBookLine(line string) parsedLine {
	if strings.TrimSpace(line) == "" {
		return parsedLine{kind: lineKindBlank}
	}
	if isHorizontalRule(line) {
		return parsedLine{kind: lineKindRule}
	}
	// Count leading spaces for indent depth
	indent := 0
	for _, r := range line {
		if r == ' ' {
			indent++
		} else {
			break
		}
	}
	trimmed := line[indent:]
	indentDepth := indent / 2
	// Fenced code block markers (``` or ~~~): render as blank (hide the fence line)
	if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
		return parsedLine{kind: lineKindBlank}
	}
	// For blockquoted text, render with a "│ " prefix
	if strings.HasPrefix(trimmed, "> ") || trimmed == ">" {
		inner := ""
		if strings.HasPrefix(trimmed, "> ") {
			inner = trimmed[2:]
		}
		return parsedLine{
			kind:   lineKindBullet,
			indent: indentDepth,
			prefix: strings.Repeat("  ", indentDepth) + "│ ",
			body:   inner,
		}
	}
	// Markdown images
	if strings.HasPrefix(trimmed, "![") {
		j := strings.Index(trimmed[2:], "](")
		if j >= 0 {
			pathStart := 2 + j + 2
			k := strings.Index(trimmed[pathStart:], ")")
			if k >= 0 && strings.TrimSpace(trimmed[pathStart+k+1:]) == "" {
				imgPath := trimmed[pathStart : pathStart+k]
				return parsedLine{kind: lineKindImage, body: imgPath}
			}
		}
	}
	// Headers
	if indent == 0 {
		switch {
		case strings.HasPrefix(trimmed, "##### "):
			return parsedLine{kind: lineKindHeader, headerLevel: 5, body: trimmed[6:]}
		case strings.HasPrefix(trimmed, "#### "):
			return parsedLine{kind: lineKindHeader, headerLevel: 4, body: trimmed[5:]}
		case strings.HasPrefix(trimmed, "### "):
			return parsedLine{kind: lineKindHeader, headerLevel: 3, body: trimmed[4:]}
		case strings.HasPrefix(trimmed, "## "):
			return parsedLine{kind: lineKindHeader, headerLevel: 2, body: trimmed[3:]}
		case strings.HasPrefix(trimmed, "# "):
			return parsedLine{kind: lineKindHeader, headerLevel: 1, body: trimmed[2:]}
		}
	}
	// Checkboxes
	if strings.HasPrefix(trimmed, "- [ ] ") || strings.HasPrefix(trimmed, "- [ ]\t") {
		return parsedLine{
			kind:   lineKindUnchecked,
			indent: indentDepth,
			prefix: strings.Repeat("  ", indentDepth) + "☐ ",
			body:   strings.TrimSpace(trimmed[6:]),
		}
	}
	if strings.HasPrefix(trimmed, "- [x] ") || strings.HasPrefix(trimmed, "- [X] ") ||
		strings.HasPrefix(trimmed, "- [x]\t") || strings.HasPrefix(trimmed, "- [X]\t") {
		return parsedLine{
			kind:   lineKindChecked,
			indent: indentDepth,
			prefix: strings.Repeat("  ", indentDepth) + "☑ ",
			body:   strings.TrimSpace(trimmed[6:]),
		}
	}
	// Unordered lists
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		return parsedLine{
			kind:   lineKindBullet,
			indent: indentDepth,
			prefix: strings.Repeat("  ", indentDepth) + "• ",
			body:   trimmed[2:],
		}
	}
	// Numbered lists
	for i, r := range trimmed {
		if unicode.IsDigit(r) {
			continue
		}
		if i > 0 && r == '.' && i+2 <= len(trimmed) && trimmed[i+1] == ' ' {
			num := trimmed[:i+2] // "1. "
			return parsedLine{
				kind:   lineKindNumbered,
				indent: indentDepth,
				prefix: strings.Repeat("  ", indentDepth) + num,
				body:   trimmed[i+2:],
			}
		}
		break
	}
	return parsedLine{kind: lineKindBody, body: line}
}

// faceForSeg returns the font face for a text segment. Bold is served by the
// regular Vollkorn face (faux bold is applied by the caller).
// Italic uses the Vollkorn Italic face.
func faceForSeg(fs *bookFontSet, seg textSegment) font.Face {
	if seg.italic {
		return fs.italic
	}
	return fs.regular
}

// drawString renders text at (x, baselineY) using face and returns the new X
func drawString(img *image.RGBA, face font.Face, x, baselineY int, text string, clr color.Color) int {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(clr),
		Face: face,
		Dot:  fixed.P(x, baselineY),
	}
	d.DrawString(text)
	return d.Dot.X.Round()
}

// drawSegments renders styled inline segments starting at (x, baselineY),
// returning the final X. Bold uses faux-bold (two draws, 1 px apart)
func drawSegments(img *image.RGBA, fs *bookFontSet, x, baselineY int, segs []textSegment, clr color.Color) int {
	for _, seg := range segs {
		face := faceForSeg(fs, seg)
		endX := drawString(img, face, x, baselineY, seg.text, clr)
		if seg.bold {
			// Faux bold: draw again 1 px to the right for thicker strokes
			drawString(img, face, x+1, baselineY, seg.text, clr)
		}
		if seg.underline {
			ulY := baselineY + 2
			if ulY < img.Bounds().Max.Y {
				right := endX
				if seg.bold {
					right++
				}
				for px := x; px < right; px++ {
					img.Set(px, ulY, clr)
				}
			}
		}
		x = endX
		if seg.bold {
			x++ // account for the extra pixel
		}
	}
	return x
}

// measureSegmentsToRune measures the pixel width up to (but not including)
// the nth visual rune across all segments.
func measureSegmentsToRune(fs *bookFontSet, segs []textSegment, targetRune int) int {
	total := fixed.Int26_6(0)
	col := 0
	for _, seg := range segs {
		face := faceForSeg(fs, seg)
		for _, r := range seg.text {
			if col >= targetRune {
				return total.Round()
			}
			adv, ok := face.GlyphAdvance(r)
			if ok {
				total += adv
			}
			if seg.bold {
				total += fixed.I(1)
			}
			col++
		}
	}
	return total.Round()
}

// drawCursorBar draws a 2-pixel-wide vertical I-beam cursor between top and
// bottom (exclusive). Callers should pass the pixel bounds of the actual text
// region (baseline – ascent … baseline + descent) rather than the full cell.
func drawCursorBar(img *image.RGBA, x, top, bottom int) {
	clr := color.NRGBA{0x00, 0x55, 0xCC, 0xFF} // blue cursor, always visible on white
	for py := top; py < bottom; py++ {
		img.Set(x, py, clr)
		img.Set(x+1, py, clr)
	}
}

var (
	bookImgCache   = map[string]image.Image{}
	bookImgCacheMu sync.Mutex
)

// bookLoadImage loads and caches an image by absolute path.
// Returns nil if the file does not exist or cannot be decoded.
func bookLoadImage(absPath string) image.Image {
	bookImgCacheMu.Lock()
	defer bookImgCacheMu.Unlock()
	if img, ok := bookImgCache[absPath]; ok {
		return img
	}
	nimg, err := imagepreview.LoadImage(absPath)
	if err != nil {
		bookImgCache[absPath] = nil
		return nil
	}
	bookImgCache[absPath] = nimg
	return nimg
}

// resolveBookImagePath resolves an image path relative to the document
func (e *Editor) resolveBookImagePath(imgPath string) string {
	if filepath.IsAbs(imgPath) {
		return imgPath
	}
	return filepath.Join(filepath.Dir(e.filename), imgPath)
}

// bookScaleImage scales src to fit within (maxW, maxH), preserving aspect ratio.
// Never upscales. Uses nearest-neighbour resampling.
func bookScaleImage(src image.Image, maxW, maxH int) image.Image {
	bounds := src.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW == 0 || srcH == 0 {
		return src
	}
	scaleX := float64(maxW) / float64(srcW)
	scaleY := float64(maxH) / float64(srcH)
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	if scale >= 1.0 {
		return src
	}
	dstW := max(int(float64(srcW)*scale), 1)
	dstH := max(int(float64(srcH)*scale), 1)
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := range dstH {
		sy := int(float64(y)/scale) + bounds.Min.Y
		for x := range dstW {
			sx := int(float64(x)/scale) + bounds.Min.X
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

// bookMaxImageRows is the default maximum height (in terminal rows) for inline images
const bookMaxImageRows = 8

// bookImageRows returns the number of display rows a Markdown image line will
// occupy. Returns 1 if the image cannot be loaded.
func (e *Editor) bookImageRows(imgPath string, lineH, maxW, maxAvailH int) int {
	abs := e.resolveBookImagePath(imgPath)
	src := bookLoadImage(abs)
	if src == nil {
		return 1
	}
	bounds := src.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW == 0 || srcH == 0 {
		return 1
	}
	maxH := min(bookMaxImageRows*lineH, maxAvailH)
	scaleX := float64(maxW) / float64(srcW)
	scaleY := float64(maxH) / float64(srcH)
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	if scale > 1.0 {
		scale = 1.0
	}
	scaledH := max(int(float64(srcH)*scale), 1)
	rows := max((scaledH+lineH-1)/lineH, 1)
	return rows
}

// bookDrawInlineImage loads, scales and draws a Markdown image into img at
// (marginLeft, cellTop). Returns the number of display rows consumed.
func (e *Editor) bookDrawInlineImage(img *image.RGBA, imgPath string, marginLeft, marginRight, cellTop, availPixH, lineH int) int {
	abs := e.resolveBookImagePath(imgPath)
	src := bookLoadImage(abs)
	if src == nil {
		return 1
	}
	maxW := marginRight - marginLeft
	maxH := min(bookMaxImageRows*lineH, availPixH)
	scaled := bookScaleImage(src, maxW, maxH)
	b := scaled.Bounds()
	dstRect := image.Rect(marginLeft, cellTop, marginLeft+b.Dx(), cellTop+b.Dy())
	draw.Draw(img, dstRect, scaled, b.Min, draw.Src)
	rows := max((b.Dy()+lineH-1)/lineH, 1)
	return rows
}

// bookBresenhamLine draws a 1-pixel-wide line using Bresenham's algorithm.
func bookBresenhamLine(img *image.RGBA, x1, y1, x2, y2 int, clr color.Color) {
	dx, dy := x2-x1, y2-y1
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	sx, sy := 1, 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy
	for {
		img.Set(x1, y1, clr)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// bookDrawCheckbox draws a checkbox icon centred vertically within (cellTop,
// cellTop+lineH). checked=true draws a green fill with a checkmark.
// Returns the X coordinate just after the icon (including a small gap).
func bookDrawCheckbox(img *image.RGBA, x, cellTop, lineH int, checked bool) int {
	size := max(
		// ~55 % of line height
		lineH*55/100, 5)
	topY := cellTop + (lineH-size)/2

	border := color.NRGBA{0x55, 0x55, 0x55, 0xff}
	var fill color.NRGBA
	if checked {
		fill = color.NRGBA{0xcc, 0xf0, 0xcc, 0xff} // light green
	} else {
		fill = color.NRGBA{0xf8, 0xf8, 0xf8, 0xff} // near-white
	}

	// Interior fill
	for py := topY + 1; py < topY+size-1; py++ {
		for px := x + 1; px < x+size-1; px++ {
			img.Set(px, py, fill)
		}
	}
	// Border
	for px := x; px < x+size; px++ {
		img.Set(px, topY, border)
		img.Set(px, topY+size-1, border)
	}
	for py := topY; py < topY+size; py++ {
		img.Set(x, py, border)
		img.Set(x+size-1, py, border)
	}

	if checked {
		// Draw a ✓ checkmark with two thick Bresenham strokes.
		checkClr := color.NRGBA{0x10, 0x80, 0x10, 0xff}
		pad := max(size/5, 1)
		// Short left arm: inner-left → mid-bottom
		lx, ly := x+pad, topY+size*2/3
		midX, midY := x+size*2/5, topY+size-1-pad
		// Long right arm: mid-bottom → upper-right
		rx, ry := x+size-1-pad, topY+pad
		bookBresenhamLine(img, lx, ly, midX, midY, checkClr)
		bookBresenhamLine(img, lx+1, ly, midX+1, midY, checkClr)
		bookBresenhamLine(img, midX, midY, rx, ry, checkClr)
		bookBresenhamLine(img, midX+1, midY, rx+1, ry, checkClr)
	}

	return x + size + 3 // gap after icon
}

// rawXToVisualX converts a raw rune index within line to the equivalent
// visual column after stripping inline markdown markers (bold/italic/code).
func rawXToVisualX(line string, rawX int) int {
	runes := []rune(line)
	if rawX > len(runes) {
		rawX = len(runes)
	}
	vis := 0
	i := 0
	for i < rawX {
		if i+2 < len(runes) && runes[i] == '*' && runes[i+1] == '*' && runes[i+2] == '*' {
			i += 3
			continue
		}
		if i+1 < len(runes) && runes[i] == '*' && runes[i+1] == '*' {
			i += 2
			continue
		}
		if runes[i] == '*' {
			i++
			continue
		}
		if i+1 < len(runes) && runes[i] == '_' && runes[i+1] == '_' {
			i += 2
			continue
		}
		// Backtick (inline code marker): skip only the backtick itself.
		if runes[i] == '`' {
			i++
			continue
		}
		vis++
		i++
	}
	return vis
}

// bookContentImage renders the visible document lines to a white RGBA image
// with proper Markdown formatting. It does NOT draw the cursor; call
// bookOverlayCursor on the result to add the I-beam before displaying.
func (e *Editor) bookContentImage(pixW, pixH, editRows int, cellH uint) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, pixW, pixH))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	if editRows <= 0 || pixW <= 0 || pixH <= 0 {
		return img
	}

	fontSize := float64(cellH) * 0.72
	if fontSize < 6 {
		fontSize = 6
	}

	fs, err := bookFaces(fontSize)
	if err != nil {
		return img
	}

	ascent := faceAscent(fs.regular, fontSize)

	lineH := int(cellH)
	if lineH <= 0 {
		lineH = int(fontSize) + 4
	}

	marginLeft := int(float64(pixW) * bookMarginLeft)
	marginTop := int(float64(pixH) * bookMarginTop)
	marginBottom := int(float64(pixH) * bookMarginBottom)

	dark := color.NRGBA{0x10, 0x10, 0x10, 0xff}

	maxLines := min((pixH-marginTop-marginBottom)/lineH, editRows)
	rightMargin := pixW - int(float64(pixW)*bookMarginRight)

	startLine := e.pos.offsetY
	totalLines := e.Len()

	// Use separate document-line (docLine) and display-row (row) counters so that
	// image lines can consume multiple display rows.
	docLine := startLine
	for row := 0; row < maxLines && docLine < totalLines; {
		rawLine := e.Line(LineIndex(docLine))
		rawLine = strings.ReplaceAll(rawLine, "\t", "    ")
		pl := parseBookLine(rawLine)
		cellTop := marginTop + row*lineH
		docLine++

		switch pl.kind {
		case lineKindImage:
			rowsUsed := e.bookDrawInlineImage(img, pl.body, marginLeft, rightMargin, cellTop, pixH-marginBottom-cellTop, lineH)
			row += rowsUsed
			continue // no right-margin clip needed for images

		case lineKindBlank:
			// nothing to draw for a blank line

		case lineKindRule:
			ruleY := cellTop + lineH/2
			for px := marginLeft; px < rightMargin; px++ {
				img.Set(px, ruleY, dark)
				img.Set(px, ruleY+1, dark)
			}

		case lineKindHeader:
			hFace := fs.headerForLevel(pl.headerLevel)
			hAscent := faceAscent(hFace, fs.headerSizeForLevel(pl.headerLevel))
			vPad := max((lineH-hAscent)/2, 0)
			d := &font.Drawer{
				Dst:  img,
				Src:  image.NewUniform(dark),
				Face: hFace,
				Dot:  fixed.P(marginLeft, cellTop+vPad+hAscent),
			}
			d.DrawString(pl.body)

		case lineKindUnchecked:
			prefixX := bookDrawCheckbox(img, marginLeft, cellTop, lineH, false)
			vPad := max((lineH-ascent)/2, 0)
			drawSegments(img, fs, prefixX, cellTop+vPad+ascent, parseLineSegments(pl.body), dark)

		case lineKindChecked:
			prefixX := bookDrawCheckbox(img, marginLeft, cellTop, lineH, true)
			vPad := max((lineH-ascent)/2, 0)
			bodyClr := color.NRGBA{0x55, 0x55, 0x55, 0xff} // slightly grayed
			drawSegments(img, fs, prefixX, cellTop+vPad+ascent, parseLineSegments(pl.body), bodyClr)

		case lineKindBullet, lineKindNumbered:
			vPad := max((lineH-ascent)/2, 0)
			baseline := cellTop + vPad + ascent
			prefixX := drawString(img, fs.regular, marginLeft, baseline, pl.prefix, dark)
			drawSegments(img, fs, prefixX, baseline, parseLineSegments(pl.body), dark)

		default: // lineKindBody
			vPad := max((lineH-ascent)/2, 0)
			drawSegments(img, fs, marginLeft, cellTop+vPad+ascent, parseLineSegments(pl.body), dark)
		}

		// Hard-clip any text that overflowed past the right margin.
		if rightMargin < pixW {
			draw.Draw(img, image.Rect(rightMargin, cellTop, pixW, cellTop+lineH), image.White, image.Point{}, draw.Src)
		}
		row++
	}

	return img
}

// bookOverlayCursor draws the I-beam cursor onto dst (which should be a copy
// of the content image) at the position matching the editor's current cursor.
func (e *Editor) bookOverlayCursor(dst *image.RGBA, pixW, pixH, editRows int, cellH uint) {
	fontSize := float64(cellH) * 0.72
	if fontSize < 6 {
		fontSize = 6
	}

	fs, err := bookFaces(fontSize)
	if err != nil {
		return
	}

	ascent := faceAscent(fs.regular, fontSize)
	lineH := int(cellH)
	if lineH <= 0 {
		lineH = int(fontSize) + 4
	}

	// Compute cursor extent for the regular face.
	regDescent := fs.regular.Metrics().Descent.Round()
	if regDescent <= 0 {
		regDescent = int(fontSize*0.2 + 0.5)
	}
	regVPad := max((lineH-ascent)/2, 0)

	marginLeft := int(float64(pixW) * bookMarginLeft)
	marginTop := int(float64(pixH) * bookMarginTop)
	marginBottom := int(float64(pixH) * bookMarginBottom)

	maxLines := min((pixH-marginTop-marginBottom)/lineH, editRows)
	marginRight := pixW - int(float64(pixW)*bookMarginRight)

	cursorDataY := int(e.DataY())
	cursorRawX := e.pos.sx + e.pos.offsetX
	startLine := e.pos.offsetY
	totalLines := e.Len()

	// Map cursor document line to its display row, accounting for multi-row images.
	cursorDisplayRow := -1
	{
		dl := startLine
		for row := 0; row < maxLines && dl < totalLines; {
			if dl == cursorDataY {
				cursorDisplayRow = row
				break
			}
			rl := e.Line(LineIndex(dl))
			rl = strings.ReplaceAll(rl, "\t", "    ")
			if parseBookLine(rl).kind == lineKindImage {
				rowsConsumed := e.bookImageRows(parseBookLine(rl).body, lineH, marginRight-marginLeft, pixH-marginTop-marginBottom)
				row += rowsConsumed
			} else {
				row++
			}
			dl++
		}
	}
	if cursorDisplayRow < 0 {
		return
	}

	rawLine := e.Line(LineIndex(cursorDataY))
	rawLine = strings.ReplaceAll(rawLine, "\t", "    ")
	pl := parseBookLine(rawLine)
	cellTop := marginTop + cursorDisplayRow*lineH

	// maxBottom ensures the cursor bar never renders inside the bottom margin.
	maxBottom := pixH - marginBottom - 1

	// If cursor is on an image line, show the I-beam at the left margin.
	if pl.kind == lineKindImage {
		drawCursorBar(dst, marginLeft, cellTop+regVPad, min(cellTop+regVPad+ascent+regDescent, maxBottom))
		return
	}

	switch pl.kind {
	case lineKindBlank:
		drawCursorBar(dst, marginLeft, cellTop+regVPad, min(cellTop+regVPad+ascent+regDescent, maxBottom))

	case lineKindHeader:
		hFace := fs.headerForLevel(pl.headerLevel)
		hSize := fs.headerSizeForLevel(pl.headerLevel)
		hAscent := faceAscent(hFace, hSize)
		hDescent := hFace.Metrics().Descent.Round()
		if hDescent <= 0 {
			hDescent = int(hSize*0.2 + 0.5)
		}
		hVPad := max((lineH-hAscent)/2, 0)
		prefixLen := pl.headerLevel + 1
		adjRawX := max(cursorRawX-prefixLen, 0)
		bodyRunes := []rune(pl.body)
		if adjRawX > len(bodyRunes) {
			adjRawX = len(bodyRunes)
		}
		hd := &font.Drawer{Face: hFace}
		adv := hd.MeasureString(string(bodyRunes[:adjRawX]))
		drawCursorBar(dst, marginLeft+adv.Round(), cellTop+hVPad, min(cellTop+hVPad+hAscent+hDescent, maxBottom))

	case lineKindBullet, lineKindUnchecked, lineKindChecked, lineKindNumbered:
		rawPrefix := rawMarkdownPrefix(rawLine)
		rawBodyStart := len([]rune(rawPrefix))
		bodyRawX := cursorRawX - rawBodyStart
		segs := parseLineSegments(pl.body)
		// Compute prefix pixel width (checkbox icon or font-rendered prefix).
		var prefixEndX int
		if pl.kind == lineKindUnchecked || pl.kind == lineKindChecked {
			size := max(lineH*55/100, 5)
			prefixEndX = marginLeft + size + 3
		} else {
			prefixEndX = marginLeft + (&font.Drawer{Face: fs.regular}).MeasureString(pl.prefix).Round()
		}
		var cursorX int
		if bodyRawX < 0 {
			cursorX = prefixEndX
		} else {
			bodyVisX := rawXToVisualX(pl.body, bodyRawX)
			cursorX = prefixEndX + measureSegmentsToRune(fs, segs, bodyVisX)
		}
		drawCursorBar(dst, cursorX, cellTop+regVPad, min(cellTop+regVPad+ascent+regDescent, maxBottom))

	default: // lineKindBody
		segs := parseLineSegments(pl.body)
		visX := rawXToVisualX(rawLine, cursorRawX)
		drawCursorBar(dst, marginLeft+measureSegmentsToRune(fs, segs, visX), cellTop+regVPad, min(cellTop+regVPad+ascent+regDescent, maxBottom))
	}
}

// rawMarkdownPrefix returns the raw syntax prefix of a list/todo line
// (the part before the body text), in runes, so we can compute body offsets.
func rawMarkdownPrefix(line string) string {
	trimmed := strings.TrimLeft(line, " ")
	indent := len(line) - len(trimmed)
	indentStr := line[:indent]

	for _, pfx := range []string{"- [ ] ", "- [x] ", "- [X] ", "- ", "* "} {
		if strings.HasPrefix(trimmed, pfx) {
			return indentStr + pfx
		}
	}
	// Numbered list: digits + ". "
	for i, r := range trimmed {
		if unicode.IsDigit(r) {
			continue
		}
		if i > 0 && r == '.' && i+2 <= len(trimmed) && trimmed[i+1] == ' ' {
			return indentStr + trimmed[:i+2]
		}
		break
	}
	return ""
}

// writeBookTextSegs writes parsed inline Markdown segments to the canvas at
// (startX, y), advancing x for each rune. maxX is the right pixel bound.
func writeBookTextSegs(c *vt.Canvas, startX uint, y uint, segs []textSegment, maxX int) {
	x := startX
	for _, seg := range segs {
		if int(x) >= maxX {
			break
		}
		var fg, bg vt.AttributeColor
		switch {
		case seg.bold && seg.italic:
			fg, bg = vt.Bold, vt.Italic // Combine → \033[1;3m
		case seg.bold:
			fg, bg = vt.Bold, vt.DefaultBackground
		case seg.italic:
			fg, bg = vt.Italic, vt.DefaultBackground
		default:
			fg, bg = vt.Default, vt.DefaultBackground
		}
		if seg.underline {
			bg = vt.Underscore // encode underline in bg → Combine adds \033[4m
		}
		runes := []rune(seg.text)
		if rem := maxX - int(x); len(runes) > rem {
			runes = runes[:rem]
		}
		if len(runes) > 0 {
			c.Write(x, y, fg, bg, string(runes))
			x += uint(len(runes))
		}
	}
}

// bookTextModeRender renders the visible document using VT100/ANSI escape codes
// for terminals without Kitty/iTerm2 graphics support. It writes styled lines
// directly to the canvas buffer; the caller must follow up with HideCursorAndDraw.
func (e *Editor) bookTextModeRender(c *vt.Canvas) {
	if c == nil {
		return
	}
	w := int(c.Width())
	h := int(c.Height())
	editRows := h - 1 // last row reserved for the status bar
	if editRows <= 0 || w <= 0 {
		return
	}

	marginLeft := max(int(float64(w)*bookMarginLeft), 2)
	marginRight := w - max(int(float64(w)*bookMarginRight), 1)

	// Clear the editing rows with default colours.
	blank := strings.Repeat(" ", w)
	for row := range editRows {
		c.Write(0, uint(row), vt.Default, vt.DefaultBackground, blank)
	}

	startLine := e.pos.offsetY
	totalLines := e.Len()

	for i := range editRows {
		lineIdx := startLine + i
		if lineIdx >= totalLines {
			break
		}
		rawLine := e.Line(LineIndex(lineIdx))
		rawLine = strings.ReplaceAll(rawLine, "\t", "    ")
		pl := parseBookLine(rawLine)
		y := uint(i)
		x := uint(marginLeft)

		switch pl.kind {
		case lineKindBlank:
			// nothing

		case lineKindRule:
			width := marginRight - marginLeft
			if width > 0 {
				c.Write(x, y, vt.Dim, vt.DefaultBackground, strings.Repeat("─", width))
			}

		case lineKindHeader:
			var fg vt.AttributeColor
			switch pl.headerLevel {
			case 1:
				fg = vt.White // bg=Bold gives \033[97;1m (bright white + bold)
			case 2:
				fg = vt.LightCyan
			default:
				fg = vt.LightGray
			}
			text := pl.body
			runes := []rune(text)
			if len(runes) > marginRight-marginLeft {
				runes = runes[:marginRight-marginLeft]
				text = string(runes)
			}
			c.Write(x, y, fg, vt.Bold, text) // bg=Bold → fg.Combine(Bold) adds bold attr

		case lineKindBullet:
			prefixFg := vt.Default
			if strings.HasPrefix(pl.prefix, "│") {
				prefixFg = vt.Dim // blockquote bar slightly dimmed
			}
			c.Write(x, y, prefixFg, vt.DefaultBackground, pl.prefix)
			writeBookTextSegs(c, x+uint(len([]rune(pl.prefix))), y, parseLineSegments(pl.body), marginRight)

		case lineKindUnchecked:
			c.Write(x, y, vt.Default, vt.DefaultBackground, pl.prefix)
			writeBookTextSegs(c, x+uint(len([]rune(pl.prefix))), y, parseLineSegments(pl.body), marginRight)

		case lineKindChecked:
			c.Write(x, y, vt.Green, vt.Bold, pl.prefix) // green + bold ✓
			writeBookTextSegs(c, x+uint(len([]rune(pl.prefix))), y, parseLineSegments(pl.body), marginRight)

		case lineKindNumbered:
			c.Write(x, y, vt.Default, vt.DefaultBackground, pl.prefix)
			writeBookTextSegs(c, x+uint(len([]rune(pl.prefix))), y, parseLineSegments(pl.body), marginRight)

		case lineKindImage:
			// In text mode, show the alt text (stored in pl.body contains the path;
			// we have no alt text here so show "[image]" as a placeholder).
			c.Write(x, y, vt.Dim, vt.DefaultBackground, "[image: "+pl.body+"]")

		default: // lineKindBody
			writeBookTextSegs(c, x, y, parseLineSegments(pl.body), marginRight)
		}
	}
}

// bookRawXToPixelX maps a raw rune index (same coordinate used by the cursor)
// within a rendered book-mode line to an absolute pixel X in the image.
func bookRawXToPixelX(fs *bookFontSet, pl parsedLine, rawLine string, rawX, marginLeft int) int {
	switch pl.kind {
	case lineKindBlank, lineKindImage:
		return marginLeft
	case lineKindHeader:
		prefixLen := pl.headerLevel + 1 // "# "=2, "## "=3, "### "=4
		adjRawX := max(rawX-prefixLen, 0)
		bodyRunes := []rune(pl.body)
		if adjRawX > len(bodyRunes) {
			adjRawX = len(bodyRunes)
		}
		hd := &font.Drawer{Face: fs.headerForLevel(pl.headerLevel)}
		return marginLeft + hd.MeasureString(string(bodyRunes[:adjRawX])).Round()
	case lineKindBullet, lineKindUnchecked, lineKindChecked, lineKindNumbered:
		rawPrefix := rawMarkdownPrefix(rawLine)
		rawBodyStart := len([]rune(rawPrefix))
		bodyRawX := rawX - rawBodyStart
		segs := parseLineSegments(pl.body)
		prefixW := (&font.Drawer{Face: fs.regular}).MeasureString(pl.prefix).Round()
		prefixX := marginLeft + prefixW
		if bodyRawX < 0 {
			return prefixX
		}
		bodyVisX := rawXToVisualX(pl.body, bodyRawX)
		return prefixX + measureSegmentsToRune(fs, segs, bodyVisX)
	default:
		segs := parseLineSegments(pl.body)
		visX := rawXToVisualX(rawLine, rawX)
		return marginLeft + measureSegmentsToRune(fs, segs, visX)
	}
}

// bookOverlaySelection renders the active text selection as a light-gray
// highlight. For each visible line that is fully or partially selected it:
//  1. Repaints the full cell white (fresh anti-aliasing slate).
//  2. Fills the selected pixel columns with a solid light-gray background.
//  3. Re-renders the line's text so glyph edges blend with the correct bg.
func (e *Editor) bookOverlaySelection(dst *image.RGBA, pixW, pixH, editRows int, cellH uint) {
	if !e.HasSelection() {
		return
	}

	fontSize := float64(cellH) * 0.72
	if fontSize < 6 {
		fontSize = 6
	}
	fs, err := bookFaces(fontSize)
	if err != nil {
		return
	}

	ascent := faceAscent(fs.regular, fontSize)
	lineH := int(cellH)
	if lineH <= 0 {
		lineH = int(fontSize) + 4
	}

	marginLeft := int(float64(pixW) * bookMarginLeft)
	marginRight := pixW - int(float64(pixW)*bookMarginRight)
	marginTop := int(float64(pixH) * bookMarginTop)
	marginBottom := int(float64(pixH) * bookMarginBottom)

	maxLines := min((pixH-marginTop-marginBottom)/lineH, editRows)

	startLine := e.pos.offsetY
	totalLines := e.Len()

	selStartY, selStartX := e.selection.start()
	selEndY, selEndX := e.selection.end()

	selBg := color.NRGBA{0xD0, 0xD0, 0xD0, 0xFF} // solid light gray
	dark := color.NRGBA{0x10, 0x10, 0x10, 0xff}

	// Iterate visible display rows, tracking document lines separately so that
	// multi-row image lines are handled correctly.
	docLine2 := startLine
	for row := 0; row < maxLines && docLine2 < totalLines; {
		lineIdx := LineIndex(docLine2)
		docLine2++

		rawLine := e.Line(lineIdx)
		rawLine = strings.ReplaceAll(rawLine, "\t", "    ")
		pl := parseBookLine(rawLine)
		cellTop := marginTop + row*lineH

		// Image lines consume multiple rows; skip selection highlighting on them.
		if pl.kind == lineKindImage {
			rowsConsumed := e.bookImageRows(pl.body, lineH, marginRight-marginLeft, pixH-marginTop-marginBottom)
			row += rowsConsumed
			continue
		}

		if lineIdx < selStartY || lineIdx > selEndY {
			row++
			continue
		}

		// Determine the pixel X range to highlight on this line.
		var hLeft, hRight int
		switch {
		case lineIdx == selStartY && lineIdx == selEndY:
			hLeft = bookRawXToPixelX(fs, pl, rawLine, selStartX, marginLeft)
			hRight = bookRawXToPixelX(fs, pl, rawLine, selEndX, marginLeft)
		case lineIdx == selStartY:
			hLeft = bookRawXToPixelX(fs, pl, rawLine, selStartX, marginLeft)
			hRight = marginRight
		case lineIdx == selEndY:
			hLeft = marginLeft
			hRight = bookRawXToPixelX(fs, pl, rawLine, selEndX, marginLeft)
		default: // fully selected middle line
			hLeft = marginLeft
			hRight = marginRight
		}
		if hLeft > hRight {
			hLeft, hRight = hRight, hLeft
		}
		if hLeft < 0 {
			hLeft = 0
		}
		if hRight > pixW {
			hRight = pixW
		}
		if hLeft >= hRight {
			row++
			continue
		}

		// 1. Repaint the full cell white so glyph anti-aliasing starts fresh.
		draw.Draw(dst, image.Rect(0, cellTop, pixW, cellTop+lineH),
			image.White, image.Point{}, draw.Src)

		// 2. Fill the selected pixel columns with the selection background.
		draw.Draw(dst, image.Rect(hLeft, cellTop, hRight, cellTop+lineH),
			image.NewUniform(selBg), image.Point{}, draw.Src)

		// 3. Re-render the line text so glyphs blend with the correct background.
		vPad := max((lineH-ascent)/2, 0)
		baseline := cellTop + vPad + ascent

		switch pl.kind {
		case lineKindBlank:
			// nothing to draw
		case lineKindRule:
			ruleY := cellTop + lineH/2
			for px := marginLeft; px < marginRight; px++ {
				dst.Set(px, ruleY, dark)
				dst.Set(px, ruleY+1, dark)
			}
		case lineKindHeader:
			hFace := fs.headerForLevel(pl.headerLevel)
			hAscent := faceAscent(hFace, fs.headerSizeForLevel(pl.headerLevel))
			hVPad := max((lineH-hAscent)/2, 0)
			d := &font.Drawer{
				Dst:  dst,
				Src:  image.NewUniform(dark),
				Face: hFace,
				Dot:  fixed.P(marginLeft, cellTop+hVPad+hAscent),
			}
			d.DrawString(pl.body)
		case lineKindUnchecked:
			bookDrawCheckbox(dst, marginLeft, cellTop, lineH, false)
			prefixX := max(marginLeft+lineH*55/100+3, marginLeft+5+3)
			drawSegments(dst, fs, prefixX, baseline, parseLineSegments(pl.body), dark)
		case lineKindChecked:
			bookDrawCheckbox(dst, marginLeft, cellTop, lineH, true)
			prefixX := max(marginLeft+lineH*55/100+3, marginLeft+5+3)
			bodyClr := color.NRGBA{0x55, 0x55, 0x55, 0xff}
			drawSegments(dst, fs, prefixX, baseline, parseLineSegments(pl.body), bodyClr)
		case lineKindBullet, lineKindNumbered:
			prefixX := drawString(dst, fs.regular, marginLeft, baseline, pl.prefix, dark)
			drawSegments(dst, fs, prefixX, baseline, parseLineSegments(pl.body), dark)
		default:
			drawSegments(dst, fs, marginLeft, baseline, parseLineSegments(pl.body), dark)
		}

		row++
	}
}

// bookPageToImage renders the visible page (content + selection + cursor) into
// an RGBA image. The content layer is cached and reused when dimensions and
// content have not changed; selection and cursor are always composited fresh.
func (e *Editor) bookPageToImage(pixW, pixH, editRows int, cellH uint) *image.RGBA {
	// Invalidate cache if dimensions, scroll offset, or document content changed.
	if bookContentCache == nil || bookContentCacheW != pixW || bookContentCacheH != pixH || bookContentCacheOffsetY != e.pos.offsetY || e.Changed() {
		bookContentCache = e.bookContentImage(pixW, pixH, editRows, cellH)
		bookContentCacheW = pixW
		bookContentCacheH = pixH
		bookContentCacheOffsetY = e.pos.offsetY
	}

	// Copy the content image so overlays don't pollute the cache.
	dst := image.NewRGBA(bookContentCache.Bounds())
	draw.Draw(dst, dst.Bounds(), bookContentCache, image.Point{}, draw.Src)

	e.bookOverlaySelection(dst, pixW, pixH, editRows, cellH)
	e.bookOverlayCursor(dst, pixW, pixH, editRows, cellH)
	return dst
}

// ── Scroll-to-cursor ──────────────────────────────────────────────────────────

// countDisplayRowsTo scans visible document lines starting at startDoc and
// returns the display-row index at which targetDoc appears, or -1 if it is
// not reached within maxRows (e.g. because multi-row images consume the budget).
func (e *Editor) countDisplayRowsTo(startDoc, targetDoc, maxRows, lineH, textW, textH int) int {
	dl := startDoc
	row := 0
	totalLines := e.Len()
	for row < maxRows && dl < totalLines {
		if dl == targetDoc {
			return row
		}
		rl := e.Line(LineIndex(dl))
		rl = strings.ReplaceAll(rl, "\t", "    ")
		pl := parseBookLine(rl)
		if pl.kind == lineKindImage {
			row += e.bookImageRows(pl.body, lineH, textW, textH)
		} else {
			row++
		}
		dl++
	}
	return -1
}

// bookModeEnsureCursorVisible adjusts e.pos.offsetY (and e.pos.sy) so that
// the cursor document line is always rendered inside the visible image area.
// It accounts for multi-row image lines that consume more than one display row.
func (e *Editor) bookModeEnsureCursorVisible(c *vt.Canvas) {
	cellW, cellH := imagepreview.TerminalCellPixels()
	if cellH == 0 {
		cellH = 16
	}
	if cellW == 0 {
		cellW = 8
	}

	rows := int(c.Height())
	editRows := rows - 1
	if editRows <= 0 {
		return
	}

	lineH := int(cellH)
	pixH := editRows * lineH
	pixW := int(uint(c.Width()) * cellW)

	marginLeft := int(float64(pixW) * bookMarginLeft)
	marginRight := pixW - int(float64(pixW)*bookMarginRight)
	marginTop := int(float64(pixH) * bookMarginTop)
	marginBottom := int(float64(pixH) * bookMarginBottom)
	textW := marginRight - marginLeft
	textH := pixH - marginTop - marginBottom

	maxDisplayRows := min((pixH-marginTop-marginBottom)/lineH, editRows)
	if maxDisplayRows <= 0 {
		return
	}

	cursorDataY := int(e.DataY())
	totalLines := e.Len()

	// Clamp to a valid document line (pos.sy can reach the status-bar row in
	// graphical mode before return.go catches it, giving a cursorDataY that
	// is one past the last line).
	if cursorDataY >= totalLines {
		cursorDataY = totalLines - 1
	}
	if cursorDataY < 0 || totalLines == 0 {
		return
	}

	// If cursor is above the scroll start, snap the view up to show it.
	if cursorDataY < e.pos.offsetY {
		e.pos.offsetY = cursorDataY
		e.pos.sy = 0
		return
	}

	// Check whether the cursor is already within the visible rows.
	if e.countDisplayRowsTo(e.pos.offsetY, cursorDataY, maxDisplayRows, lineH, textW, textH) >= 0 {
		return
	}

	// Cursor is below the visible area — advance offsetY one document line at
	// a time until the cursor fits inside maxDisplayRows.
	for e.pos.offsetY < cursorDataY && e.pos.offsetY < totalLines-1 {
		e.pos.offsetY++
		if e.countDisplayRowsTo(e.pos.offsetY, cursorDataY, maxDisplayRows, lineH, textW, textH) >= 0 {
			break
		}
	}

	// Keep sy consistent with the new offset.
	e.pos.sy = max(cursorDataY-e.pos.offsetY, 0)
}

// ── Terminal output ────────────────────────────────────────────────────────

func flushImageToTerminal(img image.Image, dispCols, dispRows uint) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return
	}
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	fmt.Fprintf(os.Stdout, "\033[H")
	imagepreview.FlushImage(os.Stdout, encoded, dispCols, dispRows)
}

func (e *Editor) bookModeRenderImage(c *vt.Canvas) {
	cols := uint(c.Width())
	rows := uint(c.Height())
	if rows < 2 {
		return
	}
	editRows := rows - 1

	cellW, cellH := imagepreview.TerminalCellPixels()
	if cellW == 0 {
		cellW = 8
	}
	if cellH == 0 {
		cellH = 16
	}

	pixW := int(cols * cellW)
	pixH := int(editRows * cellH)

	img := e.bookPageToImage(pixW, pixH, int(editRows), cellH)
	flushImageToTerminal(img, cols, editRows)
}

// bookModeStatusBar renders the bottom status row as a small image sent via
// the Kitty/iTerm2 graphics protocol, using the Montserrat Bold sans-serif face.
func (e *Editor) bookModeStatusBar(c *vt.Canvas) {
	cols := uint(c.Width())
	rows := uint(c.Height())

	cellW, cellH := imagepreview.TerminalCellPixels()
	if cellW == 0 {
		cellW = 8
	}
	if cellH == 0 {
		cellH = 16
	}

	pixW := int(cols * cellW)
	pixH := int(cellH) // one terminal row

	// Dark background, white text.
	img := image.NewRGBA(image.Rect(0, 0, pixW, pixH))
	bgClr := color.NRGBA{0x1a, 0x1a, 0x1a, 0xff}
	draw.Draw(img, img.Bounds(), image.NewUniform(bgClr), image.Point{}, draw.Src)

	// Reuse the font set already cached for the main font size.
	mainFontSize := float64(cellH) * 0.72
	if mainFontSize < 6 {
		mainFontSize = 6
	}
	if fs, err := bookFaces(mainFontSize); err == nil {
		if msg := bookModeGetStatusMsg(); msg != "" {
			// Temporary status message (e.g. "Saved foo.md") — always show over dark bar.
		} else if e.statusMode {
			// statusMode=true means "hide status bar": render the row white so it
			// blends with the book page and is visually absent.
			draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)
		}

		var text string
		if msg := bookModeGetStatusMsg(); msg != "" {
			text = "  " + strings.TrimSpace(msg) + "  "
		} else if !e.statusMode {
			// Default: position + word count.
			percentage, lineNumber, lastLineNumber := e.PLA()
			text = fmt.Sprintf("  line %d / %d  (%d%%)   col %d   words %d  ",
				lineNumber, lastLineNumber, percentage, e.ColNumber(), e.WordCount())
		}

		white := color.NRGBA{0xff, 0xff, 0xff, 0xff}
		sbAscent := faceAscent(fs.statusBar, fs.statusBarSize)
		baseline := (pixH+sbAscent)/2 + 1 // vertically centred
		d := &font.Drawer{Face: fs.statusBar}
		textPixW := d.MeasureString(text).Round()
		x := max(
			// centred horizontally
			(pixW-textPixW)/2, 4)
		drawString(img, fs.statusBar, x, baseline, text, white)
	}

	// Encode and send at the last terminal row.
	var buf bytes.Buffer
	if png.Encode(&buf, img) != nil {
		return
	}
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	fmt.Fprintf(os.Stdout, "\033[%d;1H", rows)
	imagepreview.FlushImage(os.Stdout, encoded, cols, 1)
}

// bookModeShowCursor positions the hidden terminal cursor at the current
// editing position. The visual I-beam is drawn directly in the image, so
// this only needs to keep the cursor at a sane position for OS-level
// accessibility and clipboard operations.
func (e *Editor) bookModeShowCursor(c *vt.Canvas) {
	cols := uint(c.Width())
	rows := uint(c.Height())
	editRows := rows - 1

	leftMarginCells := int(float64(cols) * bookMarginLeft)
	topMarginCells := int(float64(editRows) * bookMarginTop)

	rawLine := e.Line(e.DataY())
	cursorRawX := e.pos.sx + e.pos.offsetX

	pl := parseBookLine(rawLine)
	var x int
	if pl.kind == lineKindHeader {
		prefixLen := pl.headerLevel + 1
		adjX := max(cursorRawX-prefixLen, 0)
		x = adjX + leftMarginCells
	} else {
		x = rawXToVisualX(rawLine, cursorRawX) + leftMarginCells
	}

	y := int(e.pos.ScreenY()) + topMarginCells

	if x >= int(cols) {
		x = int(cols) - 1
	}
	if y >= int(editRows) {
		y = int(editRows) - 1
	}

	// Hide the terminal cursor — the image cursor is the visible one.
	fmt.Fprintf(os.Stdout, "\033[%d;%dH\033[?25l", y+1, x+1)
}

// bookModeResetCursor restores cursor visibility when leaving book mode.
func bookModeResetCursor() {
	fmt.Fprintf(os.Stdout, "\033[?25h\033[2 q")
}

func (e *Editor) bookToggleFormat(c *vt.Canvas, marker string) {
	if e.HasSelection() {
		sel := e.selection.Text(e)
		if strings.HasPrefix(sel, marker) && strings.HasSuffix(sel, marker) && len(sel) > 2*len(marker) {
			inner := sel[len(marker) : len(sel)-len(marker)]
			e.DeleteSelection(c, nil)
			e.InsertStringAndMove(c, inner)
		} else {
			e.DeleteSelection(c, nil)
			e.InsertStringAndMove(c, marker+sel+marker)
		}
		e.ClearSelection()
	} else {
		e.InsertStringAndMove(c, marker+marker)
		for range marker {
			e.CursorBackward(c, nil)
		}
	}
	e.changed.Store(true)
	e.redraw.Store(true)
}
