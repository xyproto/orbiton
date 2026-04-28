package main

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"github.com/xyproto/env/v2"
	"github.com/xyproto/imagepreview"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed fonts/Vollkorn-Regular.ttf.gz
var vollkornRegularTTFGz []byte

//go:embed fonts/Vollkorn-Italic.ttf.gz
var vollkornItalicTTFGz []byte

//go:embed fonts/Montserrat-Bold.ttf.gz
var montserratBoldTTFGz []byte

//go:embed fonts/FiraMono-Bold.ttf.gz
var firaMonoBoldTTFGz []byte

//go:embed fonts/Montserrat-Light.ttf.gz
var montserratLightTTFGz []byte

// DejaVu Sans is embedded as a broad-coverage fallback font: Greek, Cyrillic,
// extended Latin, math symbols, arrows and other glyphs the stylistic primary
// fonts lack.
//
// The embedded copy is a subset of the upstream DejaVuSans.ttf, containing
// only the Unicode ranges we need as fallback glyphs for book mode and SVG
// text rendering. This keeps the compiled binary small. To regenerate:
//
//	pyftsubset DejaVuSans.ttf \
//	  --unicodes="U+0020-007E,U+00A0-00FF,U+0100-024F,U+0370-03FF,U+0400-04FF,\
//	U+0500-052F,U+2000-206F,U+2070-209F,U+20A0-20CF,U+2100-214F,U+2150-218F,\
//	U+2190-21FF,U+2200-22FF,U+2300-23FF,U+2460-24FF,U+2500-257F,U+2580-259F,\
//	U+25A0-25FF,U+2600-26FF,U+2700-27BF,U+FB00-FB06,U+FFFD" \
//	  --output-file=DejaVuSans.ttf && gzip -9 DejaVuSans.ttf
//
//go:embed fonts/DejaVuSans.ttf.gz
var dejaVuSansTTFGz []byte

// gunzipBytes decompresses the given gzip-compressed byte slice. Used to
// decompress the embedded .ttf.gz font files on demand.
func gunzipBytes(gz []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// Margin ratios for book mode (fraction of pixel/column dimensions).
const (
	bookMarginLeft      = 0.10
	bookMarginRight     = 0.05
	bookMarginTop       = 0.02
	bookMarginBottom    = 0.02
	bookTextMarginRight = 0.02 // tighter right margin for text book mode
)

// bookLineHeightMul scales the per-line vertical pitch in graphical book mode
// relative to the terminal cell height; >1.0 adds breathing room between lines.
const bookLineHeightMul = 1.25

// Palette for text book mode. Using true-colour foregrounds avoids the
// "black looks gray" problem on terminals whose 16-colour palette maps
// color 0 to a dark gray (common with many xterm colour schemes). The vt
// library degrades true-colour values to xterm-256 or ANSI-16 automatically
// on terminals that don't advertise 24-bit colour support.
var (
	bookTextFGBlack    = vt.TrueColor(0, 0, 0)
	bookTextFGDarkGray = vt.TrueColor(80, 80, 80)
	bookTextFGWhite    = vt.TrueColor(255, 255, 255)
	bookTextBG         = vt.TrueBackground(255, 255, 255)
	// Dark-mode variants used when e.bookDarkMode is true
	bookTextFGLight     = vt.TrueColor(220, 220, 220)
	bookTextFGLightGray = vt.TrueColor(170, 170, 170)
	bookTextBGDark      = vt.TrueBackground(26, 26, 26)
)

// bookFG returns the appropriate foreground color for book mode rendering
// based on dark/light mode preference.
func (e *Editor) bookFG() color.NRGBA {
	if e.bookDarkMode {
		return color.NRGBA{0xe8, 0xe8, 0xe8, 0xff}
	}
	return color.NRGBA{0x10, 0x10, 0x10, 0xff}
}

// bookBG returns the appropriate background color for book mode rendering.
func (e *Editor) bookBG() color.NRGBA {
	if e.bookDarkMode {
		return color.NRGBA{0x1a, 0x1a, 0x1a, 0xff}
	}
	return color.NRGBA{0xff, 0xff, 0xff, 0xff}
}

// bookBGImage returns the background as a uniform image, suitable for draw.Draw.
func (e *Editor) bookBGImage() image.Image {
	if e.bookDarkMode {
		return image.NewUniform(e.bookBG())
	}
	return image.White
}

// bookCodeBG returns the code-block background color.
func (e *Editor) bookCodeBG() color.NRGBA {
	if e.bookDarkMode {
		return color.NRGBA{0x2a, 0x2a, 0x2a, 0xff}
	}
	return color.NRGBA{0xf0, 0xf0, 0xf0, 0xff}
}

// bookDimFG returns a dimmed foreground color (for checked items, secondary text).
func (e *Editor) bookDimFG() color.NRGBA {
	if e.bookDarkMode {
		return color.NRGBA{0x90, 0x90, 0x90, 0xff}
	}
	return color.NRGBA{0x55, 0x55, 0x55, 0xff}
}

// bookTextModeBG returns the text-mode background attribute color.
func (e *Editor) bookTextModeBG() vt.AttributeColor {
	if e.bookDarkMode {
		return bookTextBGDark
	}
	return bookTextBG
}

// bookGraphicalStatusBarBG returns the page-pixel colour that the
// graphical-mode status bar paints its background with. Kept in sync with
// bookStatusBarImage: light mode uses black, dark mode uses white. Used by
// the rounded-corner carver so the bottom corners blend into the bar.
func (e *Editor) bookGraphicalStatusBarBG() color.NRGBA {
	if e.bookDarkMode {
		return color.NRGBA{0xff, 0xff, 0xff, 0xff}
	}
	return color.NRGBA{0x00, 0x00, 0x00, 0xff}
}

// bookBottomCornerBG returns the colour that the bottom rounded corners of
// the page should fade into. That's whatever is drawn directly below the
// page in the composed frame:
//
//   - status bar visible → the status bar background
//   - status bar hidden  → terminal chrome (black)
//
// Picking the wrong value leaves a thin seam between the page's rounded
// corners and the strip below.
func (e *Editor) bookBottomCornerBG() color.NRGBA {
	if env.Str("TERM") == "xterm-kitty" {
		return color.NRGBA{0x00, 0x00, 0x00, 0x00}
	}
	if !e.statusMode || bookModeGetStatusMsg() != "" {
		return e.bookGraphicalStatusBarBG()
	}
	return color.NRGBA{0x00, 0x00, 0x00, 0xff}
}

// bookTextModeFGBlack returns the primary text color for text-mode book rendering.
func (e *Editor) bookTextModeFGBlack() vt.AttributeColor {
	if e.bookDarkMode {
		return bookTextFGLight
	}
	return bookTextFGBlack
}

// bookTextModeFGDim returns the dimmed text color for text-mode book rendering.
func (e *Editor) bookTextModeFGDim() vt.AttributeColor {
	if e.bookDarkMode {
		return bookTextFGLightGray
	}
	return bookTextFGDarkGray
}

// bookFontSet caches Vollkorn body faces, Montserrat Bold header faces, a small
// Montserrat Bold face for the status bar, and a FiraMono Bold face for code,
// all derived from the same base pixel size.
type bookFontSet struct {
	regular       font.Face
	italic        font.Face
	code          font.Face // FiraMono Bold — used for inline code, fenced blocks, indented code
	h1            font.Face
	h2            font.Face
	h3            font.Face
	h4            font.Face
	h5            font.Face
	h1Code        font.Face // FiraMono Bold sized to h1 — for inline `code` in H1 headers
	h2Code        font.Face
	h3Code        font.Face
	h4Code        font.Face
	h5Code        font.Face
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

// headerCodeForLevel returns a code (FiraMono Bold) face sized to match the
// given header level so inline `code` segments inside a header render at the
// same visual size as the surrounding header text. Falls back to fs.code if
// the level-specific code face could not be created.
func (fs *bookFontSet) headerCodeForLevel(level int) font.Face {
	var f font.Face
	switch level {
	case 1:
		f = fs.h1Code
	case 2:
		f = fs.h2Code
	case 3:
		f = fs.h3Code
	case 4:
		f = fs.h4Code
	default:
		f = fs.h5Code
	}
	if f == nil {
		return fs.code
	}
	return f
}

var (
	bookFontMu              sync.Mutex
	bookFontCache           *bookFontSet
	parsedVollkornRegular   *opentype.Font
	parsedVollkornItalic    *opentype.Font
	parsedMontserratBold    *opentype.Font
	parsedMontserratLight   *opentype.Font
	parsedFiraMonoBold      *opentype.Font
	parsedDejaVuSans        *opentype.Font // Unicode-rich fallback
	parseFontsOnce          sync.Once
	parseFontsErr           error
	bookContentCache        *image.RGBA
	bookContentCacheW       int
	bookContentCacheH       int
	bookContentCacheOffsetY int
	bookContentCacheGen     uint64 // generation counter when content cache was built
	bookContentGen          uint64 // bumped each time document content changes
	bookContentGenMu        sync.Mutex
	bookPageEncoded         string // cached base64-encoded PNG of the last page image
	bookPageEncodedCols     uint   // terminal cols when bookPageEncoded was produced
	bookPageEncodedRows     uint   // editRows when bookPageEncoded was produced
	bookStatusMsg           string
	bookStatusMsgMu         sync.Mutex
	// bookStatusClearGen coalesces status auto-clear goroutines: each
	// status message bumps the counter and only the freshest goroutine
	// actually clears + re-renders after the timeout.
	bookStatusClearGen atomic.Uint64

	// bookPNGEncoder and bookPNGBuf are reused across frames to avoid
	// per-frame allocations. BestSpeed keeps encoding time low.
	bookPNGEncoder = png.Encoder{CompressionLevel: png.BestSpeed}
	bookPNGBuf     bytes.Buffer
	// bookPageImageBuf is the reusable RGBA destination for bookPageToImage.
	bookPageImageBuf *image.RGBA
	// bookComposeBuf is the reusable RGBA destination for bookComposeFullPage.
	bookComposeBuf *image.RGBA
	// bookWriteBuf batches a frame's escape sequences and image payload into
	// one Write so the terminal sees it atomically.
	bookWriteBuf bytes.Buffer
)

func parsedFonts() error {
	parseFontsOnce.Do(func() {
		// Decompress the embedded .ttf.gz blobs once, parse, and drop the
		// decompressed TTF bytes so we only keep the gzip'd originals plus
		// the parsed font.SFNT tables in memory.
		parseOne := func(gz []byte) (*opentype.Font, error) {
			ttf, err := gunzipBytes(gz)
			if err != nil {
				return nil, err
			}
			return opentype.Parse(ttf)
		}
		var err error
		if parsedVollkornRegular, err = parseOne(vollkornRegularTTFGz); err != nil {
			parseFontsErr = err
			return
		}
		if parsedVollkornItalic, err = parseOne(vollkornItalicTTFGz); err != nil {
			parseFontsErr = err
			return
		}
		if parsedMontserratBold, err = parseOne(montserratBoldTTFGz); err != nil {
			parseFontsErr = err
			return
		}
		if parsedMontserratLight, err = parseOne(montserratLightTTFGz); err != nil {
			parseFontsErr = err
			return
		}
		if parsedFiraMonoBold, err = parseOne(firaMonoBoldTTFGz); err != nil {
			parseFontsErr = err
			return
		}
		if parsedDejaVuSans, err = parseOne(dejaVuSansTTFGz); err != nil {
			// The fallback is optional — if it fails to parse, fewer
			// glyphs render but book mode still works.
			parsedDejaVuSans = nil
		}
	})
	return parseFontsErr
}

// bookGraphicsCapable reports whether the current terminal can display inline
// pixel images via Kitty, iTerm2 or Sixel. Unlike imagepreview.HasGraphics it
// does NOT gate on NO_COLOR: NO_COLOR is a contract about ANSI colour escape
// sequences; Kitty/iTerm2/Sixel transport pixels out-of-band and do not emit
// ANSI colour, so honouring NO_COLOR there would spuriously disable the
// graphical book mode on perfectly capable terminals (the original bug that
// made NBOOKG/DNBOOKG/NBOOKSX/DNBOOKSX unreachable).
func bookGraphicsCapable() bool {
	return (imagepreview.IsKitty || imagepreview.IsITerm2 || imagepreview.IsSixel) && !imagepreview.IsVT
}

// bookGraphicalMode returns true when book mode uses the Kitty/iTerm2/Sixel
// graphics protocol for rendering (terminal supports pixel-level image display).
func (e *Editor) bookGraphicalMode() bool {
	return e.bookMode.Load() && bookGraphicsCapable() && !e.bookForceTextMode.Load()
}

// bookTextMode returns true when book mode uses VT100/xterm text rendering
// (vt100, vt220, xterm, xterm-color, xterm-256color, linux terminals).
func (e *Editor) bookTextMode() bool {
	return e.bookMode.Load() && (!bookGraphicsCapable() || e.bookForceTextMode.Load())
}

// enterBookModeText enables book mode with text rendering and saves the editor
// settings that book mode overrides, so exitBookMode can restore them. The
// dark/light auto-detect runs at most once per editor session.
func (e *Editor) enterBookModeText() {
	if !e.bookSaved {
		e.bookSavedSyntaxHighlight = e.syntaxHighlight
		e.bookSavedStatusMode = e.statusMode
		e.bookSavedWrapWhenTyping = e.wrapWhenTyping
		e.bookSavedWrapWidth = e.wrapWidth
		e.bookSaved = true
	}
	if !e.bookDarkModeInitialized {
		if initialLightBackground != nil && !*initialLightBackground {
			e.bookDarkMode = true
		}
		e.bookDarkModeInitialized = true
	}
	e.bookMode.Store(true)
	e.bookForceTextMode.Store(true)
	e.syntaxHighlight = false
	e.wrapWidth = 72
	e.wrapWhenTyping = false
	e.statusMode = true
	e.bookSavedLocalX = -1
}

// exitBookMode disables book mode and restores the saved editor settings.
func (e *Editor) exitBookMode() {
	if imagepreview.IsKitty {
		imagepreview.DeleteInlineImages()
	}
	e.bookMode.Store(false)
	e.bookForceTextMode.Store(false)
	if e.bookSaved {
		e.syntaxHighlight = e.bookSavedSyntaxHighlight
		e.statusMode = e.bookSavedStatusMode
		e.wrapWhenTyping = e.bookSavedWrapWhenTyping
		e.wrapWidth = e.bookSavedWrapWidth
		e.bookSaved = false
	}
	bookModeResetCursor()
}

// bookTextTopBarRows is the number of terminal rows reserved at the top of
// text book mode for a filename status bar. Graphical book mode has no such
// top bar (it's always 0 there).
const bookTextTopBarRows = 1

// bookEditRows returns the number of terminal rows available for the graphical
// book content. When the status bar is hidden (statusMode==true), the full
// terminal height is used; otherwise one row is reserved for the status bar.
func (e *Editor) bookEditRows(totalRows uint) uint {
	reserved := uint(0)
	if !e.statusMode {
		reserved++ // bottom status bar
	}
	if e.bookTextMode() {
		reserved += uint(bookTextTopBarRows) // top filename bar (text mode only)
	}
	if totalRows <= reserved {
		return 0
	}
	return totalRows - reserved
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

// bookBumpContentGen increments the content generation counter so the content
// cache is invalidated on the next render. This is cheaper than using
// e.Changed() which stays true until the file is saved.
func bookBumpContentGen() {
	bookContentGenMu.Lock()
	bookContentGen++
	bookContentGenMu.Unlock()
}

// bookCurrentContentGen returns the current content generation counter.
func bookCurrentContentGen() uint64 {
	bookContentGenMu.Lock()
	defer bookContentGenMu.Unlock()
	return bookContentGen
}

// Cursor affinity values used at wrap boundaries. When the cursor's raw
// rune position is exactly at a soft-wrap boundary, it maps to two distinct
// visual positions: the end of the earlier sub-row and the start of the
// next sub-row. The editor uses bookCursorAffinity to remember which of the
// two the cursor belongs to. Without this disambiguation, pressing End on
// a soft-wrapped line would place the cursor visually at the start of the
// next sub-row, and pressing Up from there would no-op.
const (
	bookAffinityForward  = 0 // cursor belongs to the start of the next sub-row
	bookAffinityBackward = 1 // cursor belongs to the end of the previous sub-row
)

// bookFallbackFaces maps a primary font.Face to a same-size DejaVu Sans face
// used when the primary lacks a glyph (e.g. Greek, Cyrillic, math).
var bookFallbackFaces sync.Map // font.Face -> font.Face

func newFace(f *opentype.Font, size float64) (font.Face, error) {
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size: size,
		// The page image is rendered at the terminal's real cell size, so we
		// always use 96 DPI here; environment-based probes proved unreliable.
		DPI:     96,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}
	// Register a matching-size DejaVu Sans fallback for this face
	if parsedDejaVuSans != nil && f != parsedDejaVuSans {
		if fb, e2 := opentype.NewFace(parsedDejaVuSans, &opentype.FaceOptions{
			Size:    size,
			DPI:     96,
			Hinting: font.HintingFull,
		}); e2 == nil {
			bookFallbackFaces.Store(face, fb)
		}
	}
	return face, nil
}

// faceFallback returns the registered DejaVu fallback for primary, or nil
func faceFallback(primary font.Face) font.Face {
	if v, ok := bookFallbackFaces.Load(primary); ok {
		return v.(font.Face)
	}
	return nil
}

// faceGlyphAdvance is like face.GlyphAdvance but falls back to DejaVu Sans
// for runes the primary face does not cover
func faceGlyphAdvance(face font.Face, r rune) (fixed.Int26_6, bool) {
	if adv, ok := face.GlyphAdvance(r); ok {
		return adv, true
	}
	if fb := faceFallback(face); fb != nil {
		if adv, ok := fb.GlyphAdvance(r); ok {
			return adv, true
		}
	}
	return face.GlyphAdvance(r)
}

// measureStringFB measures the pixel width of s, falling back to DejaVu Sans
// for runes the primary face lacks
func measureStringFB(face font.Face, s string) fixed.Int26_6 {
	var total fixed.Int26_6
	for _, r := range s {
		adv, _ := faceGlyphAdvance(face, r)
		total += adv
	}
	return total
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
	h1Size := pixelSize * 1.55
	h2Size := pixelSize * 1.35
	h3Size := pixelSize * 1.2
	h4Size := pixelSize * 1.1
	h5Size := pixelSize * 1.0
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
	sb, err := newFace(parsedMontserratLight, statusBarSize)
	if err != nil {
		return nil, err
	}
	// Code font is slightly smaller than the body to visually distinguish it.
	codeSize := pixelSize * 0.88
	if codeSize < 6 {
		codeSize = 6
	}
	cod, err := newFace(parsedFiraMonoBold, codeSize)
	if err != nil {
		return nil, err
	}
	// Code faces sized to match each header level. Inline `code` segments
	// inside a header should render at the header's font size (in the
	// FiraMono Bold face) instead of being shrunk to the body-code size.
	// Individual failures are non-fatal — headerCodeForLevel falls back to
	// fs.code in that case.
	h1Code, _ := newFace(parsedFiraMonoBold, h1Size)
	h2Code, _ := newFace(parsedFiraMonoBold, h2Size)
	h3Code, _ := newFace(parsedFiraMonoBold, h3Size)
	h4Code, _ := newFace(parsedFiraMonoBold, h4Size)
	h5Code, _ := newFace(parsedFiraMonoBold, h5Size)
	bookFontCache = &bookFontSet{
		regular:       reg,
		italic:        ita,
		code:          cod,
		h1:            h1,
		h2:            h2,
		h3:            h3,
		h4:            h4,
		h5:            h5,
		h1Code:        h1Code,
		h2Code:        h2Code,
		h3Code:        h3Code,
		h4Code:        h4Code,
		h5Code:        h5Code,
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

// textSegment is one styled run of text within a body line.
type textSegment struct {
	text      string
	bold      bool
	italic    bool
	underline bool
	strike    bool   // rendered with a strikethrough line
	code      bool   // rendered in the monospace code font (FiraMono Bold)
	linkURL   string // non-empty for inline links: the hyperlink target
}

// parseLineSegments converts a line with Markdown-like inline markers into
// styled segments. Markers consumed (not rendered):
//
//	***text***              bold + italic
//	**text**                bold
//	*text*                  italic
//	__text__                underline
//	~~text~~                strikethrough
//	`code`                  code style (backticks stripped)
//	[text](url)             rendered as literal "[text](url)" in code style
//	![alt](url)             rendered as literal "![alt](url)" in code style
//	[![alt](img)](url)      rendered as a single literal code segment so that
//	                        the trailing "](url)" is also styled as code
func parseLineSegments(line string) []textSegment {
	flush := func(segs []textSegment, cur *strings.Builder, bold, italic, underline, strike bool) []textSegment {
		if cur.Len() > 0 {
			// Decode named/numeric HTML entities ("&lt;" → "<", "&amp;" → "&",
			// "&#124;" → "|", etc.) so markdown sources authored as HTML
			// render with the intended glyphs in book mode. Inline `code`
			// segments are flushed separately and keep their literal text —
			// code blocks should preserve entities as-is.
			segs = append(segs, textSegment{text: html.UnescapeString(cur.String()), bold: bold, italic: italic, underline: underline, strike: strike})
			cur.Reset()
		}
		return segs
	}
	type state struct{ bold, italic, underline, strike bool }
	var (
		segs  []textSegment
		cur   strings.Builder
		st    state
		runes = []rune(line)
	)
	// scanParenClose returns the index of the matching ')' for a '(' at `open`,
	// or -1 if not found. Nested parentheses are respected.
	scanParenClose := func(start int) int {
		depth := 1
		for k := start; k < len(runes); k++ {
			switch runes[k] {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					return k
				}
			}
		}
		return -1
	}
	for i := 0; i < len(runes); {
		// [![alt](img)](url) — link-wrapped image. Emit a single link
		// segment so the whole thing is clickable and coloured. When the
		// alt text is empty we fall back to a short placeholder rather
		// than an invisible segment.
		if runes[i] == '[' && i+1 < len(runes) && runes[i+1] == '!' &&
			i+2 < len(runes) && runes[i+2] == '[' {
			j := i + 3
			for j < len(runes) && runes[j] != ']' {
				j++
			}
			if j < len(runes) && j+1 < len(runes) && runes[j+1] == '(' {
				imgClose := scanParenClose(j + 2)
				if imgClose > 0 && imgClose+2 < len(runes) &&
					runes[imgClose+1] == ']' && runes[imgClose+2] == '(' {
					urlClose := scanParenClose(imgClose + 3)
					if urlClose > 0 {
						segs = flush(segs, &cur, st.bold, st.italic, st.underline, st.strike)
						alt := strings.TrimSpace(string(runes[i+3 : j]))
						linkURL := strings.TrimSpace(string(runes[imgClose+3 : urlClose]))
						display := alt
						if display == "" {
							display = "image"
						}
						segs = append(segs, textSegment{
							text:      html.UnescapeString(display),
							bold:      st.bold,
							italic:    st.italic,
							underline: st.underline,
							strike:    st.strike,
							linkURL:   linkURL,
						})
						i = urlClose + 1
						continue
					}
				}
			}
			// fall through to standalone `[` handling below
		}
		// ![alt](url) — render the literal source in code style.
		if runes[i] == '!' && i+1 < len(runes) && runes[i+1] == '[' {
			j := i + 2
			for j < len(runes) && runes[j] != ']' {
				j++
			}
			if j < len(runes) && j+1 < len(runes) && runes[j+1] == '(' {
				k := scanParenClose(j + 2)
				if k > 0 {
					segs = flush(segs, &cur, st.bold, st.italic, st.underline, st.strike)
					literal := string(runes[i : k+1])
					segs = append(segs, textSegment{text: literal, code: true, underline: st.underline, strike: st.strike})
					i = k + 1
					continue
				}
			}
			cur.WriteRune(runes[i])
			i++
			continue
		}
		// [text](url) — render as a styled link (display text only).
		// The text scan tracks bracket nesting so "[foo[bar]](url)" is
		// recognised correctly.
		if runes[i] == '[' {
			j := i + 1
			depth := 1
			for j < len(runes) && depth > 0 {
				switch runes[j] {
				case '[':
					depth++
				case ']':
					depth--
				}
				if depth == 0 {
					break
				}
				j++
			}
			if j < len(runes) && j+1 < len(runes) && runes[j+1] == '(' {
				k := scanParenClose(j + 2)
				if k > 0 {
					segs = flush(segs, &cur, st.bold, st.italic, st.underline, st.strike)
					linkText := string(runes[i+1 : j])
					linkURL := strings.TrimSpace(string(runes[j+2 : k]))
					segs = append(segs, textSegment{
						text:      html.UnescapeString(linkText),
						bold:      st.bold,
						italic:    st.italic,
						underline: st.underline,
						strike:    st.strike,
						linkURL:   linkURL,
					})
					i = k + 1
					continue
				}
			}
			cur.WriteRune(runes[i])
			i++
			continue
		}
		// inline code: render in the monospace code font with backticks stripped
		if runes[i] == '`' {
			j := i + 1
			for j < len(runes) && runes[j] != '`' {
				j++
			}
			if j < len(runes) {
				segs = flush(segs, &cur, st.bold, st.italic, st.underline, st.strike)
				codeText := string(runes[i+1 : j])
				segs = append(segs, textSegment{text: codeText, code: true, underline: st.underline, strike: st.strike})
				i = j + 1
				continue
			}
			cur.WriteRune(runes[i])
			i++
			continue
		}
		if i+2 < len(runes) && runes[i] == '*' && runes[i+1] == '*' && runes[i+2] == '*' {
			segs = flush(segs, &cur, st.bold, st.italic, st.underline, st.strike)
			st.bold = !st.bold
			st.italic = !st.italic
			i += 3
			continue
		}
		if i+1 < len(runes) && runes[i] == '*' && runes[i+1] == '*' {
			segs = flush(segs, &cur, st.bold, st.italic, st.underline, st.strike)
			st.bold = !st.bold
			i += 2
			continue
		}
		if runes[i] == '*' {
			segs = flush(segs, &cur, st.bold, st.italic, st.underline, st.strike)
			st.italic = !st.italic
			i++
			continue
		}
		if i+1 < len(runes) && runes[i] == '_' && runes[i+1] == '_' {
			segs = flush(segs, &cur, st.bold, st.italic, st.underline, st.strike)
			st.underline = !st.underline
			i += 2
			continue
		}
		if i+1 < len(runes) && runes[i] == '~' && runes[i+1] == '~' {
			segs = flush(segs, &cur, st.bold, st.italic, st.underline, st.strike)
			st.strike = !st.strike
			i += 2
			continue
		}
		cur.WriteRune(runes[i])
		i++
	}
	segs = flush(segs, &cur, st.bold, st.italic, st.underline, st.strike)
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
	lineKindImage // standalone ![alt](path) image (local file only)
	lineKindCode  // fenced (``` / ~~~) or 4-space-indented code block body line
	lineKindTable // Markdown table row (header, data, or separator)
)

type parsedLine struct {
	kind        lineKind
	headerLevel int    // 1, 2, 3, 4, 5 for lineKindHeader
	indent      int    // leading spaces & 2 (list nesting depth)
	prefix      string // rendered prefix, e.g. "• ", "1. ", "☐ ", "☑ "
	body        string // text after the prefix, may contain inline markers
}

// isFencedCodeMarker reports whether the raw line is a fenced code block
// delimiter (``` or ~~~ with optional language tag / trailing spaces).
func isFencedCodeMarker(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

// fenceStateAtLine returns true if document line lineIdx is inside a fenced
// code block, by counting opening/closing fence markers from line 0.
func (e *Editor) fenceStateAtLine(lineIdx int) bool {
	inFence := false
	for i := range lineIdx {
		raw := e.Line(LineIndex(i))
		raw = strings.ReplaceAll(raw, "\t", "    ")
		if isFencedCodeMarker(raw) {
			inFence = !inFence
		}
	}
	return inFence
}

// inHTMLCommentAtLine reports whether the given line is inside an HTML
// comment block (<!-- ... -->). Tracks <!-- and --> markers across the
// document from the start to the given lineIdx.
func (e *Editor) inHTMLCommentAtLine(lineIdx int) bool {
	const open = "<!--"
	const close = "-->"
	inComment := false
	for i := range lineIdx {
		raw := e.Line(LineIndex(i))
		for j := 0; j < len(raw); {
			if strings.HasPrefix(raw[j:], open) {
				inComment = true
				j += len(open)
			} else if strings.HasPrefix(raw[j:], close) {
				inComment = false
				j += len(close)
			} else {
				j++
			}
		}
	}
	return inComment
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

// bookDocumentTitle returns a human-friendly title for the document: the
// first top-level (H1) or second-level (H2) Markdown heading found, trimmed
// to a reasonable length. Falls back to the base filename when no heading is
// present. Used by the text-mode top bar.
func (e *Editor) bookDocumentTitle() string {
	const maxLen = 80
	n := e.Len()
	scanLimit := min(n, 200) // don't scan huge files end-to-end for a title
	ellipsis := "…"
	if useASCII {
		ellipsis = "..."
	}
	truncate := func(t string) string {
		if len(t) > maxLen {
			t = t[:maxLen-len(ellipsis)] + ellipsis
		}
		return t
	}
	for i := range scanLimit {
		raw := strings.TrimRight(e.Line(LineIndex(i)), " \t")
		if strings.HasPrefix(raw, "# ") {
			return truncate(strings.TrimSpace(raw[2:]))
		}
		if strings.HasPrefix(raw, "## ") {
			return truncate(strings.TrimSpace(raw[3:]))
		}
	}
	base := filepath.Base(e.filename)
	if base == "" || base == "." {
		return "untitled"
	}
	// Strip common doc extensions for a cleaner look.
	for _, ext := range []string{".md", ".markdown", ".txt", ".rst", ".adoc"} {
		if strings.HasSuffix(strings.ToLower(base), ext) {
			return base[:len(base)-len(ext)]
		}
	}
	return base
}

// bookBarPalette returns (fg, dimFg, bg) for the text book-mode bars. The
// top bar and the bottom status bar must use the same background/foreground
// pair so they frame the reading area as a matched set.
//
// Palette rationale — evokes WordGrinder / AbiWord / classic word-processor
// aesthetics rather than the WordPerfect-5.1 blue. Warm parchment tones in
// light mode, sepia ink in dark mode: think paper and a fountain pen.
//
// Tiers (per the xyproto/vt capability model):
//
//   - NO_COLOR environment variable set → no colour escapes (always defaults)
//   - TERM=vt100, TERM=vt*, TERM=xterm → one of the 16 ANSI colours (or defaults if NO_COLOR)
//   - TERM=xterm-256color → a 256-colour palette index (or ANSI if NO_COLOR)
//   - TERM=xterm-kitty → a true (24-bit) RGB colour (or ANSI if NO_COLOR)
func bookBarPalette(dark bool) (fg, dimFg, bg vt.AttributeColor) {
	if envNoColor {
		return vt.Default, vt.Default, vt.DefaultBackground
	}
	term := env.Str("TERM")
	switch term {
	case "xterm-kitty":
		// 24-bit true colour. Neutral gray — consistent with the ANSI-16
		// palette used by vt220/linux/screen so the bar looks the same
		// regardless of $TERM.
		if dark {
			return vt.TrueColor(238, 238, 238), vt.TrueColor(175, 175, 175), vt.TrueBackground(48, 48, 48)
		}
		return vt.TrueColor(0, 0, 0), vt.TrueColor(110, 110, 110), vt.TrueBackground(192, 192, 192)
	case "xterm-256color":
		// 256-colour palette. Neutral gray — matches the ANSI-16
		// default below at higher fidelity.
		if dark {
			return vt.Color256(255), vt.Color256(250), vt.Background256(239)
		}
		return vt.Color256(16), vt.Color256(242), vt.Background256(250)
	}
	// All other TERMs (xterm, vt220, linux, screen, tmux, …): 16 ANSI
	// colours. Use LightGray/Black (neutral, AbiWord-like) for light
	// mode, and DarkGray/White for dark mode. No blue — distinct from
	// the WordPerfect 5.1 look the user asked to avoid.
	if dark {
		return vt.White, vt.LightGray, vt.BackgroundBrightBlack
	}
	return vt.Black, vt.DarkGray, vt.BackgroundLightGray
}

// bookReadingPercent converts the current line position into a reading
// progress percentage that is 0 at the very top and 100 at the very bottom.
// Using line-number / lastLine (as the generic editor does) makes line 1 of
// a 6-line document show 16%, which is confusing for a "how far have I read"
// indicator in book mode — the reader has not progressed at all yet.
func bookReadingPercent(lineNumber, lastLineNumber LineNumber) int {
	if lastLineNumber <= 1 {
		return 0
	}
	p := int(100.0 * float64(lineNumber-1) / float64(lastLineNumber-1))
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

// bookCurrentHeading walks backwards from the given document line to find
// the nearest preceding ATX heading and returns its body (without the
// leading hashes). Returns the empty string when no heading is found.
// Used by the text book-mode bottom bar so writers always see which
// section they are currently in. The walk is bounded so that a long file
// without headings does not slow down the per-keystroke redraw.
func (e *Editor) bookCurrentHeading(line LineIndex) string {
	const lookback = 5000
	from := int(line)
	to := max(from-lookback, 0)
	for i := from; i >= to; i-- {
		raw := strings.TrimRight(e.Line(LineIndex(i)), " \t")
		// Match parseBookLine: only flush-left "#"-prefixed lines count.
		if !strings.HasPrefix(raw, "#") {
			continue
		}
		for _, prefix := range []string{"# ", "## ", "### ", "#### ", "##### ", "###### "} {
			if strings.HasPrefix(raw, prefix) {
				return strings.TrimSpace(raw[len(prefix):])
			}
		}
	}
	return ""
}

// bookBarSlots holds the three text slots that make up a text book-mode
// bar row. Empty strings are treated as "no slot" so the surrounding
// slots can use the freed space.
type bookBarSlots struct {
	left, center, right string
}

// drawBookBar paints a text book-mode bar row at y. Side slots are
// dim-foreground and capped at roughly a third of the bar width each so
// neither can crowd out the others; the centre slot uses the primary
// foreground and is truncated (with an ellipsis) when it would overlap
// a side slot. Empty slots are simply omitted. This is the shared
// painter for drawBookTopBar and the text-book branch of StatusBar.Draw,
// so changing the layout of either bar is a matter of constructing the
// right bookBarSlots value.
//
// Degrades gracefully:
//   - NO_COLOR / TERM=vt100 → no colours; bar text in the terminal default.
//   - TERM=xterm*           → colour tier chosen by bookBarPalette.
func (e *Editor) drawBookBar(c *vt.Canvas, y uint, w int, slots bookBarSlots) {
	if w <= 0 {
		return
	}
	fg, dimFg, bg := bookBarPalette(e.bookDarkMode)
	c.Write(0, y, fg, bg, strings.Repeat(" ", w))

	const pad = 2
	ellipsis := "…"
	if useASCII {
		ellipsis = "..."
	}
	truncate := func(s string, maxLen int) string {
		if maxLen <= 0 || len(s) > maxLen && maxLen <= len(ellipsis) {
			return ""
		}
		if len(s) <= maxLen {
			return s
		}
		return s[:maxLen-len(ellipsis)] + ellipsis
	}

	sideMax := w/3 - pad
	left := truncate(slots.left, sideMax)
	right := truncate(slots.right, sideMax)
	leftLen, rightLen := len(left), len(right)
	rightStart := w - pad - rightLen

	if leftLen > 0 {
		c.Write(uint(pad), y, dimFg, bg, left)
	}
	if rightLen > 0 {
		c.Write(uint(rightStart), y, dimFg, bg, right)
	}

	if slots.center == "" {
		return
	}
	leftBound := pad
	if leftLen > 0 {
		leftBound = pad + leftLen + 1
	}
	rightBound := w
	if rightLen > 0 {
		rightBound = rightStart - 1
	}
	center := truncate(slots.center, rightBound-leftBound)
	if center == "" {
		return
	}
	cx := max((w-len(center))/2, leftBound)
	if cx+len(center) > rightBound {
		cx = max(rightBound-len(center), leftBound)
	}
	c.Write(uint(cx), y, fg, bg, center)
}

// drawBookTopBar paints the text book-mode top bar on row 0: the filename
// (basename) is centred so the writer always has a reminder of the document
// they are working on, and the running word count is shown on the right.
// The matching bottom bar (see StatusBar.Draw) carries the current heading,
// line position and scroll percentage.
func (e *Editor) drawBookTopBar(c *vt.Canvas, w int) {
	base := filepath.Base(e.filename)
	if base == "" || base == "." {
		base = "untitled"
	}
	e.drawBookBar(c, 0, w, bookBarSlots{
		center: base,
		right:  fmt.Sprintf("%d words", e.WordCount()),
	})
}

// isHeadingLine reports whether the given raw line is a Markdown ATX heading
// (levels 1–6). Used by ReturnPressed to insert a blank separator line after
// a heading in book mode.
func isHeadingLine(line string) bool {
	trimmed := strings.TrimLeft(line, " ")
	// Must start flush left (no indent) to be a heading — mirrors
	// parseBookLine's indent==0 guard.
	if len(trimmed) != len(line) {
		return false
	}
	for _, pfx := range []string{"# ", "## ", "### ", "#### ", "##### ", "###### "} {
		if strings.HasPrefix(trimmed, pfx) {
			return true
		}
	}
	return false
}

// extractImgSrc extracts the src attribute value from an <img ...> tag.
// Returns the src path if found, empty string otherwise. Handles both
// src="path" and src='path' quote styles.
func extractImgSrc(htmlSnippet string) string {
	srcStart := strings.Index(htmlSnippet, "src=")
	if srcStart < 0 {
		return ""
	}
	srcStart += 4 // Move past "src="
	if srcStart >= len(htmlSnippet) {
		return ""
	}
	if htmlSnippet[srcStart] != '"' && htmlSnippet[srcStart] != '\'' {
		return ""
	}
	quote := htmlSnippet[srcStart]
	srcStart++
	endIdx := strings.IndexByte(htmlSnippet[srcStart:], quote)
	if endIdx < 0 {
		return ""
	}
	return htmlSnippet[srcStart : srcStart+endIdx]
}

// parseBookLine converts a raw Markdown line into a parsedLine, with an
// optional inComment flag to indicate the line is inside an HTML comment block.
func parseBookLine(line string) parsedLine {
	return parseBookLineInContext(line, false)
}

// parseBookLineInContext converts a raw Markdown line into a parsedLine,
// accounting for context like being inside an HTML comment block.
func parseBookLineInContext(line string, inComment bool) parsedLine {
	if strings.TrimSpace(line) == "" {
		return parsedLine{kind: lineKindBlank}
	}
	// If we're currently inside a multiline HTML comment block, skip the line
	if inComment {
		return parsedLine{kind: lineKindBlank}
	}
	// HTML comments: <!-- ... --> (render as blank to hide from output)
	trimmedWhole := strings.TrimSpace(line)
	if strings.HasPrefix(trimmedWhole, "<!--") && strings.Contains(trimmedWhole, "-->") {
		return parsedLine{kind: lineKindBlank}
	}
	if isHorizontalRule(line) {
		return parsedLine{kind: lineKindRule}
	}
	// Markdown table row: starts and ends with "|" (after trimming), and has
	// at least two pipes. The separator row "|---|:-:|---:|" is also a
	// lineKindTable (the renderer detects separator rows by shape).
	if len(trimmedWhole) >= 2 && trimmedWhole[0] == '|' &&
		trimmedWhole[len(trimmedWhole)-1] == '|' &&
		strings.Count(trimmedWhole, "|") >= 2 {
		return parsedLine{kind: lineKindTable, body: trimmedWhole}
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
	// Markdown images, possibly wrapped in a link: [![alt](imgURL)](linkURL)
	if strings.HasPrefix(trimmed, "[![") {
		// Find the closing "](" of the inner ![alt](imgURL)
		j := strings.Index(trimmed[3:], "](")
		if j >= 0 {
			imgStart := 3 + j + 2
			k := strings.Index(trimmed[imgStart:], ")")
			if k >= 0 {
				rest := trimmed[imgStart+k+1:]
				// Require the outer link pattern and nothing else on the line
				if strings.HasPrefix(rest, "](") {
					endIdx := strings.Index(rest[2:], ")")
					if endIdx >= 0 && strings.TrimSpace(rest[2+endIdx+1:]) == "" {
						imgPath := trimmed[imgStart : imgStart+k]
						return parsedLine{kind: lineKindImage, body: imgPath}
					}
				}
			}
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
	// HTML anchor tags with embedded images: <a href="..."><img src="path" .../></a>
	// Extract the image src from the img tag inside the anchor.
	if strings.HasPrefix(trimmed, "<a") && strings.Contains(trimmed, "<img") {
		imgStart := strings.Index(trimmed, "<img")
		if imgStart >= 0 {
			imgSnippet := trimmed[imgStart:]
			if imgPath := extractImgSrc(imgSnippet); imgPath != "" {
				return parsedLine{kind: lineKindImage, body: imgPath}
			}
		}
	}
	// HTML images: <img src="path" ...> or <img src='path' ...> (src must be present)
	if strings.HasPrefix(trimmed, "<img") {
		if imgPath := extractImgSrc(trimmed); imgPath != "" {
			return parsedLine{kind: lineKindImage, body: imgPath}
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
	// A leading indent of 4+ spaces is a CommonMark indented code block.
	// Note: the Tab key in book mode inserts 4 leading spaces on the first
	// line of a paragraph — that first line then renders as code, as
	// requested.
	if indent >= 4 {
		return parsedLine{kind: lineKindCode, body: trimmed}
	}
	return parsedLine{kind: lineKindBody, body: line}
}

// faceForSeg returns the font face for a text segment. Bold is served by the
// regular Vollkorn face (faux bold is applied by the caller). Italic uses the
// Vollkorn Italic face. Code segments use FiraMono Bold.
func faceForSeg(fs *bookFontSet, seg textSegment) font.Face {
	if seg.code {
		return fs.code
	}
	if seg.italic {
		return fs.italic
	}
	return fs.regular
}

// drawString renders text at (x, baselineY) using face and returns the new X.
// Runes the primary face lacks are rendered with the registered DejaVu Sans
// fallback face.
func drawString(img *image.RGBA, face font.Face, x, baselineY int, text string, clr color.Color) int {
	fb := faceFallback(face)
	if fb == nil {
		d := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(clr),
			Face: face,
			Dot:  fixed.P(x, baselineY),
		}
		d.DrawString(text)
		return d.Dot.X.Round()
	}
	// Split text into runs of same face (primary / fallback) so each run draws
	// in a single call
	src := image.NewUniform(clr)
	dot := fixed.P(x, baselineY)
	var curFace font.Face
	var buf []rune
	flush := func() {
		if len(buf) == 0 {
			return
		}
		d := &font.Drawer{Dst: img, Src: src, Face: curFace, Dot: dot}
		d.DrawString(string(buf))
		dot = d.Dot
		buf = buf[:0]
	}
	for _, r := range text {
		use := face
		if _, ok := face.GlyphAdvance(r); !ok {
			if _, ok2 := fb.GlyphAdvance(r); ok2 {
				use = fb
			}
		}
		if use != curFace {
			flush()
			curFace = use
		}
		buf = append(buf, r)
	}
	flush()
	return dot.X.Round()
}

// drawHeaderSegments renders styled inline segments of a header line at
// (x, baselineY). Non-code segments use the header face for the given level;
// code segments use a FiraMono Bold face sized to match the header so
// inline `code` inside a header visually matches the surrounding header
// text. Headers are always bold, so all segments are drawn with faux-bold.
func drawHeaderSegments(img *image.RGBA, fs *bookFontSet, level, x, baselineY int, segs []textSegment, clr color.Color) int {
	hFace := fs.headerForLevel(level)
	hCodeFace := fs.headerCodeForLevel(level)
	for _, seg := range segs {
		face := hFace
		if seg.code {
			face = hCodeFace
		}
		segClr := clr
		segUnderline := seg.underline
		if seg.linkURL != "" {
			segClr = bookLinkColor(seg.linkURL)
			segUnderline = true
		}
		endX := drawString(img, face, x, baselineY, seg.text, segClr)
		// Faux-bold: extra pass 1 px to the right (headers render bold).
		drawString(img, face, x+1, baselineY, seg.text, segClr)
		right := endX + 1
		if segUnderline {
			ulY := baselineY + 2
			if ulY < img.Bounds().Max.Y {
				for px := x; px < right; px++ {
					img.Set(px, ulY, segClr)
				}
			}
		}
		if seg.strike {
			asc := face.Metrics().Ascent.Round()
			stY := baselineY - asc/3
			if stY >= img.Bounds().Min.Y && stY < img.Bounds().Max.Y {
				for px := x; px < right; px++ {
					img.Set(px, stY, segClr)
				}
			}
		}
		x = endX + 1
	}
	return x
}

// measureHeaderSegmentsToRune returns the pixel width of the first targetRune
// runes of segs when rendered with drawHeaderSegments. Used for cursor and
// selection-rectangle positioning so they match the mixed-face output.
func measureHeaderSegmentsToRune(fs *bookFontSet, level int, segs []textSegment, targetRune int) int {
	if targetRune <= 0 {
		return 0
	}
	hFace := fs.headerForLevel(level)
	hCodeFace := fs.headerCodeForLevel(level)
	total := fixed.Int26_6(0)
	col := 0
	for _, seg := range segs {
		face := hFace
		if seg.code {
			face = hCodeFace
		}
		for _, r := range seg.text {
			if col >= targetRune {
				// Each segment contributes +1 px per rendered rune for faux-bold;
				// account for that at segment boundaries below.
				return total.Round() + col
			}
			adv, ok := face.GlyphAdvance(r)
			if !ok {
				if fb := faceFallback(face); fb != nil {
					if a2, ok2 := fb.GlyphAdvance(r); ok2 {
						adv = a2
						ok = true
					}
				}
			}
			if ok {
				total += adv
			}
			col++
		}
	}
	// Add +1 px per rune for faux-bold drift.
	return total.Round() + col
}

// bookLinkURLUnderCursor returns the URL of a Markdown link "[text](url)"
// whose raw source contains the current cursor position, or "" if no link
// is under the cursor. The scan treats the cursor rune index as inclusive
// on the opening "[" and exclusive on the closing ")".
func (e *Editor) bookLinkURLUnderCursor() string {
	dataX, err := e.DataX()
	if err != nil {
		return ""
	}
	line := e.CurrentLine()
	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		if runes[i] != '[' {
			continue
		}
		// Find the matching "]" for the link text.
		j := i + 1
		depth := 1
		for j < len(runes) && depth > 0 {
			switch runes[j] {
			case '[':
				depth++
			case ']':
				depth--
				if depth == 0 {
					break
				}
			}
			if depth == 0 {
				break
			}
			j++
		}
		if j >= len(runes) || runes[j] != ']' {
			continue
		}
		if j+1 >= len(runes) || runes[j+1] != '(' {
			continue
		}
		// Find the matching ")" (respecting nesting).
		k := j + 2
		pdepth := 1
		for k < len(runes) && pdepth > 0 {
			switch runes[k] {
			case '(':
				pdepth++
			case ')':
				pdepth--
				if pdepth == 0 {
					break
				}
			}
			if pdepth == 0 {
				break
			}
			k++
		}
		if k >= len(runes) || runes[k] != ')' {
			continue
		}
		// Cursor inside the whole "[text](url)" span?
		if dataX >= i && dataX <= k {
			return strings.TrimSpace(string(runes[j+2 : k]))
		}
		i = k
	}
	return ""
}

// bookLinkUnvisited is the paint colour for unvisited Markdown links in
// graphical book mode. bookLinkVisited is used after a link has been
// followed during this session.
var (
	bookLinkUnvisited = color.NRGBA{0x46, 0xa3, 0xbf, 0xff}
	bookLinkVisited   = color.NRGBA{0x7e, 0x46, 0xbf, 0xff}
)

// bookVisitedLinks is a session-wide, concurrency-safe set of URLs the user
// has followed via Ctrl-R in book mode. The set is intentionally not
// persisted to disk — it resets on every Orbiton invocation.
var bookVisitedLinks sync.Map

// bookMarkLinkVisited records that url has been visited this session so
// future paints switch it to the visited colour.
func bookMarkLinkVisited(url string) {
	if url == "" {
		return
	}
	bookVisitedLinks.Store(url, struct{}{})
}

// bookIsLinkVisited reports whether url has been visited this session.
func bookIsLinkVisited(url string) bool {
	if url == "" {
		return false
	}
	_, ok := bookVisitedLinks.Load(url)
	return ok
}

// bookLinkColor returns the colour to paint a link with url (visited or not).
func bookLinkColor(url string) color.Color {
	if bookIsLinkVisited(url) {
		return bookLinkVisited
	}
	return bookLinkUnvisited
}

// Session-wide navigation history for book-mode URL browsing. bookCurrentURL
// remembers the URL of the document currently displayed (empty for local
// files). bookURLHistory is a stack of previously-visited URLs: following a
// link pushes the current URL, Ctrl-R on a non-link location pops and loads
// the top entry.
var (
	bookHistoryMu  sync.Mutex
	bookCurrentURL string
	bookURLHistory []string
)

// bookSetCurrentURL records the URL of the document currently being viewed.
// Called from main.go on startup and from the link-follow handler.
func bookSetCurrentURL(url string) {
	bookHistoryMu.Lock()
	bookCurrentURL = url
	bookHistoryMu.Unlock()
}

// bookGetCurrentURL returns the URL of the document currently being viewed,
// or "" if none.
func bookGetCurrentURL() string {
	bookHistoryMu.Lock()
	defer bookHistoryMu.Unlock()
	return bookCurrentURL
}

// bookPushHistory appends url to the back-navigation stack. Consecutive
// duplicates are collapsed so rapid re-follows don't bloat the stack.
func bookPushHistory(url string) {
	if url == "" {
		return
	}
	bookHistoryMu.Lock()
	defer bookHistoryMu.Unlock()
	n := len(bookURLHistory)
	if n > 0 && bookURLHistory[n-1] == url {
		return
	}
	bookURLHistory = append(bookURLHistory, url)
}

// bookPopHistory removes and returns the most recently pushed URL, or "" if
// the stack is empty.
func bookPopHistory() string {
	bookHistoryMu.Lock()
	defer bookHistoryMu.Unlock()
	n := len(bookURLHistory)
	if n == 0 {
		return ""
	}
	top := bookURLHistory[n-1]
	bookURLHistory = bookURLHistory[:n-1]
	return top
}

func drawSegments(img *image.RGBA, fs *bookFontSet, x, baselineY int, segs []textSegment, clr color.Color) int {
	for _, seg := range segs {
		face := faceForSeg(fs, seg)
		// Links are painted in a distinct colour and always underlined.
		segClr := clr
		segUnderline := seg.underline
		if seg.linkURL != "" {
			segClr = bookLinkColor(seg.linkURL)
			segUnderline = true
		}
		endX := drawString(img, face, x, baselineY, seg.text, segClr)
		if seg.bold {
			// Faux bold: draw again 1 px to the right for thicker strokes
			drawString(img, face, x+1, baselineY, seg.text, segClr)
		}
		right := endX
		if seg.bold {
			right++
		}
		if segUnderline {
			ulY := baselineY + 2
			if ulY < img.Bounds().Max.Y {
				for px := x; px < right; px++ {
					img.Set(px, ulY, segClr)
				}
			}
		}
		if seg.strike {
			// Strikethrough: horizontal line roughly at x-height centre.
			asc := face.Metrics().Ascent.Round()
			stY := baselineY - asc/3
			if stY >= img.Bounds().Min.Y && stY < img.Bounds().Max.Y {
				for px := x; px < right; px++ {
					img.Set(px, stY, segClr)
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
			adv, ok := faceGlyphAdvance(face, r)
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
	bookImgCache    = map[string]image.Image{}
	bookImgInFlight = map[string]bool{}
	bookImgCacheMu  sync.Mutex

	// Download images when rendering?
	bookDownloadImages = true
)

// bookIsRemoteURL reports whether imgPath is an http(s) URL.
func bookIsRemoteURL(imgPath string) bool {
	if len(imgPath) < 7 {
		return false
	}
	lower := strings.ToLower(imgPath)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// svgTextSpan describes a single piece of text extracted from an SVG.
// All coordinates are in the SVG's user-space (before the viewBox→pixel
// mapping) and already have the cumulative ancestor transforms folded in
// (only scale/translate are honored — enough for shields.io-style badges).
type svgTextSpan struct {
	x, y        float64
	text        string
	fontSize    float64
	fill        color.NRGBA
	anchor      string // "start" | "middle" | "end"
	bold        bool
	italic      bool
	monospace   bool
	fillOpacity float64
}

// svgAttrFrame captures the CSS/presentation attributes inherited down the
// SVG element tree.
type svgAttrFrame struct {
	fontSize    float64
	fill        color.NRGBA
	fillSet     bool
	anchor      string
	family      string
	bold        bool
	italic      bool
	scaleX      float64
	scaleY      float64
	tx, ty      float64
	fillOpacity float64
}

var (
	svgNumRe       = regexp.MustCompile(`-?(?:\d+(?:\.\d+)?|\.\d+)`)
	svgHexShortRe  = regexp.MustCompile(`^#([0-9A-Fa-f]{3})$`)
	svgHexLongRe   = regexp.MustCompile(`^#([0-9A-Fa-f]{6})$`)
	svgRGBFuncRe   = regexp.MustCompile(`^rgb\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*\)$`)
	svgTranslateRe = regexp.MustCompile(`translate\s*\(\s*(-?(?:\d+(?:\.\d+)?|\.\d+))(?:[,\s]+(-?(?:\d+(?:\.\d+)?|\.\d+)))?\s*\)`)
	svgScaleRe     = regexp.MustCompile(`scale\s*\(\s*(-?(?:\d+(?:\.\d+)?|\.\d+))(?:[,\s]+(-?(?:\d+(?:\.\d+)?|\.\d+)))?\s*\)`)
)

// svgParseColor parses a subset of SVG color strings: #rgb, #rrggbb, rgb(r,g,b)
// and the common named colors used by badges. Returns (c, ok).
func svgParseColor(s string) (color.NRGBA, bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == "none" || s == "transparent" || s == "inherit" {
		return color.NRGBA{}, false
	}
	if m := svgHexLongRe.FindStringSubmatch(s); m != nil {
		v, err := strconv.ParseUint(m[1], 16, 32)
		if err != nil {
			return color.NRGBA{}, false
		}
		return color.NRGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}, true
	}
	if m := svgHexShortRe.FindStringSubmatch(s); m != nil {
		r, _ := strconv.ParseUint(string(m[1][0]), 16, 32)
		g, _ := strconv.ParseUint(string(m[1][1]), 16, 32)
		b, _ := strconv.ParseUint(string(m[1][2]), 16, 32)
		return color.NRGBA{uint8(r * 17), uint8(g * 17), uint8(b * 17), 0xff}, true
	}
	if m := svgRGBFuncRe.FindStringSubmatch(s); m != nil {
		r, _ := strconv.Atoi(m[1])
		g, _ := strconv.Atoi(m[2])
		b, _ := strconv.Atoi(m[3])
		return color.NRGBA{uint8(r), uint8(g), uint8(b), 0xff}, true
	}
	switch strings.ToLower(s) {
	case "white":
		return color.NRGBA{0xff, 0xff, 0xff, 0xff}, true
	case "black":
		return color.NRGBA{0, 0, 0, 0xff}, true
	case "red":
		return color.NRGBA{0xff, 0, 0, 0xff}, true
	case "green":
		return color.NRGBA{0, 0x80, 0, 0xff}, true
	case "blue":
		return color.NRGBA{0, 0, 0xff, 0xff}, true
	case "gray", "grey":
		return color.NRGBA{0x80, 0x80, 0x80, 0xff}, true
	}
	return color.NRGBA{}, false
}

// svgParseStyle returns a map of name→value from a CSS-like style string
// ("font-size:11px;fill:#fff").
func svgParseStyle(s string) map[string]string {
	m := map[string]string{}
	for p := range strings.SplitSeq(s, ";") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if i := strings.IndexByte(p, ':'); i > 0 {
			k := strings.TrimSpace(p[:i])
			v := strings.TrimSpace(p[i+1:])
			m[k] = v
		}
	}
	return m
}

// svgParseLength parses "11px" / "11" as a float. Returns 0 on failure.
func svgParseLength(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	for _, suf := range []string{"px", "pt", "em"} {
		s = strings.TrimSuffix(s, suf)
	}
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

// svgApplyAttrs returns a new frame derived from parent with the given
// element attributes merged in. Only a subset of attributes used by common
// badge SVGs are honored: font-size, font-family, font-weight, font-style,
// fill, fill-opacity, text-anchor, transform (translate + scale).
func svgApplyAttrs(parent svgAttrFrame, attrs []xml.Attr) svgAttrFrame {
	f := parent
	// Merge style= into named attrs first so explicit attrs win.
	var styleMap map[string]string
	for _, a := range attrs {
		if a.Name.Local == "style" {
			styleMap = svgParseStyle(a.Value)
			break
		}
	}
	apply := func(name, value string) {
		switch name {
		case "font-size":
			if v := svgParseLength(value); v > 0 {
				f.fontSize = v
			}
		case "font-family":
			f.family = value
		case "font-weight":
			v := strings.TrimSpace(value)
			f.bold = v == "bold" || v == "bolder" || v == "700" || v == "800" || v == "900"
		case "font-style":
			f.italic = strings.TrimSpace(value) == "italic"
		case "fill":
			if c, ok := svgParseColor(value); ok {
				f.fill = c
				f.fillSet = true
			}
		case "fill-opacity":
			if v, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
				f.fillOpacity = v
			}
		case "text-anchor":
			f.anchor = strings.TrimSpace(value)
		case "transform":
			// Compose translate/scale (ignore rotate/skew — uncommon in badges).
			if m := svgTranslateRe.FindStringSubmatch(value); m != nil {
				tx, _ := strconv.ParseFloat(m[1], 64)
				ty := 0.0
				if m[2] != "" {
					ty, _ = strconv.ParseFloat(m[2], 64)
				}
				// The new translate maps child (0,0) → (parent+scale*tx, ...)
				f.tx += f.scaleX * tx
				f.ty += f.scaleY * ty
			}
			if m := svgScaleRe.FindStringSubmatch(value); m != nil {
				sx, _ := strconv.ParseFloat(m[1], 64)
				sy := sx
				if m[2] != "" {
					sy, _ = strconv.ParseFloat(m[2], 64)
				}
				f.scaleX *= sx
				f.scaleY *= sy
			}
		}
	}
	for k, v := range styleMap {
		apply(k, v)
	}
	for _, a := range attrs {
		if a.Name.Local == "style" {
			continue
		}
		apply(a.Name.Local, a.Value)
	}
	return f
}

// svgExtractTextSpans parses an SVG document and returns the viewBox width
// and height plus all <text> / <tspan> leaves in render order. When the
// document cannot be parsed, all return values are zero/nil.
func svgExtractTextSpans(data []byte) (viewW, viewH float64, spans []svgTextSpan) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	dec.Entity = xml.HTMLEntity

	type frameLevel struct {
		frame      svgAttrFrame
		inText     bool
		textBuf    strings.Builder
		xOverride  float64
		yOverride  float64
		hasX, hasY bool
	}
	root := svgAttrFrame{
		fontSize:    16,
		fill:        color.NRGBA{0, 0, 0, 0xff},
		anchor:      "start",
		scaleX:      1,
		scaleY:      1,
		fillOpacity: 1,
	}
	stack := []frameLevel{{frame: root}}

	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			parent := stack[len(stack)-1].frame
			cur := svgApplyAttrs(parent, t.Attr)
			lvl := frameLevel{frame: cur}
			lvl.xOverride = parent.tx
			lvl.yOverride = parent.ty
			// Snapshot the x/y on <text>/<tspan> before the element's own
			// transform is applied — SVG applies attributes first, then
			// transforms. But shields.io uses only transform="scale(...)"
			// and absolute x/y on <text>. We fold as: final = parentTrans +
			// parentScale*(x,y)*ownScaleFactor — for the common case of a
			// single scale on the element, parentScale is 1.
			for _, a := range t.Attr {
				switch a.Name.Local {
				case "x":
					if v, err := strconv.ParseFloat(strings.TrimSpace(a.Value), 64); err == nil {
						lvl.xOverride = v
						lvl.hasX = true
					}
				case "y":
					if v, err := strconv.ParseFloat(strings.TrimSpace(a.Value), 64); err == nil {
						lvl.yOverride = v
						lvl.hasY = true
					}
				}
			}
			if strings.EqualFold(t.Name.Local, "svg") {
				// viewBox="minX minY W H"
				for _, a := range t.Attr {
					if a.Name.Local == "viewBox" {
						nums := svgNumRe.FindAllString(a.Value, -1)
						if len(nums) >= 4 {
							w, _ := strconv.ParseFloat(nums[2], 64)
							h, _ := strconv.ParseFloat(nums[3], 64)
							viewW, viewH = w, h
						}
					}
					if a.Name.Local == "width" && viewW == 0 {
						viewW = svgParseLength(a.Value)
					}
					if a.Name.Local == "height" && viewH == 0 {
						viewH = svgParseLength(a.Value)
					}
				}
			}
			if strings.EqualFold(t.Name.Local, "text") || strings.EqualFold(t.Name.Local, "tspan") {
				lvl.inText = true
			}
			stack = append(stack, lvl)
		case xml.CharData:
			top := &stack[len(stack)-1]
			if top.inText {
				top.textBuf.WriteString(string(t))
			}
		case xml.EndElement:
			top := stack[len(stack)-1]
			if top.inText {
				text := strings.TrimSpace(top.textBuf.String())
				if text != "" {
					f := top.frame
					// f.tx / f.scaleX already include all ancestor and own
					// transforms (svgApplyAttrs composes them left-to-right).
					// If this element provided x/y, map them through the full
					// transform; otherwise use the accumulated translation.
					x := f.tx
					y := f.ty
					if top.hasX {
						x = f.tx + f.scaleX*top.xOverride
					}
					if top.hasY {
						y = f.ty + f.scaleY*top.yOverride
					}
					op := f.fillOpacity
					if op <= 0 {
						op = 1
					}
					fc := f.fill
					fc.A = uint8(float64(fc.A) * op)
					spans = append(spans, svgTextSpan{
						x:           x,
						y:           y,
						text:        text,
						fontSize:    f.fontSize * f.scaleY,
						fill:        fc,
						anchor:      f.anchor,
						bold:        f.bold,
						italic:      f.italic,
						monospace:   strings.Contains(strings.ToLower(f.family), "mono") || strings.Contains(strings.ToLower(f.family), "courier"),
						fillOpacity: op,
					})
				}
			}
			stack = stack[:len(stack)-1]
		}
	}
	return viewW, viewH, spans
}

// bookRenderSVGTextOverlay draws <text>/<tspan> content onto img after oksvg
// has rasterized the shapes. oksvg itself doesn't render SVG text, so this
// recovers legibility for shields.io-style badges and similar SVGs that rely
// on <text>. Text is rendered with Orbiton's bundled fonts — DejaVu Sans is
// used as the Unicode-safe fallback for "sans-serif" / "Verdana" / "Arial"
// requests, Fira Mono Bold for monospace, and Vollkorn for serif.
func bookRenderSVGTextOverlay(img *image.RGBA, data []byte, viewW, viewH float64) {
	w, h := viewW, viewH
	if w <= 0 || h <= 0 {
		b := img.Bounds()
		w, h = float64(b.Dx()), float64(b.Dy())
	}
	_, _, spans := svgExtractTextSpans(data)
	if len(spans) == 0 {
		return
	}
	b := img.Bounds()
	sx := float64(b.Dx()) / w
	sy := float64(b.Dy()) / h
	_ = parsedFonts() // ensure fonts are loaded; safe to ignore error here
	for _, sp := range spans {
		// Apply a small legibility adjustment: DejaVu Sans at the SVG's
		// declared px size tends to render visually larger than Verdana
		// (shields.io's intended face), which nudges text past the badge
		// edges. Shave a touch off the size so it fits cleanly.
		size := sp.fontSize * sy * 0.82
		if size < 4 {
			size = 4
		}
		// Pick a face
		var baseFont *opentype.Font
		switch {
		case sp.monospace && parsedFiraMonoBold != nil:
			baseFont = parsedFiraMonoBold
		case sp.bold && parsedMontserratBold != nil:
			baseFont = parsedMontserratBold
		case parsedDejaVuSans != nil:
			baseFont = parsedDejaVuSans
		case parsedMontserratLight != nil:
			baseFont = parsedMontserratLight
		case parsedVollkornRegular != nil:
			baseFont = parsedVollkornRegular
		}
		if baseFont == nil {
			continue
		}
		face, err := newFace(baseFont, size)
		if err != nil {
			continue
		}
		// Measure for anchoring
		width := measureStringFB(face, sp.text).Round()
		px := sp.x * sx
		py := sp.y * sy
		switch strings.ToLower(sp.anchor) {
		case "middle":
			px -= float64(width) / 2
		case "end":
			px -= float64(width)
		}
		clr := sp.fill
		if clr.A == 0 {
			continue
		}
		drawString(img, face, int(px+0.5), int(py+0.5), sp.text, clr)
		face.Close()
	}
}

func bookLooksLikeSVG(data []byte) bool {
	// Scan past whitespace and an optional XML prolog / doctype / comments
	i := 0
	for i < len(data) {
		for i < len(data) && (data[i] == ' ' || data[i] == '\t' || data[i] == '\n' || data[i] == '\r') {
			i++
		}
		if i >= len(data) || data[i] != '<' {
			return false
		}
		rest := data[i:]
		switch {
		case bytes.HasPrefix(rest, []byte("<?xml")):
			end := bytes.Index(rest, []byte("?>"))
			if end < 0 {
				return false
			}
			i += end + 2
		case bytes.HasPrefix(rest, []byte("<!--")):
			end := bytes.Index(rest, []byte("-->"))
			if end < 0 {
				return false
			}
			i += end + 3
		case bytes.HasPrefix(rest, []byte("<!DOCTYPE")), bytes.HasPrefix(rest, []byte("<!doctype")):
			end := bytes.IndexByte(rest, '>')
			if end < 0 {
				return false
			}
			i += end + 1
		case bytes.HasPrefix(rest, []byte("<svg")):
			return true
		default:
			return false
		}
	}
	return false
}

// bookRenderSVG rasterises SVG bytes to an image.Image. Returns nil on any
// parse or render error. oksvg only handles the geometric shapes, so any
// <text> / <tspan> content is composited afterwards using Orbiton's built-in
// fonts (bookRenderSVGTextOverlay).
func bookRenderSVG(data []byte) image.Image {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	w := 800
	h := 600
	if icon.ViewBox.W > 0 {
		w = int(icon.ViewBox.W)
		h = int(icon.ViewBox.H)
		if h <= 0 {
			h = w
		}
	}
	// Cap the raster size so a large viewBox can't blow up memory
	const maxDim = 2048
	if w > maxDim {
		scale := float64(maxDim) / float64(w)
		w = maxDim
		h = int(float64(h) * scale)
	}
	if h > maxDim {
		scale := float64(maxDim) / float64(h)
		h = maxDim
		w = int(float64(w) * scale)
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	icon.SetTarget(0, 0, float64(w), float64(h))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	dasher := rasterx.NewDasher(w, h, scanner)
	icon.Draw(dasher, 1.0)
	// Render any SVG <text> / <tspan> elements on top. oksvg ignores text
	// entirely, so this is what makes shields.io badges legible.
	bookRenderSVGTextOverlay(img, data, icon.ViewBox.W, icon.ViewBox.H)
	return img
}

// bookDownloadImage fetches a remote image and decodes it. Returns nil on
// any failure so the caller can fall back to rendering without the image.
// Handles SVG via oksvg/rasterx as well as standard PNG/JPEG/GIF.
func bookDownloadImage(rawURL string) image.Image {
	if !bookIsRemoteURL(rawURL) {
		return nil
	}
	resp, err := httpGet(rawURL, map[string]string{"User-Agent": "orbiton"}, 10*time.Second)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	const maxBytes = 16 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return nil
	}
	// Treat both by URL extension and by content sniffing: remote SVGs often
	// have path components like "/badge.svg" or Content-Type "image/svg+xml"
	// but some redirect-served URLs don't, so check the payload too
	contentType := resp.Header["content-type"]
	if strings.HasSuffix(strings.ToLower(rawURL), ".svg") ||
		strings.Contains(contentType, "svg") ||
		bookLooksLikeSVG(data) {
		if img := bookRenderSVG(data); img != nil {
			return img
		}
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return img
}

// bookLoadImage loads and caches an image by absolute path or remote URL.
// Returns nil if the image cannot be loaded or decoded. Remote URLs are
// fetched asynchronously in the background (only when --downloadimages is
// enabled) so the render loop is never blocked on the network; subsequent
// frames pick up completed downloads.
func bookLoadImage(absPath string) image.Image {
	bookImgCacheMu.Lock()
	if img, ok := bookImgCache[absPath]; ok {
		bookImgCacheMu.Unlock()
		return img
	}
	if bookIsRemoteURL(absPath) {
		if !bookDownloadImages {
			bookImgCache[absPath] = nil
			bookImgCacheMu.Unlock()
			return nil
		}
		if bookImgInFlight[absPath] {
			bookImgCacheMu.Unlock()
			return nil
		}
		bookImgInFlight[absPath] = true
		bookImgCacheMu.Unlock()
		go func(u string) {
			nimg := bookDownloadImage(u)
			bookImgCacheMu.Lock()
			bookImgCache[u] = nimg
			delete(bookImgInFlight, u)
			bookImgCacheMu.Unlock()
			// Invalidate the content cache so the next frame includes
			// the newly-available image
			bookBumpContentGen()
		}(absPath)
		return nil
	}
	bookImgCacheMu.Unlock()
	nimg := bookLoadLocalImage(absPath)
	bookImgCacheMu.Lock()
	bookImgCache[absPath] = nimg
	bookImgCacheMu.Unlock()
	return nimg
}

// bookLoadLocalImage loads and decodes a local image file, with SVG support
// via oksvg/rasterx. Returns nil on error.
func bookLoadLocalImage(absPath string) image.Image {
	if strings.HasSuffix(strings.ToLower(absPath), ".svg") {
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil
		}
		return bookRenderSVG(data)
	}
	nimg, err := imagepreview.LoadImage(absPath)
	if err != nil {
		return nil
	}
	return nimg
}

// resolveBookImagePath resolves an image path relative to the document.
// Remote (http/https) paths are returned unchanged.
func (e *Editor) resolveBookImagePath(imgPath string) string {
	if bookIsRemoteURL(imgPath) {
		return imgPath
	}
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
	if maxW < 1 || maxH < 1 {
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
	if scale <= 0 {
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

// bookImagePlacement describes one image's position within an image group.
type bookImagePlacement struct {
	img     image.Image
	dstX    int // x relative to the group's left edge (marginLeft)
	dstY    int // y relative to the group's top edge (cellTop)
	width   int
	height  int
	loading bool // true if the image is still being fetched (placeholder)
}

// bookCollectImageGroup returns the list of image URLs for an image-only
// paragraph starting at startLine, plus how many source lines it spans.
// Returns (nil, 0) when startLine is not an image line. Adjacent image-only
// lines with no blank line between them are treated as a single paragraph,
// matching Markdown's paragraph-coalescing semantics.
func (e *Editor) bookCollectImageGroup(startLine int) ([]string, int) {
	total := e.Len()
	if startLine >= total {
		return nil, 0
	}
	raw := strings.ReplaceAll(e.Line(LineIndex(startLine)), "\t", "    ")
	pl := parseBookLine(raw)
	if pl.kind != lineKindImage {
		return nil, 0
	}
	urls := []string{pl.body}
	i := startLine + 1
	for i < total {
		r := strings.ReplaceAll(e.Line(LineIndex(i)), "\t", "    ")
		pl2 := parseBookLine(r)
		if pl2.kind != lineKindImage {
			break
		}
		urls = append(urls, pl2.body)
		i++
	}
	return urls, i - startLine
}

// bookLayoutImageGroup places images left-to-right, wrapping to a new row
// when the next image would overflow availW. Each image is scaled to fit
// bookMaxImageRows*lineH tall (and never wider than availW). A missing or
// not-yet-downloaded image gets a small placeholder slot so the paragraph
// takes roughly the same footprint before and after async image loads
// complete. Returns the placements and total rows consumed.
func (e *Editor) bookLayoutImageGroup(urls []string, availW, lineH int) ([]bookImagePlacement, int) {
	if availW < 1 {
		availW = 1
	}
	if lineH < 1 {
		lineH = 1
	}
	const gapX = 6
	const gapY = 2
	maxH := bookMaxImageRows * lineH
	// Placeholder size for images that aren't loaded yet — roughly matches a
	// typical shields.io badge so the layout doesn't snap when downloads land.
	phW := min(120, availW)
	phH := max(lineH*9/10, 14)

	var placed []bookImagePlacement
	curX, rowTop, rowH := 0, 0, 0
	for _, u := range urls {
		abs := e.resolveBookImagePath(u)
		src := bookLoadImage(abs)
		var w, h int
		loading := src == nil
		if loading {
			w, h = phW, phH
		} else {
			b := src.Bounds()
			sw, sh := b.Dx(), b.Dy()
			if sw <= 0 || sh <= 0 {
				continue
			}
			scale := 1.0
			if sh > maxH {
				scale = float64(maxH) / float64(sh)
			}
			if float64(sw)*scale > float64(availW) {
				scale = float64(availW) / float64(sw)
			}
			w = max(int(float64(sw)*scale), 1)
			h = max(int(float64(sh)*scale), 1)
		}
		// Wrap when the image wouldn't fit on the current row (but never on
		// an empty row — if it's too wide it goes alone and gets clipped).
		if curX > 0 && curX+w > availW {
			rowTop += rowH + gapY
			curX = 0
			rowH = 0
		}
		placed = append(placed, bookImagePlacement{img: src, dstX: curX, dstY: rowTop, width: w, height: h, loading: loading})
		curX += w + gapX
		if h > rowH {
			rowH = h
		}
	}
	totalPx := rowTop + rowH
	rows := max((totalPx+lineH-1)/lineH, 1)
	return placed, rows
}

// bookImageGroupRows returns how many display rows an image paragraph will
// occupy, without drawing anything. Used during row counting / cursor mapping.
func (e *Editor) bookImageGroupRows(urls []string, availW, lineH int) int {
	_, rows := e.bookLayoutImageGroup(urls, availW, lineH)
	return rows
}

// bookDrawImageGroup draws a group of images at (marginLeft, cellTop), laid
// out horizontally with wrapping. Loading placeholders render as a faint
// rounded rectangle with three animated circles to indicate loading. Returns the
// number of display rows consumed.
func (e *Editor) bookDrawImageGroup(img *image.RGBA, urls []string, marginLeft, marginRight, cellTop, lineH int) int {
	availW := marginRight - marginLeft
	placed, rows := e.bookLayoutImageGroup(urls, availW, lineH)
	phBg := color.NRGBA{0xe0, 0xe0, 0xe0, 0xff}
	phBorder := color.NRGBA{0xb0, 0xb0, 0xb0, 0xff}
	if e.bookDarkMode {
		phBg = color.NRGBA{0x2a, 0x2a, 0x2a, 0xff}
		phBorder = color.NRGBA{0x55, 0x55, 0x55, 0xff}
	}
	for _, p := range placed {
		if p.loading {
			rect := image.Rect(marginLeft+p.dstX, cellTop+p.dstY, marginLeft+p.dstX+p.width, cellTop+p.dstY+p.height)
			draw.Draw(img, rect, image.NewUniform(phBg), image.Point{}, draw.Src)
			for x := rect.Min.X; x < rect.Max.X; x++ {
				img.Set(x, rect.Min.Y, phBorder)
				img.Set(x, rect.Max.Y-1, phBorder)
			}
			for y := rect.Min.Y; y < rect.Max.Y; y++ {
				img.Set(rect.Min.X, y, phBorder)
				img.Set(rect.Max.X-1, y, phBorder)
			}

			// Draw three circles in the center to indicate loading
			centerX := rect.Min.X + (rect.Dx() / 2)
			centerY := rect.Min.Y + (rect.Dy() / 2)
			circleRadius := 3
			circleDist := 12

			// Choose circle color based on theme
			circleColor := color.NRGBA{0x4a, 0xb0, 0x60, 0xff} // green
			if e.bookDarkMode {
				circleColor = color.NRGBA{0x5a, 0xd0, 0x70, 0xff} // bright green
			}

			// Draw three circles
			for i := -1; i <= 1; i++ {
				x := centerX + i*circleDist
				e.bookDrawCircle(img, x, centerY, circleRadius, circleColor)
			}
			continue
		}
		scaled := bookScaleImage(p.img, p.width, p.height)
		b := scaled.Bounds()
		dstRect := image.Rect(marginLeft+p.dstX, cellTop+p.dstY, marginLeft+p.dstX+b.Dx(), cellTop+p.dstY+b.Dy())
		draw.Draw(img, dstRect, scaled, b.Min, draw.Src)
	}
	return rows
}

// bookImageRows returns the number of display rows a Markdown image line will
// occupy. The size is fixed — it does not depend on where on the page the image
// appears — so that images don't change size when scrolled.
// Returns 1 if the image cannot be loaded.
func (e *Editor) bookImageRows(imgPath string, lineH, maxW int) int {
	if lineH < 1 {
		lineH = 1
	}
	if maxW < 1 {
		return 1
	}
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
	maxH := bookMaxImageRows * lineH // fixed ceiling — no available-height dependency
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

// bookDrawCircle draws a filled circle at (centerX, centerY) with given radius and color
func (e *Editor) bookDrawCircle(img *image.RGBA, centerX, centerY, radius int, col color.NRGBA) {
	r2 := radius * radius
	for y := centerY - radius; y <= centerY+radius; y++ {
		for x := centerX - radius; x <= centerX+radius; x++ {
			dx := x - centerX
			dy := y - centerY
			if dx*dx+dy*dy <= r2 {
				img.Set(x, y, col)
			}
		}
	}
}

// bookDrawInlineImage loads, scales and draws a Markdown image into img at
// (marginLeft, cellTop). Returns the number of display rows consumed.
// Image size is fixed (same as bookImageRows) — it does not shrink near the
// bottom of the page. Pixels outside img.Bounds() are automatically clipped.
func (e *Editor) bookDrawInlineImage(img *image.RGBA, imgPath string, marginLeft, marginRight, cellTop, lineH int) int {
	if lineH < 1 {
		lineH = 1
	}
	abs := e.resolveBookImagePath(imgPath)
	src := bookLoadImage(abs)
	if src == nil {
		return 1
	}
	maxW := marginRight - marginLeft
	maxH := bookMaxImageRows * lineH // fixed ceiling — no available-height dependency
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
func (e *Editor) bookDrawCheckbox(img *image.RGBA, x, cellTop, lineH int, checked bool) int {
	size := max(
		// ~55 % of line height
		lineH*55/100, 5)
	topY := cellTop + (lineH-size)/2

	var border, fill color.NRGBA
	if e.bookDarkMode {
		border = color.NRGBA{0xaa, 0xaa, 0xaa, 0xff}
		if checked {
			fill = color.NRGBA{0x1f, 0x3a, 0x1f, 0xff} // muted dark green
		} else {
			fill = color.NRGBA{0x2a, 0x2a, 0x2a, 0xff} // slightly lighter than page
		}
	} else {
		border = color.NRGBA{0x55, 0x55, 0x55, 0xff}
		if checked {
			fill = color.NRGBA{0xcc, 0xf0, 0xcc, 0xff} // light green
		} else {
			fill = color.NRGBA{0xf8, 0xf8, 0xf8, 0xff} // near-white
		}
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
		if i+1 < len(runes) && runes[i] == '~' && runes[i+1] == '~' {
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

// visualXToRawX is the inverse of rawXToVisualX: given a visual (rendered)
// rune offset into a line, return the corresponding raw rune offset that
// includes Markdown formatting characters (*, **, ___, `).
func visualXToRawX(line string, visX int) int {
	runes := []rune(line)
	vis := 0
	i := 0
	for i < len(runes) && vis < visX {
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
		if i+1 < len(runes) && runes[i] == '~' && runes[i+1] == '~' {
			i += 2
			continue
		}
		if runes[i] == '`' {
			i++
			continue
		}
		vis++
		i++
	}
	// Skip any trailing formatting markers so the cursor lands after them.
	for i < len(runes) {
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
		if i+1 < len(runes) && runes[i] == '~' && runes[i+1] == '~' {
			i += 2
			continue
		}
		if runes[i] == '`' {
			i++
			continue
		}
		break
	}
	return i
}

// bookWrapLine splits a body/bullet/numbered/checked/unchecked line into display
// segments that each fit within availW runes. Headers, code, images, rules and
// blanks return a single segment. The first segment keeps any line prefix;
// continuation segments are indented to the same column. Each segment is a
// rune-range [start, end) into the raw line.
type wrapSegment struct {
	start int // rune offset into the raw line (for cursor mapping)
	end   int // exclusive rune end
	text  string
}

// bookWrapPlainRunes wraps s into consecutive rune-chunks of at most width
// runes each, with no word-boundary awareness. Used by the text book mode
// for content types (headers, code lines, table rows, image placeholders)
// where the text is already either pre-formatted or irreducible, so a
// word-aware wrap would either be lossy or look worse than a hard split.
func bookWrapPlainRunes(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	runes := []rune(s)
	if len(runes) <= width {
		return []string{s}
	}
	var out []string
	for i := 0; i < len(runes); i += width {
		end := min(i+width, len(runes))
		out = append(out, string(runes[i:end]))
	}
	return out
}

// bookWrapBody wraps a body string (no prefix) to fit within availW visible
// columns, breaking at word boundaries. Visible columns ignore Markdown
// inline markers (*, **, ***, __, ~~ and `), so the wrap result reflects
// what the reader will actually see, while the returned rune offsets still
// refer to the raw body string (needed for cursor mapping).
func bookWrapBody(body string, availW int) []wrapSegment {
	if availW <= 0 {
		availW = 1
	}
	runes := []rune(body)
	if len(runes) == 0 {
		return []wrapSegment{{0, 0, body}}
	}
	// markerLen returns how many runes make up a Markdown marker at
	// position p that should be counted as zero visible width, or 0 if
	// there is no marker there.
	markerLen := func(p int) int {
		if p+2 < len(runes) && runes[p] == '*' && runes[p+1] == '*' && runes[p+2] == '*' {
			return 3
		}
		if p+1 < len(runes) && runes[p] == '*' && runes[p+1] == '*' {
			return 2
		}
		if runes[p] == '*' {
			return 1
		}
		if p+1 < len(runes) && runes[p] == '_' && runes[p+1] == '_' {
			return 2
		}
		if p+1 < len(runes) && runes[p] == '~' && runes[p+1] == '~' {
			return 2
		}
		if runes[p] == '`' {
			return 1
		}
		return 0
	}
	var segs []wrapSegment
	pos := 0
	for pos < len(runes) {
		rowStart := pos
		visCount := 0
		lastSpaceEnd := -1 // raw index just after the last space seen on this row
		end := pos
		for end < len(runes) {
			if ml := markerLen(end); ml > 0 {
				end += ml
				continue
			}
			if visCount >= availW {
				break
			}
			visCount++
			// Record the break point only for the first space in a run of
			// spaces, so consecutive spaces stay visible as leading
			// whitespace on the next row instead of being absorbed into
			// the invisible trailing region of the current row.
			if runes[end] == ' ' && (end == rowStart || runes[end-1] != ' ') {
				lastSpaceEnd = end + 1
			}
			end++
		}
		if end >= len(runes) {
			segs = append(segs, wrapSegment{rowStart, len(runes), string(runes[rowStart:])})
			break
		}
		brk := end
		if lastSpaceEnd > rowStart {
			brk = lastSpaceEnd
		}
		if brk <= rowStart {
			// No progress would be made: force at least one rune of
			// progress to avoid an infinite loop on pathological input.
			brk = rowStart + 1
		}
		segs = append(segs, wrapSegment{rowStart, brk, string(runes[rowStart:brk])})
		pos = brk
	}
	if len(segs) == 0 {
		segs = append(segs, wrapSegment{0, len(runes), body})
	}
	return segs
}

// bookWrapLineRunes returns the number of display rows a parsed line will
// occupy when soft-wrapped to fit within availW visible columns. Code,
// images, rules, blanks, and headers are never wrapped and always return 1
// (images return their own row count elsewhere).
func bookWrapLineRunes(pl parsedLine, availW int) int {
	switch pl.kind {
	case lineKindHeader, lineKindCode, lineKindTable, lineKindBlank, lineKindRule, lineKindImage:
		return 1
	}
	pfxLen := len([]rune(pl.prefix))
	bodyAvailW := availW - pfxLen
	if bodyAvailW <= 0 {
		bodyAvailW = 1
	}
	n := len(bookWrapBody(pl.body, bodyAvailW))
	if n < 1 {
		return 1
	}
	return n
}

// bookWrapSegmentsPixel splits styled inline segments into display rows that
// fit within availPx pixels. It returns a slice of per-row segment slices
// that the caller can pass to drawSegments. Word-boundary breaking is done
// by scanning for spaces.
type wrappedRow struct {
	segs       []textSegment // styled segments for this display row
	runeOffset int           // rune offset into the original body text
	runeCount  int           // number of visual body runes on this row
	plainText  string        // set by bookWrapPlainPixel; unused by segment renderers
}

func bookWrapSegmentsPixel(fs *bookFontSet, body string, availPx int) []wrappedRow {
	if availPx <= 0 {
		availPx = 1
	}
	segs := parseLineSegments(body)
	// Flatten all segments into a single rune+face list so we can measure
	// and break across segment boundaries.
	type runeInfo struct {
		r    rune
		face font.Face
		seg  int // index into segs
		bold bool
	}
	var runes []runeInfo
	for si, seg := range segs {
		f := faceForSeg(fs, seg)
		for _, r := range seg.text {
			runes = append(runes, runeInfo{r, f, si, seg.bold})
		}
	}
	if len(runes) == 0 {
		return []wrappedRow{{segs: segs, runeOffset: 0, runeCount: 0}}
	}

	// Measure and break into rows.
	var rows []wrappedRow
	pos := 0
	for pos < len(runes) {
		rowStart := pos
		x := fixed.Int26_6(0)
		lastSpace := -1
		end := pos
		for end < len(runes) {
			adv, _ := faceGlyphAdvance(runes[end].face, runes[end].r)
			extra := fixed.Int26_6(0)
			if runes[end].bold {
				extra = fixed.I(1)
			}
			if (x+adv+extra).Round() > availPx && end > rowStart {
				break
			}
			x += adv + extra
			// Only break on the first space in a run of spaces (mirrors
			// bookWrapBody) so consecutive spaces don't all get absorbed
			// into the previous row, making an inserted space invisible.
			if runes[end].r == ' ' && (end == rowStart || runes[end-1].r != ' ') {
				lastSpace = end
			}
			end++
		}
		if end < len(runes) && lastSpace > rowStart {
			end = lastSpace + 1 // break after the first space of the run
		}
		// Build textSegments for this row from the runes slice.
		var rowSegs []textSegment
		i := rowStart
		for i < end {
			si := runes[i].seg
			var b strings.Builder
			for i < end && runes[i].seg == si {
				b.WriteRune(runes[i].r)
				i++
			}
			orig := segs[si]
			rowSegs = append(rowSegs, textSegment{
				text:      b.String(),
				bold:      orig.bold,
				italic:    orig.italic,
				underline: orig.underline,
				strike:    orig.strike,
				code:      orig.code,
			})
		}
		rows = append(rows, wrappedRow{
			segs:       rowSegs,
			runeOffset: rowStart,
			runeCount:  end - rowStart,
		})
		pos = end
	}
	return rows
}

// bookWrapLinePixel returns the number of display rows a body/list line
// will occupy when rendered with the given font set within the available
// pixel width (excluding the prefix). Non-body kinds always return 1.
func bookWrapLinePixel(fs *bookFontSet, pl parsedLine, bodyAvailPx int) int {
	switch pl.kind {
	case lineKindHeader, lineKindCode, lineKindTable, lineKindBlank, lineKindRule, lineKindImage:
		return 1
	}
	return len(bookWrapSegmentsPixel(fs, pl.body, bodyAvailPx))
}

// bookRoundedCorners paints anti-aliased dark quarter-circles in the specified
// bookRoundedCorners carves quarter-circle rounded corners into img by
// filling the outside of each corner quadrant with bg. Only the corners
// selected via top/bottom are modified. A 1-pixel anti-aliasing fringe
// keeps edges smooth. bg is typically the colour of whatever will sit
// immediately adjacent to the rounded edge (terminal chrome above the
// top corners; the status bar below the bottom corners); passing a
// mismatched bg produces a visible seam.
func bookRoundedCorners(img *image.RGBA, radius int, top, bottom bool, bg color.NRGBA) {
	if radius <= 0 || (!top && !bottom) {
		return
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	// Blend a single pixel between the background and the existing colour based
	// on how far outside the quarter-circle it lies. A 1-pixel anti-aliasing
	// fringe keeps edges smooth.
	blend := func(x, y, cx, cy int) {
		dx := float64(x) - float64(cx) + 0.5
		dy := float64(y) - float64(cy) + 0.5
		dist := math.Sqrt(dx*dx+dy*dy) - float64(radius)
		if dist >= 1.0 {
			img.Set(x, y, bg)
		} else if dist > 0 {
			// Anti-alias fringe.
			a := dist
			orig := img.RGBAAt(x, y)
			img.Set(x, y, color.NRGBA{
				R: uint8(float64(bg.R)*a + float64(orig.R)*(1-a)),
				G: uint8(float64(bg.G)*a + float64(orig.G)*(1-a)),
				B: uint8(float64(bg.B)*a + float64(orig.B)*(1-a)),
				A: 0xff,
			})
		}
	}

	if top {
		// Top-left corner (centre of circle at radius, radius).
		for y := range radius {
			for x := range radius {
				blend(x, y, radius, radius)
			}
		}
		// Top-right corner.
		for y := range radius {
			for x := w - radius; x < w; x++ {
				blend(x, y, w-radius-1, radius)
			}
		}
	}
	if bottom {
		// Bottom-left corner.
		for y := h - radius; y < h; y++ {
			for x := range radius {
				blend(x, y, radius, h-radius-1)
			}
		}
		// Bottom-right corner.
		for y := h - radius; y < h; y++ {
			for x := w - radius; x < w; x++ {
				blend(x, y, w-radius-1, h-radius-1)
			}
		}
	}
}

// bookContentImage renders the visible document lines to a white RGBA image
// with proper Markdown formatting. It does NOT draw the cursor; call
// bookOverlayCursor on the result to add the I-beam before displaying.
func (e *Editor) bookContentImage(pixW, pixH, editRows int, cellH uint) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, pixW, pixH))
	bg := e.bookBGImage()
	draw.Draw(img, img.Bounds(), bg, image.Point{}, draw.Src)

	// Bottom corners are part of the cached content layer so they scroll
	// with the document. Top corners are applied per-frame in bookPageToImage
	// so they stay fixed at the top of the window. The outside-fill colour
	// for the bottom corners must match whatever sits directly below the
	// page — the status bar in the usual case — otherwise the corners
	// produce a visible seam. bookBottomCornerBG handles both the bar-
	// visible and status-hidden cases (see its doc).
	// Rounded corners look great against a bright page on a dark terminal
	// chrome but they produce a visible "bite" on dark backgrounds where the
	// carved-out pixels contrast with the page. Skip them in dark mode.
	if !e.bookDarkMode {
		cornerRadius := max(min(pixW, pixH)/40, 6)
		bookRoundedCorners(img, cornerRadius, false, true, e.bookBottomCornerBG())
	}

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

	lineH := int(float64(cellH) * bookLineHeightMul)
	if lineH <= 0 {
		lineH = int(fontSize) + 4
	}

	marginLeft := int(float64(pixW) * bookMarginLeft)
	marginTop := int(float64(pixH) * bookMarginTop)
	marginBottom := int(float64(pixH) * bookMarginBottom)

	dark := e.bookFG()
	pageBG := e.bookBGImage()
	maxLines := min((pixH-marginTop-marginBottom)/lineH, editRows)
	rightMargin := pixW - int(float64(pixW)*bookMarginRight)

	startLine := e.pos.offsetY
	totalLines := e.Len()

	// Determine whether we start inside a fenced code block.
	inFence := e.fenceStateAtLine(startLine)
	// Determine whether we start inside an HTML comment block.
	inComment := e.inHTMLCommentAtLine(startLine)
	// codeBlockStart is flipped to true when we transition into a fenced
	// code block via a fence marker; the first subsequent code line uses it
	// to inset its gray background by a few pixels, producing a visible
	// margin below the preceding text.
	codeBlockStart := false

	codeBg := e.bookCodeBG() // light-gray (dark gray in dark mode) background for code blocks

	// Use separate document-line (docLine) and display-row (row) counters so that
	// image lines and soft-wrapped body lines can consume multiple display rows.
	docLine := startLine
	textW := rightMargin - marginLeft
	for row := 0; row < maxLines && docLine < totalLines; {
		rawLine := e.Line(LineIndex(docLine))
		rawLine = strings.ReplaceAll(rawLine, "\t", "    ")

		// Track HTML comment state: a line is hidden if it's inside a comment
		// (inComment was true at line start) OR if the line contains both <!-- and -->
		// on the same line (single-line comment). Update inComment for next iteration.
		const htmlCommentOpen = "<!--"
		const htmlCommentClose = "-->"
		wasInComment := inComment
		commentStarted := false
		commentEnded := false
		for j := 0; j < len(rawLine); {
			if strings.HasPrefix(rawLine[j:], htmlCommentOpen) {
				inComment = true
				commentStarted = true
				j += len(htmlCommentOpen)
			} else if strings.HasPrefix(rawLine[j:], htmlCommentClose) {
				inComment = false
				commentEnded = true
				j += len(htmlCommentClose)
			} else {
				j++
			}
		}

		// A line should be hidden if:
		// 1. It started inside a comment block (wasInComment), OR
		// 2. It contains the opening <!-- marker (commentStarted), OR
		// 3. It contains the closing --> marker (commentEnded)
		lineIsComment := wasInComment || commentStarted || commentEnded

		docLine++

		// Fence marker: toggle state, render as blank row.
		if isFencedCodeMarker(rawLine) {
			if !inFence {
				codeBlockStart = true
			}
			inFence = !inFence
			row++
			continue
		}

		// HTML comment line: skip without rendering or consuming display row
		if lineIsComment {
			continue
		}

		pl := parseBookLineInContext(rawLine, false)
		if inFence {
			pl = parsedLine{kind: lineKindCode, body: rawLine}
		}
		cellTop := marginTop + row*lineH

		switch pl.kind {
		case lineKindImage:
			// Coalesce consecutive image-only lines into one paragraph so
			// badges (and similar) lay out side by side, wrapping as needed.
			urls := []string{pl.body}
			for docLine < totalLines {
				nextRaw := strings.ReplaceAll(e.Line(LineIndex(docLine)), "\t", "    ")
				nextPl := parseBookLine(nextRaw)
				if nextPl.kind != lineKindImage {
					break
				}
				urls = append(urls, nextPl.body)
				docLine++
			}
			rowsUsed := e.bookDrawImageGroup(img, urls, marginLeft, rightMargin, cellTop, lineH)
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
			hSize := fs.headerSizeForLevel(pl.headerLevel)
			hAscent := faceAscent(hFace, hSize)
			hDescent := hFace.Metrics().Descent.Round()
			if hDescent <= 0 {
				hDescent = int(hSize*0.2 + 0.5)
			}
			nRows := bookHeaderRows(fs, pl.headerLevel, lineH)
			blockH := nRows * lineH
			// Center text vertically within the allocated block.
			totalH := hAscent + hDescent
			vPad := max((blockH-totalH)/2, 0)
			drawHeaderSegments(img, fs, pl.headerLevel, marginLeft, cellTop+vPad+hAscent, parseLineSegments(pl.body), dark)
			if rightMargin < pixW {
				draw.Draw(img, image.Rect(rightMargin, cellTop, pixW, cellTop+blockH), pageBG, image.Point{}, draw.Src)
			}
			row += nRows
			continue

		case lineKindUnchecked, lineKindChecked:
			checked := pl.kind == lineKindChecked
			prefixX := e.bookDrawCheckbox(img, marginLeft, cellTop, lineH, checked)
			bodyClr := dark
			if checked {
				bodyClr = color.NRGBA{0x55, 0x55, 0x55, 0xff}
			}
			bodyPx := rightMargin - prefixX
			wrows := bookWrapSegmentsPixel(fs, pl.body, bodyPx)
			for wi, wr := range wrows {
				if row+wi >= maxLines {
					break
				}
				ct := marginTop + (row+wi)*lineH
				vPad := max((lineH-ascent)/2, 0)
				if wi == 0 {
					drawSegments(img, fs, prefixX, ct+vPad+ascent, wr.segs, bodyClr)
				} else {
					drawSegments(img, fs, prefixX, ct+vPad+ascent, wr.segs, bodyClr)
				}
				// Hard-clip overflow.
				if rightMargin < pixW {
					draw.Draw(img, image.Rect(rightMargin, ct, pixW, ct+lineH), pageBG, image.Point{}, draw.Src)
				}
			}
			row += len(wrows)
			continue

		case lineKindBullet, lineKindNumbered:
			vPad := max((lineH-ascent)/2, 0)
			baseline := cellTop + vPad + ascent
			prefixX := drawString(img, fs.regular, marginLeft, baseline, pl.prefix, dark)
			bodyPx := rightMargin - prefixX
			wrows := bookWrapSegmentsPixel(fs, pl.body, bodyPx)
			for wi, wr := range wrows {
				if row+wi >= maxLines {
					break
				}
				ct := marginTop + (row+wi)*lineH
				bl := ct + vPad + ascent
				drawSegments(img, fs, prefixX, bl, wr.segs, dark)
				// Hard-clip overflow.
				if rightMargin < pixW {
					draw.Draw(img, image.Rect(rightMargin, ct, pixW, ct+lineH), pageBG, image.Point{}, draw.Src)
				}
			}
			row += len(wrows)
			continue

		case lineKindCode:
			// Transitioning into a code block: inset the top of the gray
			// background slightly so there is a small visible gap between
			// the preceding text (or the fence-marker blank row) and the
			// code block, as requested in book-mode polish.
			topPad := 0
			if codeBlockStart {
				topPad = max(lineH/6, 3)
				codeBlockStart = false
			}
			// Light-gray background spanning left-margin to right-margin.
			draw.Draw(img, image.Rect(marginLeft, cellTop+topPad, rightMargin, cellTop+lineH),
				image.NewUniform(codeBg), image.Point{}, draw.Src)
			codeAscent := faceAscent(fs.code, float64(cellH)*0.72*0.88)
			codeDescent := fs.code.Metrics().Descent.Round()
			if codeDescent <= 0 {
				codeDescent = int(float64(cellH)*0.72*0.88*0.2 + 0.5)
			}
			// Center the full glyph box (ascent+descent) within the visible
			// (possibly inset) portion of the cell so the text stays inside
			// the gray box after top-padding is applied.
			innerH := lineH - topPad
			vPad := max((innerH-codeAscent-codeDescent)/2, 0)
			drawString(img, fs.code, marginLeft+4, cellTop+topPad+vPad+codeAscent, pl.body, dark)

		case lineKindTable:
			// Walk backward and forward in the document to locate the
			// full table block so columns size consistently across rows
			blockStart := docLine - 1
			for blockStart > 0 {
				prev := e.Line(LineIndex(blockStart - 1))
				prev = strings.ReplaceAll(prev, "\t", "    ")
				if parseBookLine(prev).kind != lineKindTable {
					break
				}
				blockStart--
			}
			blockEnd := docLine // exclusive
			for blockEnd < totalLines {
				nxt := e.Line(LineIndex(blockEnd))
				nxt = strings.ReplaceAll(nxt, "\t", "    ")
				if parseBookLine(nxt).kind != lineKindTable {
					break
				}
				blockEnd++
			}
			block := make([]string, 0, blockEnd-blockStart)
			for i := blockStart; i < blockEnd; i++ {
				ln := e.Line(LineIndex(i))
				ln = strings.ReplaceAll(ln, "\t", "    ")
				block = append(block, strings.TrimSpace(ln))
			}
			avail := rightMargin - marginLeft
			aligns, widths := bookTableLayout(fs, block, avail)
			// Header row = first non-separator row in the block
			rowInBlock := (docLine - 1) - blockStart
			headerRow := -1
			for i, r := range block {
				if !bookIsTableSeparator(r) {
					headerRow = i
					break
				}
			}
			// Same per-row computation as bookPixelRowCount, so pixel
			// bookkeeping stays consistent
			subRows := bookTableRowHeight(fs, pl.body, marginLeft, rightMargin)
			if subRows == 0 {
				// Separator row: the adjacent rows already paint the
				// dividing line, so skip drawing entirely
				continue
			}
			rowPixelH := subRows * lineH
			headerBg := color.NRGBA{0x40, 0x40, 0x40, 0xff}
			headerFg := color.NRGBA{0xff, 0xff, 0xff, 0xff}
			if e.bookDarkMode {
				// On a dark page a dark-gray header bar disappears;
				// invert the contrast so headers still read as headers.
				headerBg = color.NRGBA{0xd0, 0xd0, 0xd0, 0xff}
				headerFg = color.NRGBA{0x10, 0x10, 0x10, 0xff}
			}
			bookDrawTableRow(img, fs, pl.body, block, rowInBlock, headerRow, aligns, widths,
				marginLeft, rightMargin, cellTop, lineH, rowPixelH, dark, codeBg, headerBg, headerFg)
			if rightMargin < pixW {
				draw.Draw(img, image.Rect(rightMargin, cellTop, pixW, cellTop+rowPixelH),
					pageBG, image.Point{}, draw.Src)
			}
			row += subRows
			continue

		default: // lineKindBody
			vPad := max((lineH-ascent)/2, 0)
			wrows := bookWrapSegmentsPixel(fs, pl.body, textW)
			for wi, wr := range wrows {
				if row+wi >= maxLines {
					break
				}
				ct := marginTop + (row+wi)*lineH
				bl := ct + vPad + ascent
				drawSegments(img, fs, marginLeft, bl, wr.segs, dark)
				// Hard-clip overflow.
				if rightMargin < pixW {
					draw.Draw(img, image.Rect(rightMargin, ct, pixW, ct+lineH), pageBG, image.Point{}, draw.Src)
				}
			}
			row += len(wrows)
			continue
		}

		// Hard-clip any text that overflowed past the right margin (for
		// non-wrapped line kinds that fall through: blank, rule, header, code).
		if rightMargin < pixW {
			draw.Draw(img, image.Rect(rightMargin, cellTop, pixW, cellTop+lineH), pageBG, image.Point{}, draw.Src)
		}
		row++
	}

	_ = textW // used above
	return img
}

// bookOverlayCursor draws the I-beam cursor onto dst (which should be a copy
// of the content image) at the position matching the editor's current cursor.
// Soft-wrapped body/list lines are accounted for: the cursor is placed on the
// correct sub-row within the wrapped line.
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
	lineH := int(float64(cellH) * bookLineHeightMul)
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
	textW := marginRight - marginLeft

	cursorDataY := int(e.DataY())
	cursorRawX := e.pos.sx + e.pos.offsetX
	startLine := e.pos.offsetY
	totalLines := e.Len()

	// Map cursor document line to its display row, accounting for multi-row
	// images, fenced code blocks, and soft-wrapped body/list lines.
	inFence := e.fenceStateAtLine(startLine)
	cursorDisplayRow := -1
	cursorInFence := false
	{
		dl := startLine
		for row := 0; row < maxLines && dl < totalLines; {
			rl := e.Line(LineIndex(dl))
			rl = strings.ReplaceAll(rl, "\t", "    ")
			if isFencedCodeMarker(rl) {
				inFence = !inFence
				if dl == cursorDataY {
					cursorDisplayRow = row
					cursorInFence = false
					break
				}
				row++
				dl++
				continue
			}
			pl := parseBookLine(rl)
			if inFence {
				pl = parsedLine{kind: lineKindCode, body: rl}
			}
			if pl.kind == lineKindImage {
				// Coalesce a group of image-only lines, then check whether
				// the cursor is inside it. A cursor on any member maps to
				// the top display row of the group.
				groupStart := dl
				urls := []string{pl.body}
				dl++
				for dl < totalLines {
					nextRaw := strings.ReplaceAll(e.Line(LineIndex(dl)), "\t", "    ")
					nextPl := parseBookLine(nextRaw)
					if nextPl.kind != lineKindImage {
						break
					}
					urls = append(urls, nextPl.body)
					dl++
				}
				if cursorDataY >= groupStart && cursorDataY < dl {
					cursorDisplayRow = row
					cursorInFence = false
					break
				}
				row += e.bookImageGroupRows(urls, textW, lineH)
				continue
			}
			if dl == cursorDataY {
				cursorDisplayRow = row
				cursorInFence = inFence
				break
			}
			row += bookPixelRowCount(fs, pl, lineH, marginLeft, marginRight)
			dl++
		}
	}
	if cursorDisplayRow < 0 {
		fallback := cursorDataY - startLine
		if fallback >= 0 && fallback < maxLines {
			cursorDisplayRow = fallback
		} else {
			return
		}
	}

	rawLine := e.Line(LineIndex(cursorDataY))
	rawLine = strings.ReplaceAll(rawLine, "\t", "    ")
	pl := parseBookLine(rawLine)
	if cursorInFence {
		pl = parsedLine{kind: lineKindCode, body: rawLine}
	}

	// maxBottom ensures the cursor bar never renders inside the bottom margin.
	maxBottom := pixH - marginBottom - 1

	// Helper to draw the bar at a given row and pixel X. nRows is the number
	// of display rows the line occupies (>1 for headers). The bar height is
	// clamped to nRows*lineH so it never overflows into an adjacent line's rows.
	drawBar := func(px, dispRow, vpad, asc, desc, nRows int) {
		ct := marginTop + dispRow*lineH
		top := ct + vpad
		bottom := top + asc + desc
		if rowBottom := ct + nRows*lineH; bottom > rowBottom {
			bottom = rowBottom
		}
		if bottom > maxBottom {
			bottom = maxBottom
		}
		drawCursorBar(dst, px, top, bottom)
	}

	switch pl.kind {
	case lineKindImage, lineKindBlank:
		drawBar(marginLeft, cursorDisplayRow, regVPad, ascent, regDescent, 1)

	case lineKindHeader:
		hFace := fs.headerForLevel(pl.headerLevel)
		hSize := fs.headerSizeForLevel(pl.headerLevel)
		hAscent := faceAscent(hFace, hSize)
		hDescent := hFace.Metrics().Descent.Round()
		if hDescent <= 0 {
			hDescent = int(hSize*0.2 + 0.5)
		}
		nRows := bookHeaderRows(fs, pl.headerLevel, lineH)
		blockH := nRows * lineH
		totalH := hAscent + hDescent
		hVPad := max((blockH-totalH)/2, 0)
		prefixLen := pl.headerLevel + 1
		adjRawX := max(cursorRawX-prefixLen, 0)
		bodyRunes := []rune(pl.body)
		if adjRawX > len(bodyRunes) {
			adjRawX = len(bodyRunes)
		}
		adv := measureHeaderSegmentsToRune(fs, pl.headerLevel, parseLineSegments(pl.body), adjRawX)
		drawBar(marginLeft+adv, cursorDisplayRow, hVPad, hAscent, hDescent, nRows)

	case lineKindCode:
		codeAscent := faceAscent(fs.code, float64(cellH)*0.72*0.88)
		codeDescent := fs.code.Metrics().Descent.Round()
		if codeDescent <= 0 {
			codeDescent = int(float64(cellH)*0.72*0.88*0.2 + 0.5)
		}
		codeVPad := max((lineH-codeAscent-codeDescent)/2, 0)
		codeRunes := []rune(pl.body)
		adjX := cursorRawX
		if pl.body != rawLine {
			adjX = max(cursorRawX-4, 0)
		}
		if adjX > len(codeRunes) {
			adjX = len(codeRunes)
		}
		adv := measureStringFB(fs.code, string(codeRunes[:adjX]))
		drawBar(marginLeft+4+adv.Round(), cursorDisplayRow, codeVPad, codeAscent, codeDescent, 1)

	case lineKindTable:
		// Measure using the proportional font so the cursor tracks the
		// rendered row text. Raw cursor X is clamped to the body length.
		tRunes := []rune(pl.body)
		adjX := max(min(cursorRawX, len(tRunes)), 0)
		adv := measureStringFB(fs.regular, string(tRunes[:adjX]))
		drawBar(marginLeft+adv.Round(), cursorDisplayRow, regVPad, ascent, regDescent, 1)

	case lineKindBullet, lineKindUnchecked, lineKindChecked, lineKindNumbered:
		rawPrefix := rawMarkdownPrefix(rawLine)
		rawBodyStart := len([]rune(rawPrefix))
		bodyRawX := cursorRawX - rawBodyStart
		// Compute prefix pixel width.
		var prefixEndX int
		if pl.kind == lineKindUnchecked || pl.kind == lineKindChecked {
			size := max(lineH*55/100, 5)
			prefixEndX = marginLeft + size + 3
		} else {
			prefixEndX = marginLeft + measureStringFB(fs.regular, pl.prefix).Round()
		}
		if bodyRawX < 0 {
			drawBar(prefixEndX, cursorDisplayRow, regVPad, ascent, regDescent, 1)
			break
		}
		bodyVisX := rawXToVisualX(pl.body, bodyRawX)
		// Determine which wrapped sub-row the cursor falls on.
		bodyAvailPx := marginRight - prefixEndX
		wrows := bookWrapSegmentsPixel(fs, pl.body, bodyAvailPx)
		subRow := 0
		localVisX := bodyVisX
		backward := e.bookCursorAffinity == bookAffinityBackward
		for i, wr := range wrows {
			// Backward affinity: the wrap-boundary rune belongs to this
			// sub-row (end of it), so use <=. Forward affinity: it belongs
			// to the next sub-row, so use < as before.
			if i == len(wrows)-1 || localVisX < wr.runeCount || (backward && localVisX == wr.runeCount) {
				subRow = i
				break
			}
			localVisX -= wr.runeCount
			subRow = i + 1
		}
		if subRow >= len(wrows) {
			subRow = len(wrows) - 1
		}
		cursorPx := prefixEndX + measureSegmentsToRune(fs, wrows[subRow].segs, localVisX)
		drawBar(cursorPx, cursorDisplayRow+subRow, regVPad, ascent, regDescent, 1)

	default: // lineKindBody
		visX := rawXToVisualX(rawLine, cursorRawX)
		wrows := bookWrapSegmentsPixel(fs, pl.body, textW)
		subRow := 0
		localVisX := visX
		backward := e.bookCursorAffinity == bookAffinityBackward
		for i, wr := range wrows {
			if i == len(wrows)-1 || localVisX < wr.runeCount || (backward && localVisX == wr.runeCount) {
				subRow = i
				break
			}
			localVisX -= wr.runeCount
			subRow = i + 1
		}
		if subRow >= len(wrows) {
			subRow = len(wrows) - 1
		}
		cursorPx := marginLeft + measureSegmentsToRune(fs, wrows[subRow].segs, localVisX)
		drawBar(cursorPx, cursorDisplayRow+subRow, regVPad, ascent, regDescent, 1)
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

// nextListPrefix returns the prefix to use for a new list item that follows a
// line with the given prefix. Numbered lists are incremented; checked checkboxes
// become unchecked; all other prefixes are carried through unchanged.
func nextListPrefix(pfx string) string {
	trimmed := strings.TrimLeft(pfx, " \t")
	indent := pfx[:len(pfx)-len(trimmed)]

	// Checked checkbox → unchecked
	if strings.HasPrefix(trimmed, "- [x] ") || strings.HasPrefix(trimmed, "- [X] ") {
		return indent + "- [ ] "
	}

	// Numbered list: increment the number
	for i, r := range trimmed {
		if unicode.IsDigit(r) {
			continue
		}
		if i > 0 && r == '.' && i+2 <= len(trimmed) && trimmed[i+1] == ' ' {
			numStr := trimmed[:i]
			if n, err := strconv.Atoi(numStr); err == nil {
				return indent + strconv.Itoa(n+1) + ". "
			}
		}
		break
	}

	// Bullet, unchecked checkbox — same prefix
	return pfx
}

// writeBookTextSegs writes parsed inline Markdown segments to the canvas at
// (startX, y), advancing x for each rune. maxX is the right pixel bound.
// bookTextTheme returns the foreground and background colours text book mode
// should use for a given logical fg. The background is always a bright white
// (independent of the user's terminal theme) and the fg defaults to black,
// falling back to dark gray or light gray for secondary text, to match the
// look of the graphical book mode.
// bookTextTheme returns the foreground and background colours text book mode
// should use for a given logical fg. The background is always a true white
// and the fg defaults to pure black, falling back to dark gray for
// secondary text. We map the semantic colour constants to true-colour
// values because many terminal palettes remap color 0 ("black") to a dark
// gray, which ruins legibility on a white background.
func (e *Editor) bookTextTheme(fg vt.AttributeColor, bold bool) (vt.AttributeColor, vt.AttributeColor) {
	if bold {
		// vt.AttributeColor.Combine truncates its operands to the lower
		// 16 bits, which drops the extended / true-colour flag bits and
		// turns a true-colour value into a bogus SGR sequence. For bold
		// headers we use the palette (vt.Black / vt.DarkGray) so the
		// attribute combination emits a valid "\x1b[30;1m" / "\x1b[90;1m"
		// instead of the corrupted escape that otherwise leaks SGR
		// attributes onto subsequent cells.
		return fg.Combine(vt.Bold), e.bookTextModeBG()
	}
	return e.bookTextResolveFG(fg), e.bookTextModeBG()
}

// bookTextResolveFG converts the semantic colour constants used by the
// text-mode renderer (vt.Black / vt.DarkGray / vt.White) into true-colour
// equivalents so the final SGR output stays faithful to the intended look
// regardless of the user's configured palette. vt.TrueColor downgrades
// automatically to the nearest xterm-256 or ANSI-16 colour on terminals
// that don't support 24-bit colour. Dark-mode-aware: vt.Black maps to the
// light foreground on dark backgrounds.
func (e *Editor) bookTextResolveFG(fg vt.AttributeColor) vt.AttributeColor {
	switch fg {
	case vt.Black:
		return e.bookTextModeFGBlack()
	case vt.DarkGray:
		return e.bookTextModeFGDim()
	case vt.White:
		return bookTextFGWhite
	}
	return fg
}

func (e *Editor) writeBookTextSegs(c *vt.Canvas, startX uint, y uint, segs []textSegment, maxX int) {
	x := startX
	bodyFG := e.bookTextModeFGBlack()
	codeFG := e.bookTextModeFGDim()
	textBG := e.bookTextModeBG()
	for _, seg := range segs {
		if int(x) >= maxX {
			break
		}
		// Base colours for text book mode: the theme-appropriate body
		// foreground on the theme background, with inline `code`
		// rendered in dim foreground so it's visually set apart without
		// introducing a second background colour.
		fg := bodyFG
		if seg.code {
			fg = codeFG
		}
		// Links (Markdown [text](url)) use the same palette as the
		// graphical renderer: #46a3bf unvisited, #7e46bf visited. The
		// colour alone differentiates links — we intentionally do NOT
		// combine vt.Underscore into the true-colour foreground or
		// background here, because vt.AttributeColor.Combine masks the
		// operand to the lower 16 bits and destroys the extended /
		// true-colour flag bits, emitting a bogus SGR like
		// "\x1b[65535;4m" that can leave the underline attribute stuck
		// on cells that follow. Colour-only link styling side-steps the
		// bug while remaining visibly distinct.
		isLink := seg.linkURL != ""
		if isLink {
			if bookIsLinkVisited(seg.linkURL) {
				fg = vt.TrueColor(0x7e, 0x46, 0xbf)
			} else {
				fg = vt.TrueColor(0x46, 0xa3, 0xbf)
			}
		}
		bg := textBG
		// For the same reason we can't pack attributes onto a true-colour
		// fg: Combine would drop the extended-colour flag and corrupt the
		// SGR sequence. When we need bold / italic, fall back to a
		// palette foreground (vt.Black / vt.DarkGray) which combines
		// cleanly with the attribute codes. We lose a touch of colour
		// fidelity on bold/italic runs but the alternative is broken
		// output. Underline / strikethrough are dropped altogether in
		// text mode — they can't be expressed safely on top of a
		// true-colour bg, and link colour already signals the link.
		if seg.bold || seg.italic {
			paletteFG := vt.Black
			if seg.code {
				paletteFG = vt.DarkGray
			}
			if e.bookDarkMode {
				// Dark mode: a palette "Black" foreground would be
				// invisible on a dark background. Swap to White /
				// LightGray which combines cleanly with Bold/Italic.
				paletteFG = vt.White
				if seg.code {
					paletteFG = vt.LightGray
				}
			}
			if isLink {
				// Preserve link colour even on bold/italic by using a
				// 256-colour nearest palette index — Combine on a
				// non-extended 256-colour value still drops the
				// extended flag, so keep it simple and skip the bold
				// here; colour wins over weight for link emphasis.
				paletteFG = fg
			}
			fg = paletteFG
			if seg.bold && seg.italic {
				fg = fg.Combine(vt.Bold)
				// No safe slot for italic on top of a true-colour bg;
				// bold wins. bg is left untouched.
			} else if seg.bold {
				fg = fg.Combine(vt.Bold)
			} else if seg.italic {
				fg = fg.Combine(vt.Italic)
			}
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
// Long body and list lines are soft-wrapped at word boundaries.
func (e *Editor) bookTextModeRender(c *vt.Canvas) {
	if c == nil {
		return
	}
	w := int(c.Width())
	h := int(c.Height())
	editRows := int(e.bookEditRows(uint(h))) // rows for content (excludes both top and bottom bars)
	if editRows <= 0 || w <= 0 {
		return
	}

	marginLeft := max(int(float64(w)*bookMarginLeft), 2)
	marginRight := w - max(int(float64(w)*bookTextMarginRight), 1)
	textW := marginRight - marginLeft
	// topMargin already shifts content below the top status bar: it includes
	// both the decorative top padding and the one-row filename bar.
	topMargin := int(float64(editRows)*bookMarginTop) + bookTextTopBarRows
	// yMax is the exclusive upper bound on y used by all loop guards below.
	// editRows is already the content budget (top and bottom bars excluded);
	// adding bookTextTopBarRows converts it to the absolute y just past the
	// last drawable row (i.e. the bottom status row).
	yMax := editRows + bookTextTopBarRows

	// Clear every row (including the status row) with the book theme's
	// background. Clearing the status row too is important on startup:
	// before Loop() calls InitialRedraw, neweditor.go had a chance to
	// paint regular editor text onto the canvas — if we left the status
	// row untouched, the old syntax-highlighted text would ghost through
	// behind the status bar. The status bar will repaint its own row on
	// top of this blank line anyway.
	blank := strings.Repeat(" ", w)
	clearFG := e.bookTextModeFGBlack()
	clearBG := e.bookTextModeBG()
	for row := range h {
		c.Write(0, uint(row), clearFG, clearBG, blank)
	}

	// Top book-title bar — palette blends with the book-mode theme and
	// surfaces genuinely useful information at a glance:
	//   [📖] <book title>                       words · ~N min · NN%
	// The title is the first H1 found in the document (up to the first 80
	// runes), falling back to the base filename. Word count, estimated
	// reading time, and scroll progress are right-aligned. A small book
	// glyph anchors the left so the bar feels like a header, not a label.
	e.drawBookTopBar(c, w)

	startLine := e.pos.offsetY
	totalLines := e.Len()

	// Compute the display-row range occupied by the active paragraph (the
	// one containing the cursor) for focus-mode dimming. Rows are measured
	// relative to (row + topMargin), i.e. absolute canvas y.
	var (
		focusEnabled = e.bookFocusMode
		activeDocY   LineIndex
		activeStartY LineIndex
		activeEndY   LineIndex
		activeRowLo  = -1
		activeRowHi  = -1
	)
	if focusEnabled {
		activeDocY = e.DataY()
		activeStartY = e.bookParagraphStart(activeDocY)
		activeEndY = e.bookParagraphEnd(activeDocY)
	}

	row := 0
	docLine := startLine
	for row+topMargin < yMax && docLine < totalLines {
		rawLine := e.Line(LineIndex(docLine))
		rawLine = strings.ReplaceAll(rawLine, "\t", "    ")
		pl := parseBookLine(rawLine)
		rowBefore := row
		currentDocLine := LineIndex(docLine)
		docLine++
		y := uint(row + topMargin)
		x := uint(marginLeft)

		switch pl.kind {
		case lineKindBlank:
			row++

		case lineKindRule:
			if textW > 0 {
				c.Write(x, y, e.bookTextModeFGDim(), e.bookTextModeBG(), strings.Repeat("─", textW))
			}
			row++

		case lineKindHeader:
			// H1 black, H2 dark gray, H3+ dark gray — all bold. The
			// graphical renderer similarly decreases emphasis as the
			// header level increases.
			fg := vt.Black
			if pl.headerLevel >= 2 {
				fg = vt.DarkGray
			}
			hfg, hbg := e.bookTextTheme(fg, true)
			headerText := html.UnescapeString(pl.body)
			for _, chunk := range bookWrapPlainRunes(headerText, textW) {
				if row+topMargin >= yMax {
					break
				}
				c.Write(x, uint(row+topMargin), hfg, hbg, chunk)
				row++
			}

		case lineKindCode:
			// Dark gray on bright white for code lines — the same
			// muted contrast used by the graphical renderer's code
			// background.
			cfg, cbg := e.bookTextTheme(vt.DarkGray, false)
			for _, chunk := range bookWrapPlainRunes(pl.body, textW) {
				if row+topMargin >= yMax {
					break
				}
				c.Write(x, uint(row+topMargin), cfg, cbg, chunk)
				row++
			}

		case lineKindTable:
			// In text mode we just print the raw pipe-delimited row. The
			// content is already monospace in the terminal so columns align.
			tfg, tbg := e.bookTextTheme(vt.Black, false)
			tableText := html.UnescapeString(pl.body)
			for _, chunk := range bookWrapPlainRunes(tableText, textW) {
				if row+topMargin >= yMax {
					break
				}
				c.Write(x, uint(row+topMargin), tfg, tbg, chunk)
				row++
			}

		case lineKindImage:
			ifg, ibg := e.bookTextTheme(vt.DarkGray, false)
			for _, chunk := range bookWrapPlainRunes("[image: "+pl.body+"]", textW) {
				if row+topMargin >= yMax {
					break
				}
				c.Write(x, uint(row+topMargin), ifg, ibg, chunk)
				row++
			}

		case lineKindBullet, lineKindNumbered, lineKindUnchecked, lineKindChecked:
			prefixFg, prefixBg := e.bookTextTheme(vt.Black, false)
			if pl.kind == lineKindBullet && strings.HasPrefix(pl.prefix, "│") {
				prefixFg, prefixBg = e.bookTextTheme(vt.DarkGray, false)
			}
			if pl.kind == lineKindChecked {
				prefixFg, prefixBg = e.bookTextTheme(vt.DarkGray, true)
			}
			pfxRunes := []rune(pl.prefix)
			pfxLen := len(pfxRunes)
			bodyAvailW := textW - pfxLen
			if bodyAvailW <= 0 {
				bodyAvailW = 1
			}
			wsegs := bookWrapBody(pl.body, bodyAvailW)
			for i, ws := range wsegs {
				if row+topMargin >= yMax {
					break
				}
				dy := uint(row + topMargin)
				if i == 0 {
					c.Write(x, dy, prefixFg, prefixBg, pl.prefix)
					e.writeBookTextSegs(c, x+uint(pfxLen), dy, parseLineSegments(ws.text), marginRight)
				} else {
					// Continuation line: indent to match prefix width.
					e.writeBookTextSegs(c, x+uint(pfxLen), dy, parseLineSegments(ws.text), marginRight)
				}
				row++
			}

		default: // lineKindBody
			wsegs := bookWrapBody(pl.body, textW)
			for _, ws := range wsegs {
				if row+topMargin >= yMax {
					break
				}
				dy := uint(row + topMargin)
				e.writeBookTextSegs(c, x, dy, parseLineSegments(ws.text), marginRight)
				row++
			}
		}

		// Remember the y-range of the active paragraph so the focus-mode
		// post-pass can leave it alone and dim everything else.
		if focusEnabled && currentDocLine >= activeStartY && currentDocLine <= activeEndY {
			absLo := rowBefore + topMargin
			absHi := max(row+topMargin-1, absLo)
			if activeRowLo == -1 || absLo < activeRowLo {
				activeRowLo = absLo
			}
			if absHi > activeRowHi {
				activeRowHi = absHi
			}
		}
	}

	// Focus mode: dim rows outside the active paragraph by rewriting each
	// cell's foreground to the dim palette colour. The content itself is
	// preserved; only the visual emphasis shifts away from it.
	if focusEnabled {
		dimFg := vt.DarkGray
		if e.bookDarkMode {
			dimFg = vt.TrueColor(90, 90, 95)
		} else {
			dimFg = vt.TrueColor(190, 190, 195)
		}
		for y := bookTextTopBarRows; y < yMax; y++ {
			if activeRowLo != -1 && y >= activeRowLo && y <= activeRowHi {
				continue
			}
			for x := range w {
				if r, err := c.At(uint(x), uint(y)); err == nil && r != 0 && r != ' ' {
					c.PlotColor(uint(x), uint(y), dimFg, r)
				}
			}
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
		return marginLeft + measureHeaderSegmentsToRune(fs, pl.headerLevel, parseLineSegments(pl.body), adjRawX)
	case lineKindBullet, lineKindUnchecked, lineKindChecked, lineKindNumbered:
		rawPrefix := rawMarkdownPrefix(rawLine)
		rawBodyStart := len([]rune(rawPrefix))
		bodyRawX := rawX - rawBodyStart
		segs := parseLineSegments(pl.body)
		prefixW := measureStringFB(fs.regular, pl.prefix).Round()
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
	lineH := int(float64(cellH) * bookLineHeightMul)
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

	// Selection background (mid-gray, legible on either page tone).
	var selBg color.NRGBA
	if e.bookDarkMode {
		selBg = color.NRGBA{0x44, 0x44, 0x48, 0xFF}
	} else {
		selBg = color.NRGBA{0xD0, 0xD0, 0xD0, 0xFF}
	}
	dark := e.bookFG()
	codeBg := e.bookCodeBG()
	pageBG := e.bookBGImage()

	// paintSubRow paints one wrapped sub-row with a selection stripe from
	// pxL..pxR, then re-renders the segments starting at segsStartX so
	// glyphs blend correctly with the new background.
	paintSubRow := func(subRow int, pxL, pxR int, segs []textSegment, segsStartX int, textClr color.Color) {
		ct := marginTop + subRow*lineH
		draw.Draw(dst, image.Rect(0, ct, pixW, ct+lineH),
			pageBG, image.Point{}, draw.Src)
		if pxR > pxL {
			draw.Draw(dst, image.Rect(pxL, ct, pxR, ct+lineH),
				image.NewUniform(selBg), image.Point{}, draw.Src)
		}
		vPad := max((lineH-ascent)/2, 0)
		baseline := ct + vPad + ascent
		drawSegments(dst, fs, segsStartX, baseline, segs, textClr)
	}

	clampX := func(x int) int {
		if x < 0 {
			return 0
		}
		if x > pixW {
			return pixW
		}
		return x
	}

	// wrappedSel paints selection for a body-style wrapped line.
	wrappedSel := func(lineIdx LineIndex, pl parsedLine, rowBase, nRows int, prefixEndX, rawPfxLen int, textClr color.Color) {
		body := pl.body
		bodyAvailPx := marginRight - prefixEndX
		if bodyAvailPx <= 0 {
			return
		}
		wrows := bookWrapSegmentsPixel(fs, body, bodyAvailPx)
		if len(wrows) == 0 {
			return
		}
		if nRows > len(wrows) {
			nRows = len(wrows)
		}

		bodyRunes := []rune(body)
		bodyVisTotal := rawXToVisualX(body, len(bodyRunes))

		var visStart, visEnd int
		switch {
		case lineIdx == selStartY && lineIdx == selEndY:
			s := selStartX - rawPfxLen
			en := selEndX - rawPfxLen
			if s < 0 {
				s = 0
			}
			if en < 0 {
				en = 0
			}
			visStart = rawXToVisualX(body, s)
			visEnd = rawXToVisualX(body, en)
			if visStart > visEnd {
				visStart, visEnd = visEnd, visStart
			}
		case lineIdx == selStartY:
			s := max(selStartX-rawPfxLen, 0)
			visStart = rawXToVisualX(body, s)
			visEnd = bodyVisTotal
		case lineIdx == selEndY:
			en := max(selEndX-rawPfxLen, 0)
			visStart = 0
			visEnd = rawXToVisualX(body, en)
		default:
			visStart = 0
			visEnd = bodyVisTotal
		}

		offset := 0
		for i := 0; i < len(wrows) && i < nRows; i++ {
			wr := wrows[i]
			subStart := offset
			subEnd := offset + wr.runeCount
			offset = subEnd

			lo := visStart
			hi := visEnd
			if lo < subStart {
				lo = subStart
			}
			if hi > subEnd {
				hi = subEnd
			}
			if hi <= lo {
				// No highlight on this sub-row: just re-render content.
				paintSubRow(rowBase+i, 0, 0, wr.segs, prefixEndX, textClr)
				continue
			}
			localL := lo - subStart
			localH := hi - subStart
			pxL := prefixEndX + measureSegmentsToRune(fs, wr.segs, localL)
			var pxR int
			// Extend highlight to right margin on sub-rows that are fully
			// inside a selection spanning into the next data line, or when
			// this is a non-terminal sub-row of a fully selected middle line.
			extend := false
			if lineIdx < selEndY && hi == subEnd {
				extend = true
			} else if i < len(wrows)-1 && hi == subEnd && lineIdx < selEndY {
				extend = true
			}
			if extend {
				pxR = marginRight
			} else {
				pxR = prefixEndX + measureSegmentsToRune(fs, wr.segs, localH)
			}
			paintSubRow(rowBase+i, clampX(pxL), clampX(pxR), wr.segs, prefixEndX, textClr)
		}
	}

	inFenceSel := e.fenceStateAtLine(startLine)
	codeBlockStartSel := false
	docLine2 := startLine
	for row := 0; row < maxLines && docLine2 < totalLines; {
		lineIdx := LineIndex(docLine2)
		docLine2++

		rawLine := e.Line(lineIdx)
		rawLine = strings.ReplaceAll(rawLine, "\t", "    ")

		if isFencedCodeMarker(rawLine) {
			if !inFenceSel {
				codeBlockStartSel = true
			}
			inFenceSel = !inFenceSel
			row++
			continue
		}

		pl := parseBookLine(rawLine)
		if inFenceSel {
			pl = parsedLine{kind: lineKindCode, body: rawLine}
		}

		// Image paragraphs coalesce consecutive image-only lines into one
		// visual group. Advance past the whole group in lockstep with the
		// draw loop; selection highlighting for images is a no-op.
		if pl.kind == lineKindImage {
			urls := []string{pl.body}
			for docLine2 < totalLines {
				nextRaw := strings.ReplaceAll(e.Line(LineIndex(docLine2)), "\t", "    ")
				nextPl := parseBookLine(nextRaw)
				if nextPl.kind != lineKindImage {
					break
				}
				urls = append(urls, nextPl.body)
				docLine2++
			}
			row += e.bookImageGroupRows(urls, marginRight-marginLeft, lineH)
			continue
		}

		var rowsConsumed int
		switch pl.kind {
		default:
			rowsConsumed = bookPixelRowCount(fs, pl, lineH, marginLeft, marginRight)
		}
		if rowsConsumed < 1 {
			rowsConsumed = 1
		}

		if lineIdx < selStartY || lineIdx > selEndY {
			row += rowsConsumed
			continue
		}

		cellTop := marginTop + row*lineH

		switch pl.kind {
		case lineKindBlank, lineKindRule:
			if !(lineIdx == selStartY && lineIdx == selEndY && selStartX == selEndX) {
				draw.Draw(dst, image.Rect(marginLeft, cellTop, marginRight, cellTop+lineH),
					image.NewUniform(selBg), image.Point{}, draw.Src)
				if pl.kind == lineKindRule {
					ruleY := cellTop + lineH/2
					for px := marginLeft; px < marginRight; px++ {
						dst.Set(px, ruleY, dark)
						dst.Set(px, ruleY+1, dark)
					}
				}
			}

		case lineKindHeader:
			hFace := fs.headerForLevel(pl.headerLevel)
			hSize := fs.headerSizeForLevel(pl.headerLevel)
			hAscent := faceAscent(hFace, hSize)
			hDescent := hFace.Metrics().Descent.Round()
			if hDescent <= 0 {
				hDescent = int(hSize*0.2 + 0.5)
			}
			nRows := bookHeaderRows(fs, pl.headerLevel, lineH)
			blockH := nRows * lineH
			totalH := hAscent + hDescent
			hVPad := max((blockH-totalH)/2, 0)

			prefixLen := pl.headerLevel + 1
			bodyRunes := []rune(pl.body)
			hSegs := parseLineSegments(pl.body)
			measureTo := func(n int) int {
				if n < 0 {
					n = 0
				}
				if n > len(bodyRunes) {
					n = len(bodyRunes)
				}
				return measureHeaderSegmentsToRune(fs, pl.headerLevel, hSegs, n)
			}
			var hLeft, hRight int
			switch {
			case lineIdx == selStartY && lineIdx == selEndY:
				hLeft = marginLeft + measureTo(selStartX-prefixLen)
				hRight = marginLeft + measureTo(selEndX-prefixLen)
			case lineIdx == selStartY:
				hLeft = marginLeft + measureTo(selStartX-prefixLen)
				hRight = marginRight
			case lineIdx == selEndY:
				hLeft = marginLeft
				hRight = marginLeft + measureTo(selEndX-prefixLen)
			default:
				hLeft = marginLeft
				hRight = marginRight
			}
			if hLeft > hRight {
				hLeft, hRight = hRight, hLeft
			}
			hLeft = clampX(hLeft)
			hRight = clampX(hRight)
			if hRight > hLeft {
				draw.Draw(dst, image.Rect(0, cellTop, pixW, cellTop+blockH),
					pageBG, image.Point{}, draw.Src)
				draw.Draw(dst, image.Rect(hLeft, cellTop, hRight, cellTop+blockH),
					image.NewUniform(selBg), image.Point{}, draw.Src)
				drawHeaderSegments(dst, fs, pl.headerLevel, marginLeft, cellTop+hVPad+hAscent, hSegs, dark)
			}

		case lineKindCode:
			topPad := 0
			if codeBlockStartSel {
				topPad = max(lineH/6, 3)
				codeBlockStartSel = false
			}
			codeAscent := faceAscent(fs.code, fontSize*0.88)
			codeDescent := fs.code.Metrics().Descent.Round()
			if codeDescent <= 0 {
				codeDescent = int(fontSize*0.88*0.2 + 0.5)
			}
			innerH := lineH - topPad
			codeVPad := max((innerH-codeAscent-codeDescent)/2, 0)
			codeRunes := []rune(pl.body)
			measureTo := func(n int) int {
				if n < 0 {
					n = 0
				}
				if n > len(codeRunes) {
					n = len(codeRunes)
				}
				return measureStringFB(fs.code, string(codeRunes[:n])).Round()
			}
			adj := 0
			if pl.body != rawLine {
				adj = 4
			}
			var hLeft, hRight int
			switch {
			case lineIdx == selStartY && lineIdx == selEndY:
				hLeft = marginLeft + 4 + measureTo(selStartX-adj)
				hRight = marginLeft + 4 + measureTo(selEndX-adj)
			case lineIdx == selStartY:
				hLeft = marginLeft + 4 + measureTo(selStartX-adj)
				hRight = marginRight
			case lineIdx == selEndY:
				hLeft = marginLeft + 4
				hRight = marginLeft + 4 + measureTo(selEndX-adj)
			default:
				hLeft = marginLeft
				hRight = marginRight
			}
			if hLeft > hRight {
				hLeft, hRight = hRight, hLeft
			}
			hLeft = clampX(hLeft)
			hRight = clampX(hRight)
			draw.Draw(dst, image.Rect(marginLeft, cellTop+topPad, marginRight, cellTop+lineH),
				image.NewUniform(codeBg), image.Point{}, draw.Src)
			if hRight > hLeft {
				draw.Draw(dst, image.Rect(hLeft, cellTop+topPad, hRight, cellTop+lineH),
					image.NewUniform(selBg), image.Point{}, draw.Src)
			}
			drawString(dst, fs.code, marginLeft+4, cellTop+topPad+codeVPad+codeAscent, pl.body, dark)

		case lineKindTable:
			// Separator rows take no visual space — nothing to highlight
			if rowsConsumed == 0 {
				break
			}
			// Selection inside a rendered table cell is approximated by a
			// solid highlight across the visible row (across all its
			// wrapped sub-rows) — accurate cell-level selection math
			// would require per-column layout info here.
			if lineIdx >= selStartY && lineIdx <= selEndY {
				rowH := rowsConsumed * lineH
				draw.Draw(dst, image.Rect(marginLeft, cellTop, marginRight, cellTop+rowH),
					image.NewUniform(selBg), image.Point{}, draw.Src)
			}

		case lineKindUnchecked, lineKindChecked:
			checked := pl.kind == lineKindChecked
			size := max(lineH*55/100, 5)
			prefixEndX := marginLeft + size + 3
			rawPrefix := rawMarkdownPrefix(rawLine)
			rawPfxLen := len([]rune(rawPrefix))
			textClr := dark
			if checked {
				textClr = color.NRGBA{0x55, 0x55, 0x55, 0xff}
			}
			wrappedSel(lineIdx, pl, row, rowsConsumed, prefixEndX, rawPfxLen, textClr)
			e.bookDrawCheckbox(dst, marginLeft, cellTop, lineH, checked)

		case lineKindBullet, lineKindNumbered:
			prefixPx := measureStringFB(fs.regular, pl.prefix).Round()
			prefixEndX := marginLeft + prefixPx
			rawPrefix := rawMarkdownPrefix(rawLine)
			rawPfxLen := len([]rune(rawPrefix))
			wrappedSel(lineIdx, pl, row, rowsConsumed, prefixEndX, rawPfxLen, dark)
			vPad := max((lineH-ascent)/2, 0)
			drawString(dst, fs.regular, marginLeft, cellTop+vPad+ascent, pl.prefix, dark)

		default: // lineKindBody
			wrappedSel(lineIdx, pl, row, rowsConsumed, marginLeft, 0, dark)
		}

		row += rowsConsumed
	}
}

// bookSaveScreenshot writes the current graphical-book-mode content cache
// to /tmp as a PNG. Used by the SIGUSR1 handler so the user can capture
// whatever Orbiton has most recently composited — useful for diagnosing
// rendering problems when the UI appears to hang.
// Returns the saved path or an empty string on failure.
func bookSaveScreenshot() string {
	redrawMutex.Lock()
	defer redrawMutex.Unlock()
	if bookContentCache == nil {
		return ""
	}
	// Copy so we don't hold a reference to the live cache while encoding.
	src := bookContentCache
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	path := fmt.Sprintf("/tmp/orbiton-book-%d.png", time.Now().UnixNano())
	f, err := os.Create(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	if err := png.Encode(f, dst); err != nil {
		return ""
	}
	return path
}

// bookPageToImage renders the visible page (content + selection + cursor) into
// an RGBA image. The content layer is cached and reused when dimensions and
// content have not changed; selection and cursor are always composited fresh.
func (e *Editor) bookPageToImage(pixW, pixH, editRows int, cellH uint) *image.RGBA {
	// Invalidate cache if dimensions, scroll offset, or document content changed.
	gen := bookCurrentContentGen()
	if bookContentCache == nil || bookContentCacheW != pixW || bookContentCacheH != pixH || bookContentCacheOffsetY != e.pos.offsetY || gen != bookContentCacheGen {
		bookContentCache = e.bookContentImage(pixW, pixH, editRows, cellH)
		bookContentCacheW = pixW
		bookContentCacheH = pixH
		bookContentCacheOffsetY = e.pos.offsetY
		bookContentCacheGen = gen
	}

	// Copy the content image so overlays don't pollute the cache.
	// Reuse a single destination buffer across frames to avoid a fresh
	// multi-MB allocation (and matching GC work) on every keystroke.
	bounds := bookContentCache.Bounds()
	if bookPageImageBuf == nil || bookPageImageBuf.Bounds() != bounds {
		bookPageImageBuf = image.NewRGBA(bounds)
	}
	dst := bookPageImageBuf
	copy(dst.Pix, bookContentCache.Pix)

	e.bookOverlaySelection(dst, pixW, pixH, editRows, cellH)
	if e.bookFocusMode {
		e.bookOverlayFocusDim(dst, pixW, pixH, editRows, cellH)
	}
	e.bookOverlayCursor(dst, pixW, pixH, editRows, cellH)

	// Top corners: decorative carve on bright pages, skipped in dark mode
	// where the carved pixels would otherwise contrast as dark "teeth" at
	// the top edge of the page.
	if !e.bookDarkMode {
		cornerRadius := max(min(pixW, pixH)/40, 6)
		bookRoundedCorners(dst, cornerRadius, true, false, color.NRGBA{0x00, 0x00, 0x00, 0xff})
	}

	return dst
}

// bookOverlayFocusDim dims pixel rows outside the active paragraph by blending
// them toward the book background. Called only when bookFocusMode is true.
// The content image itself is unchanged, so toggling focus mode off restores
// the original appearance on the next render.
func (e *Editor) bookOverlayFocusDim(dst *image.RGBA, pixW, pixH, editRows int, cellH uint) {
	if dst == nil || pixW <= 0 || pixH <= 0 || editRows <= 0 {
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
	lineH := int(float64(cellH) * bookLineHeightMul)
	if lineH <= 0 {
		lineH = int(fontSize) + 4
	}

	marginLeft := int(float64(pixW) * bookMarginLeft)
	marginTop := int(float64(pixH) * bookMarginTop)
	marginBottom := int(float64(pixH) * bookMarginBottom)
	rightMargin := pixW - int(float64(pixW)*bookMarginRight)
	maxLines := min((pixH-marginTop-marginBottom)/lineH, editRows)

	cursorDataY := e.DataY()
	activeStart := e.bookParagraphStart(cursorDataY)
	activeEnd := e.bookParagraphEnd(cursorDataY)

	startLine := e.pos.offsetY
	totalLines := e.Len()

	// Walk docLines from offsetY, measuring pixel height of each, to find
	// where the active paragraph falls in the content image.
	pxTop := marginTop
	activePxLo, activePxHi := -1, -1
	row := 0
	dl := LineIndex(startLine)
	for row < maxLines && int(dl) < totalLines {
		rl := e.Line(dl)
		rl = strings.ReplaceAll(rl, "\t", "    ")
		pl := parseBookLine(rl)
		rows := bookPixelRowCount(fs, pl, lineH, marginLeft, rightMargin)
		pxBot := pxTop + rows*lineH
		if dl >= activeStart && dl <= activeEnd {
			if activePxLo == -1 || pxTop < activePxLo {
				activePxLo = pxTop
			}
			if pxBot > activePxHi {
				activePxHi = pxBot
			}
		}
		pxTop = pxBot
		row += rows
		dl++
	}

	// If the active paragraph is off-screen (shouldn't happen because
	// ensureCursorVisible has already scrolled it in), just dim everything.
	bgR, bgG, bgB, _ := e.bookBG().RGBA()
	bgr8, bgg8, bgb8 := uint8(bgR>>8), uint8(bgG>>8), uint8(bgB>>8)
	// Alpha of 140/255 ≈ 55% toward bg keeps the dim text legible as context
	// but clearly secondary. In dark mode, bump to ~75% for stronger dimming,
	// since the text is already naturally dim and needs more contrast.
	dimAlpha := 140
	if e.bookDarkMode {
		dimAlpha = 190
	}

	dimRow := func(y int) {
		if y < 0 || y >= pixH {
			return
		}
		// stride = 4 bytes per pixel in RGBA
		base := y * dst.Stride
		for x := range pixW {
			i := base + x*4
			r := dst.Pix[i]
			g := dst.Pix[i+1]
			b := dst.Pix[i+2]
			// blend toward bg
			r = uint8((int(r)*(255-dimAlpha) + int(bgr8)*dimAlpha) / 255)
			g = uint8((int(g)*(255-dimAlpha) + int(bgg8)*dimAlpha) / 255)
			b = uint8((int(b)*(255-dimAlpha) + int(bgb8)*dimAlpha) / 255)
			dst.Pix[i] = r
			dst.Pix[i+1] = g
			dst.Pix[i+2] = b
		}
	}

	for y := range pixH {
		if activePxLo != -1 && y >= activePxLo && y < activePxHi {
			continue
		}
		dimRow(y)
	}
}

// bookStatusBarImage renders the status bar strip as a standalone RGBA image.
// Line/col anchor left, words/progress anchor right, and numeric padding is
// fixed-width so the bar doesn't reflow as the cursor moves. A pending message
// replaces the stats half and is shown centered.
func (e *Editor) bookStatusBarImage(pixW, statusPixH int, renderH uint) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, pixW, statusPixH))

	// Status bar colour scheme is intentionally the *inverse* of the page:
	// a light page gets a black bar with white text; a dark page gets a
	// white bar with black text. The high contrast frames the reading
	// area regardless of mode. The page's bottom rounded corners are
	// carved with the same bg (see bookBottomCornerBG) so the join is
	// seamless.
	var bgClr, textClr, dimClr color.NRGBA
	if e.bookDarkMode {
		// Dark mode (dark page): white bar, black text.
		bgClr = color.NRGBA{0xff, 0xff, 0xff, 0xff}
		textClr = color.NRGBA{0x00, 0x00, 0x00, 0xff}
		dimClr = color.NRGBA{0x60, 0x60, 0x60, 0xff}
	} else {
		// Light mode (white page): black bar, white text.
		bgClr = color.NRGBA{0x00, 0x00, 0x00, 0xff}
		textClr = color.NRGBA{0xff, 0xff, 0xff, 0xff}
		dimClr = color.NRGBA{0xa0, 0xa0, 0xa0, 0xff}
	}

	draw.Draw(img, img.Bounds(), image.NewUniform(bgClr), image.Point{}, draw.Src)

	// No separator line at the top edge: the previous subtle 1 px rule
	// made sense when the bar colour sat next to a near-identical page
	// colour, but against the new high-contrast bar (inverse of the
	// page) the rule reads as a seam between the page's rounded corners
	// and the status strip.

	mainFontSize := float64(renderH) * 0.72
	if mainFontSize < 6 {
		mainFontSize = 6
	}
	fs, err := bookFaces(mainFontSize)
	if err != nil {
		return img
	}

	sbAscent := faceAscent(fs.statusBar, fs.statusBarSize)
	baseline := (statusPixH+sbAscent)/2 + 1

	// A centered message takes over the whole bar.
	if msg := bookModeGetStatusMsg(); msg != "" {
		text := strings.TrimSpace(msg)
		d := &font.Drawer{Face: fs.statusBar}
		w := d.MeasureString(text).Round()
		x := max((pixW-w)/2, 8)
		drawString(img, fs.statusBar, x, baseline, text, textClr)
		return img
	}

	// No status bar content when stats are hidden and no message is pending.
	if e.statusMode {
		return img
	}

	_, lineNumber, lastLineNumber := e.PLA()
	// Reading progress: 0% at the very top, 100% at the very bottom.
	// Using the raw line / lastLine ratio would show 16% on line 1 of 6.
	percentage := bookReadingPercent(lineNumber, lastLineNumber)

	// Left half: cursor position — fixed-width so digits don't shift the
	// surrounding labels as the cursor moves. The widths (5 / 4) are chosen
	// to cover documents up to 99 999 lines and 9999 columns comfortably;
	// longer values still render correctly, they just push the separator.
	leftText := fmt.Sprintf("Line %5d of %-5d   Col %4d",
		int(lineNumber), int(lastLineNumber), e.ColNumber())

	// Right half: enhanced document stats with more useful information
	words := e.WordCount()
	// Estimate reading time: ~200 words per minute
	readingTimeMinutes := words / 200
	var rightText string
	if readingTimeMinutes > 0 {
		rightText = fmt.Sprintf("Words %6d   ~%d min   %3d%%", words, readingTimeMinutes, percentage)
	} else {
		rightText = fmt.Sprintf("Words %6d   %3d%%", words, percentage)
	}

	// Draw left, left-anchored with a small padding.
	marginX := 16
	drawString(img, fs.statusBar, marginX, baseline, leftText, textClr)

	// Draw right, right-anchored.
	dRight := &font.Drawer{Face: fs.statusBar}
	rightW := dRight.MeasureString(rightText).Round()
	drawString(img, fs.statusBar, pixW-marginX-rightW, baseline, rightText, textClr)

	// Draw a small center crumb: the basename of the current file, dimmed,
	// so the user always has a reminder of what they're editing. We only
	// show it when there's room between the left and right halves.
	dLeft := &font.Drawer{Face: fs.statusBar}
	leftW := dLeft.MeasureString(leftText).Round()
	leftEnd := marginX + leftW
	rightStart := pixW - marginX - rightW
	if rightStart-leftEnd > 80 && e.filename != "" {
		base := e.filename
		if i := strings.LastIndexAny(base, "/\\"); i >= 0 {
			base = base[i+1:]
		}
		dCenter := &font.Drawer{Face: fs.statusBar}
		cw := dCenter.MeasureString(base).Round()
		cx := (pixW - cw) / 2
		if cx-leftEnd >= 20 && rightStart-(cx+cw) >= 20 {
			drawString(img, fs.statusBar, cx, baseline, base, dimClr)
		}
	}

	return img
}

// bookComposeFullPage builds a single RGBA image covering the full terminal:
// the rounded page (content + selection + cursor) on top, and the status bar
// strip at the bottom unless it is hidden with no pending message.
func (e *Editor) bookComposeFullPage(pixW, rowsTotal, editRows int, renderH uint) *image.RGBA {
	editPixH := editRows * int(renderH)
	pageImg := e.bookPageToImage(pixW, editPixH, editRows, renderH)

	// If the page already covers the full terminal (status bar hidden and no
	// pending message), just return the page image unchanged.
	showStatus := !e.statusMode || bookModeGetStatusMsg() != ""
	if !showStatus || rowsTotal <= editRows {
		return pageImg
	}

	totalPixH := rowsTotal * int(renderH)
	bounds := image.Rect(0, 0, pixW, totalPixH)
	if bookComposeBuf == nil || bookComposeBuf.Bounds() != bounds {
		bookComposeBuf = image.NewRGBA(bounds)
	}
	full := bookComposeBuf
	// Fill the entire buffer with black so any undrawn gap matches the
	// terminal chrome background (which is typically black). This also
	// ensures the page + status bar composite has a seamless background.
	// Clearing via a direct slice fill is cheaper than draw.Draw with a
	// uniform source.
	for i := 0; i < len(full.Pix); i += 4 {
		full.Pix[i] = 0      // R
		full.Pix[i+1] = 0    // G
		full.Pix[i+2] = 0    // B
		full.Pix[i+3] = 0xff // A
	}
	// Copy the page (content + selection + cursor + top rounded corners) to the top.
	draw.Draw(full, image.Rect(0, 0, pixW, editPixH), pageImg, image.Point{}, draw.Src)
	// Paint the status strip(s) at the bottom (typically one row).
	statusPixH := totalPixH - editPixH
	status := e.bookStatusBarImage(pixW, statusPixH, renderH)
	draw.Draw(full, image.Rect(0, editPixH, pixW, totalPixH), status, image.Point{}, draw.Src)
	return full
}

// countDisplayRowsTo returns the display-row index at which targetDoc appears
// when starting at startDoc, or -1 if it is not reached within maxRows.
// Accounts for multi-row images and soft-wrapped body/list lines. When fs is
// non-nil, pixel-based wrapping is used; otherwise rune-based wrapping.
func (e *Editor) countDisplayRowsTo(startDoc, targetDoc, maxRows, lineH, textW, marginLeft, marginRight int, fs *bookFontSet) int {
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
		switch {
		case pl.kind == lineKindImage:
			row += e.bookImageRows(pl.body, lineH, textW)
		default:
			if fs != nil {
				row += bookPixelRowCount(fs, pl, lineH, marginLeft, marginRight)
			} else {
				row += bookWrapRowCount(pl, textW)
			}
		}
		dl++
	}
	return -1
}

// bookWrapRowCount returns the number of display rows a parsed line occupies
// when soft-wrapped to fit within availW characters. Non-body line kinds
// return 1. Rune-based; use bookPixelRowCount for the graphical renderer.
func bookWrapRowCount(pl parsedLine, availW int) int {
	switch pl.kind {
	case lineKindBlank, lineKindRule:
		return 1
	case lineKindHeader, lineKindCode, lineKindTable:
		// Text book mode wraps these via bookWrapPlainRunes, so the
		// row count tracks the chunk count.
		return len(bookWrapPlainRunes(pl.body, availW))
	case lineKindImage:
		return len(bookWrapPlainRunes("[image: "+pl.body+"]", availW))
	}
	pfxLen := len([]rune(pl.prefix))
	bodyLen := len([]rune(pl.body))
	firstW := availW - pfxLen
	if firstW <= 0 {
		firstW = 1
	}
	if bodyLen <= firstW {
		return 1
	}
	rows := 1
	remaining := bodyLen - firstW
	contW := availW - pfxLen
	if contW <= 0 {
		contW = 1
	}
	rows += (remaining + contW - 1) / contW
	return rows
}

// bookHeaderRows returns the number of display rows a header line occupies
// based on its actual font metrics. Large headers need multiple rows to avoid
// overflowing into adjacent line slots.
func bookHeaderRows(fs *bookFontSet, level, lineH int) int {
	if lineH < 1 {
		lineH = 1
	}
	hFace := fs.headerForLevel(level)
	hSize := fs.headerSizeForLevel(level)
	hAscent := faceAscent(hFace, hSize)
	hDescent := hFace.Metrics().Descent.Round()
	if hDescent <= 0 {
		hDescent = int(hSize*0.2 + 0.5)
	}
	totalH := hAscent + hDescent
	return max((totalH+lineH-1)/lineH, 1)
}

// bookPixelRowCount returns the number of display rows a parsed line occupies
// when soft-wrapped using pixel-based measurement with proportional fonts.
// The cursor overlay and scroll logic rely on this matching bookContentImage.
func bookPixelRowCount(fs *bookFontSet, pl parsedLine, lineH, marginLeft, rightMargin int) int {
	switch pl.kind {
	case lineKindHeader:
		return bookHeaderRows(fs, pl.headerLevel, lineH)
	case lineKindTable:
		return bookTableRowHeight(fs, pl.body, marginLeft, rightMargin)
	case lineKindCode, lineKindBlank, lineKindRule, lineKindImage:
		return 1
	}
	textW := rightMargin - marginLeft
	switch pl.kind {
	case lineKindUnchecked, lineKindChecked:
		size := max(lineH*55/100, 5)
		prefixEndX := marginLeft + size + 3
		bodyPx := rightMargin - prefixEndX
		if bodyPx <= 0 {
			return 1
		}
		return len(bookWrapSegmentsPixel(fs, pl.body, bodyPx))
	case lineKindBullet, lineKindNumbered:
		prefixPx := measureStringFB(fs.regular, pl.prefix).Round()
		prefixEndX := marginLeft + prefixPx
		bodyPx := rightMargin - prefixEndX
		if bodyPx <= 0 {
			return 1
		}
		return len(bookWrapSegmentsPixel(fs, pl.body, bodyPx))
	default: // lineKindBody
		if textW <= 0 {
			return 1
		}
		return len(bookWrapSegmentsPixel(fs, pl.body, textW))
	}
}

// bookModeEnsureCursorVisible adjusts e.pos.offsetY (and e.pos.sy) so that
// the cursor document line is always rendered inside the visible image area.
// It accounts for multi-row image lines that consume more than one display row.
func (e *Editor) bookModeEnsureCursorVisible(c *vt.Canvas) {
	rows := int(c.Height())
	editRows := int(e.bookEditRows(uint(rows)))
	if editRows <= 0 {
		return
	}

	// Two code paths share this helper:
	//  - graphical book mode: sub-row counting happens in pixels via
	//    bookPixelRowCount, so we build a font set and pass pixel widths.
	//  - text book mode: sub-row counting happens in character cells via
	//    bookWrapRowCount, so we pass the cell-based wrap width and leave
	//    fs nil. The pixel branch used to be the only branch, which made
	//    scrolling in text mode spuriously believe the cursor was always
	//    "visible" (because bookWrapRowCount returned 1 for pixel-wide
	//    lines that actually soft-wrap across many cell rows) and the
	//    viewport never advanced when the cursor walked off the bottom.
	graphical := e.bookGraphicalMode()

	var (
		lineH          int
		textW          int
		marginLeft     int
		marginRight    int
		maxDisplayRows int
		fs             *bookFontSet
	)

	if graphical {
		cellW, cellH := imagepreview.TerminalCellPixels()
		if cellH == 0 {
			cellH = 16
		}
		if cellW == 0 {
			cellW = 8
		}
		_, renderH := bookRenderCellSize(cellW, cellH)
		lineH = int(renderH)
		pixH := editRows * lineH
		pixW := int(uint(c.Width()) * cellW)

		marginLeft = int(float64(pixW) * bookMarginLeft)
		marginRight = pixW - int(float64(pixW)*bookMarginRight)
		marginTop := int(float64(pixH) * bookMarginTop)
		marginBottom := int(float64(pixH) * bookMarginBottom)
		textW = marginRight - marginLeft

		maxDisplayRows = min((pixH-marginTop-marginBottom)/lineH, editRows)

		fontSize := float64(renderH) * 0.72
		if fontSize < 6 {
			fontSize = 6
		}
		if f, err := bookFaces(fontSize); err == nil {
			fs = f
		}
	} else {
		// Text mode uses cells for everything. Mirror the accounting done
		// by bookTextModeRender: top padding consumes some editRows, then
		// content fills the remainder. lineH/marginLeft/marginRight are
		// unused when fs == nil, but bookWrapRowCount needs textW in cells.
		lineH = 1
		textW = e.bookWrapWidth(c)
		if textW <= 0 {
			return
		}
		topMargin := int(float64(editRows) * bookMarginTop)
		maxDisplayRows = editRows - topMargin
		if maxDisplayRows <= 0 {
			maxDisplayRows = editRows
		}
	}
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
		if cursorDataY < 0 {
			return
		}
		// Fix the editor position so bookOverlayCursor finds the cursor.
		e.pos.sy = cursorDataY - e.pos.offsetY
		if e.pos.sy < 0 {
			e.pos.offsetY = cursorDataY
			e.pos.sy = 0
		}
	}
	if cursorDataY < 0 || totalLines == 0 {
		return
	}

	// Focus (typewriter) mode: anchor the cursor row near the vertical
	// centre of the edit area. Scroll offsetY so that the number of
	// display rows from offsetY to cursorDataY equals roughly
	// maxDisplayRows/2 — the page slides under a stationary cursor.
	if e.bookFocusMode {
		targetRow := max(maxDisplayRows/2, 1)
		// Walk offsetY down until the rows-from-offset matches target,
		// or up until we can't go any further (cursor near top of doc).
		// Use countDisplayRowsTo to stay honest about soft-wrapped rows.
		e.pos.offsetY = cursorDataY
		for e.pos.offsetY > 0 {
			candidate := e.pos.offsetY - 1
			rowsFromCandidate := e.countDisplayRowsTo(candidate, cursorDataY, maxDisplayRows, lineH, textW, marginLeft, marginRight, fs)
			if rowsFromCandidate < 0 || rowsFromCandidate > targetRow {
				break
			}
			e.pos.offsetY = candidate
		}
		e.pos.sy = max(cursorDataY-e.pos.offsetY, 0)
		return
	}

	// If cursor is above the scroll start, snap the view up to show it.
	if cursorDataY < e.pos.offsetY {
		e.pos.offsetY = cursorDataY
		e.pos.sy = 0
		return
	}

	// Check whether the cursor is already within the visible rows.
	if e.countDisplayRowsTo(e.pos.offsetY, cursorDataY, maxDisplayRows, lineH, textW, marginLeft, marginRight, fs) >= 0 {
		return
	}

	// Cursor is below the visible area — advance offsetY one document line at
	// a time until the cursor fits inside maxDisplayRows.
	for e.pos.offsetY < cursorDataY && e.pos.offsetY < totalLines-1 {
		e.pos.offsetY++
		if e.countDisplayRowsTo(e.pos.offsetY, cursorDataY, maxDisplayRows, lineH, textW, marginLeft, marginRight, fs) >= 0 {
			break
		}
	}

	// Keep sy consistent with the new offset.
	e.pos.sy = max(cursorDataY-e.pos.offsetY, 0)
}

func flushImageToTerminal(img image.Image, dispCols, dispRows uint) string {
	if imagepreview.IsSixel {
		// DECSDM (CSI ? 80 h) keeps sixel output from scrolling the terminal.
		// "\033[2J\033[H" clears any stale image remnants before the new frame.
		fmt.Fprintf(os.Stdout, "\033[?80h\033[2J\033[H")
		imagepreview.FlushSixelImage(os.Stdout, img)
		// Home the cursor again in case the terminal ignored DECSDM.
		fmt.Fprintf(os.Stdout, "\033[H")
		return "" // no cached encoded form for Sixel
	}
	bookPNGBuf.Reset()
	// BestSpeed keeps per-frame PNG encoding fast; Kitty decodes to RGBA
	// internally, so a larger base64 payload is cheap compared to deflate time.
	if err := bookPNGEncoder.Encode(&bookPNGBuf, img); err != nil {
		return ""
	}
	encoded := base64.StdEncoding.EncodeToString(bookPNGBuf.Bytes())
	// Buffer the whole frame (cursor-home + kitty graphics payload) into a
	// single Write so the terminal receives it atomically.
	bookWriteBuf.Reset()
	fmt.Fprintf(&bookWriteBuf, "\033[H")
	imagepreview.FlushImageWithID(&bookWriteBuf, encoded, dispCols, dispRows, 1)
	os.Stdout.Write(bookWriteBuf.Bytes())
	// Stable image ID (1) lets Kitty replace the image in place; iTerm2 ignores it.
	return encoded
}

// bookRenderCellSize returns the pixel (width, height) to use when building
// the page image. The page is rendered at the terminal's reported cell size
// so glyphs are pixel-perfect without terminal-side upscaling.
func bookRenderCellSize(cellW, cellH uint) (uint, uint) {
	return cellW, cellH
}

// bookModeRenderAll performs a full synchronized render of book mode:
// content image + status bar + cursor positioning, all wrapped in a DEC 2026
// synchronized update. Takes a cursor-only fast path when the content cache
// is valid and a Sixel strip fast path when the terminal supports Sixel.
//
// NOTE: Callers MUST hold redrawMutex. The underlying opentype.Face shares
// an internal mask buffer between Glyph() calls; without serialization the
// status-bar auto-clear goroutine can re-slice the mask under image/draw
// and panic with "index out of range" from mask.Pix.
func (e *Editor) bookModeRenderAll(c *vt.Canvas, status *StatusBar) {
	cols := uint(c.Width())
	rows := uint(c.Height())
	if rows < 2 {
		return
	}
	editRows := e.bookEditRows(rows)

	cellW, cellH := imagepreview.TerminalCellPixels()
	if cellW == 0 {
		cellW = 8
	}
	if cellH == 0 {
		cellH = 16
	}
	renderW, renderH := bookRenderCellSize(cellW, cellH)
	pixW := int(cols * renderW)
	pixH := int(editRows * renderH)

	// Consume messageAfterRedraw before deciding fast vs full path.
	hasPendingMsg := false
	if status != nil {
		if msg := status.messageAfterRedraw; len(msg) > 0 {
			status.SetMessage(msg)
			status.messageAfterRedraw = ""
			bookModeSetStatusMsg(msg)
			hasPendingMsg = true
			mut.RLock()
			dur := status.show
			isErr := status.isError
			mut.RUnlock()
			if isErr {
				dur *= 3
			}
			// Coalesce auto-clear goroutines via bookStatusClearGen so a burst
			// of messageAfterRedraw bumps doesn't pile up stale goroutines.
			myGen := bookStatusClearGen.Add(1)
			go func() {
				time.Sleep(dur)
				if bookStatusClearGen.Load() != myGen {
					return
				}
				bookModeSetStatusMsg("")
				mut.Lock()
				status.msg = ""
				status.isError = false
				mut.Unlock()
				redrawMutex.Lock()
				e.bookModeFullFrame(c)
				redrawMutex.Unlock()
			}()
		}
	}

	// Fast path: if the content cache is valid (no scroll, no edits) and no
	// pending message requires a status-bar re-render, skip re-encoding the
	// full page image. Just reposition the terminal cursor and update the
	// (small) status-bar image — both are very cheap compared to PNG encoding.
	gen := bookCurrentContentGen()
	contentCacheValid := bookContentCache != nil &&
		bookContentCacheW == pixW &&
		bookContentCacheH == pixH &&
		bookContentCacheOffsetY == e.pos.offsetY &&
		gen == bookContentCacheGen

	if contentCacheValid && !hasPendingMsg {
		e.bookModeFullFrame(c)
		return
	}

	// Full path: content changed or scrolled — re-encode and send the page image.
	vt.BeginSyncUpdate()
	if imagepreview.IsSixel {
		fmt.Fprintf(os.Stdout, "\033[1;%dr", rows)
	}
	// Stable image ID (see flushImageToTerminal) makes Kitty replace the image
	// in place, so no DeleteInlineImages() call is needed here. Calling delete
	// every frame was the main source of the "image jumps / flashes" flicker.
	e.bookModeRenderImageAt(cols, rows, editRows, renderH, pixW, pixH)
	e.bookModeShowCursor(c)
	if imagepreview.IsSixel {
		fmt.Fprintf(os.Stdout, "\033[r")
	}
	vt.EndSyncUpdate()
}

// bookModeFullFrame re-composes and flushes the whole page image.
// The caller must hold redrawMutex.
func (e *Editor) bookModeFullFrame(c *vt.Canvas) {
	cols := uint(c.Width())
	rows := uint(c.Height())
	if rows < 2 {
		return
	}
	editRows := e.bookEditRows(rows)

	cellW, cellH := imagepreview.TerminalCellPixels()
	if cellW == 0 {
		cellW = 8
	}
	if cellH == 0 {
		cellH = 16
	}
	renderW, renderH := bookRenderCellSize(cellW, cellH)
	pixW := int(cols * renderW)

	vt.BeginSyncUpdate()
	if imagepreview.IsSixel {
		// Lock scroll region so Sixel output can't push past the bottom.
		fmt.Fprintf(os.Stdout, "\033[1;%dr", rows)
	}
	img := e.bookComposeFullPage(pixW, int(rows), int(editRows), renderH)
	encoded := flushImageToTerminal(img, cols, rows)
	bookPageEncoded = encoded
	bookPageEncodedCols = cols
	bookPageEncodedRows = rows
	e.bookModeShowCursor(c)
	if imagepreview.IsSixel {
		// Reset scroll region.
		fmt.Fprintf(os.Stdout, "\033[r")
	}
	vt.EndSyncUpdate()
}

// bookModeRenderImageAt renders the unified page image and flushes it to the
// terminal, caching the encoded PNG for reuse.
func (e *Editor) bookModeRenderImageAt(cols, rowsTotal, editRows, renderH uint, pixW, pixH int) {
	img := e.bookComposeFullPage(pixW, int(rowsTotal), int(editRows), renderH)
	encoded := flushImageToTerminal(img, cols, rowsTotal)
	bookPageEncoded = encoded
	bookPageEncodedCols = cols
	bookPageEncodedRows = rowsTotal
}

// bookModeStatusBar renders the bottom status row as a small image sent via
// the Kitty/iTerm2 graphics protocol, using the Montserrat Bold sans-serif face.
// When called outside of a bookModeRenderAll cycle, the caller should wrap the
// call in vt.BeginSyncUpdate / vt.EndSyncUpdate to avoid flicker on Kitty.
func (e *Editor) bookModeStatusBar(c *vt.Canvas) {
	// When the status bar is hidden, don't render anything — the content
	// image already fills the full terminal height.
	if e.statusMode && bookModeGetStatusMsg() == "" {
		return
	}

	cols := uint(c.Width())
	rows := uint(c.Height())

	cellW, cellH := imagepreview.TerminalCellPixels()
	if cellW == 0 {
		cellW = 8
	}
	if cellH == 0 {
		cellH = 16
	}
	renderW, renderH := bookRenderCellSize(cellW, cellH)

	pixW := int(cols * renderW)
	pixH := int(renderH) // one terminal row

	// Render the status bar image. bookStatusBarImage paints its own dark
	// background, separator line and text; sharing this code path with the
	// unified page compositor keeps the layout consistent.
	img := e.bookStatusBarImage(pixW, pixH, renderH)

	// Encode and send at the last terminal row.
	fmt.Fprintf(os.Stdout, "\033[%d;1H", rows)
	if imagepreview.IsSixel {
		// DECSDM ON: don't let the sixel output move the cursor and
		// scroll the terminal (see flushImageToTerminal for details).
		fmt.Fprintf(os.Stdout, "\033[?80h")
		imagepreview.FlushSixelImage(os.Stdout, img)
	} else {
		var buf bytes.Buffer
		if png.Encode(&buf, img) != nil {
			return
		}
		encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
		imagepreview.FlushImage(os.Stdout, encoded, cols, 1)
	}
}

// bookModeShowCursor positions the hidden terminal cursor at the current
// editing position. The visual I-beam is drawn directly in the image, so
// this only needs to keep the cursor at a sane position for OS-level
// accessibility and clipboard operations.
func (e *Editor) bookModeShowCursor(c *vt.Canvas) {
	cols := uint(c.Width())
	rows := uint(c.Height())
	editRows := e.bookEditRows(rows)

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

// bookTextModePlaceCursor positions the terminal cursor at the correct
// column and row in text book mode, accounting for the left margin and
// any Markdown prefix that is hidden from the editing position.
func (e *Editor) bookTextModePlaceCursor(c *vt.Canvas) {
	w := int(c.Width())
	marginLeft := max(int(float64(w)*bookMarginLeft), 2)
	marginRight := w - max(int(float64(w)*bookTextMarginRight), 1)
	textW := marginRight - marginLeft

	rawLine := e.Line(e.DataY())
	rawLine = strings.ReplaceAll(rawLine, "\t", "    ")
	cursorRawX := e.pos.sx + e.pos.offsetX

	pl := parseBookLine(rawLine)

	// Count display rows from offsetY to DataY, accounting for soft wrapping.
	startLine := e.pos.offsetY
	cursorDataY := int(e.DataY())
	totalLines := e.Len()
	displayRow := 0
	for dl := startLine; dl < cursorDataY && dl < totalLines; dl++ {
		rl := e.Line(LineIndex(dl))
		rl = strings.ReplaceAll(rl, "\t", "    ")
		dpl := parseBookLine(rl)
		displayRow += bookWrapRowCount(dpl, textW)
	}

	// Determine the sub-row within the current (possibly wrapped) line.
	subRow := 0
	var x int
	switch pl.kind {
	case lineKindHeader:
		prefixLen := pl.headerLevel + 1
		adjX := max(cursorRawX-prefixLen, 0)
		if textW > 0 && adjX >= textW {
			subRow = adjX / textW
			adjX -= subRow * textW
		}
		x = adjX + marginLeft

	case lineKindBullet, lineKindNumbered, lineKindUnchecked, lineKindChecked:
		pfxRunes := []rune(pl.prefix)
		pfxLen := len(pfxRunes)
		bodyAvailW := textW - pfxLen
		if bodyAvailW <= 0 {
			bodyAvailW = 1
		}
		rawPrefix := rawMarkdownPrefix(rawLine)
		rawBodyStart := len([]rune(rawPrefix))
		bodyRawX := cursorRawX - rawBodyStart
		if bodyRawX < 0 {
			x = marginLeft + cursorRawX
		} else {
			// Find the wrap segment containing bodyRawX, then convert the
			// raw offset within that segment to a visual offset so markers
			// stripped by parseLineSegments don't shift the cursor.
			wsegs := bookWrapBody(pl.body, bodyAvailW)
			for i, ws := range wsegs {
				if bodyRawX < ws.end || i == len(wsegs)-1 {
					subRow = i
					localRawX := max(bodyRawX-ws.start, 0)
					localVisX := rawXToVisualX(ws.text, localRawX)
					x = marginLeft + pfxLen + localVisX
					break
				}
			}
		}

	case lineKindCode, lineKindTable, lineKindImage:
		// These now soft-wrap via bookWrapPlainRunes in bookTextModeRender.
		adjX := cursorRawX
		if pl.kind == lineKindImage {
			// The rendered line is "[image: " + body + "]" — shift the
			// cursor by the prefix length so it stays aligned with what
			// the user sees.
			adjX += len("[image: ")
		}
		if textW > 0 && adjX >= textW {
			subRow = adjX / textW
			adjX -= subRow * textW
		}
		x = adjX + marginLeft

	case lineKindBlank, lineKindRule:
		x = cursorRawX + marginLeft

	default: // lineKindBody
		wsegs := bookWrapBody(pl.body, textW)
		for i, ws := range wsegs {
			if cursorRawX < ws.end || i == len(wsegs)-1 {
				subRow = i
				localRawX := max(cursorRawX-ws.start, 0)
				localVisX := rawXToVisualX(ws.text, localRawX)
				x = marginLeft + localVisX
				break
			}
		}
	}

	h := int(c.Height())
	editRows := int(e.bookEditRows(uint(h)))
	topMargin := int(float64(editRows) * bookMarginTop)

	// The filename bar occupies row 0 in text mode; shift the cursor
	// down by that many rows so it lands on the rendered text row.
	y := displayRow + subRow + topMargin + bookTextTopBarRows

	if x >= w {
		x = w - 1
	}
	if x < marginLeft {
		x = marginLeft
	}

	// Position first, THEN show. If ShowCursor ran before SetXY the
	// caret would briefly appear at the position left behind by the
	// preceding HideCursorAndDraw (typically the end of the last row
	// drawn, which can be inside the top bar), then jump to the final
	// position — a visible flash. Reversing the order keeps the caret
	// invisible until it is already where it belongs.
	vt.SetXY(uint(x), uint(y))
	c.ShowCursor()
}

// bookWrapWidth returns the available text width in runes for wrapping
// in the current terminal (marginRight - marginLeft).
// bookWrapWidth returns the column width available to book-mode text after
// subtracting the left and right margins. The right margin differs between
// text and graphical modes (tighter in text mode) — branching here keeps
// scroll-row-counting in sync with what bookTextModeRender/bookContentImage
// actually paint.
func (e *Editor) bookWrapWidth(c *vt.Canvas) int {
	w := int(c.Width())
	marginLeft := max(int(float64(w)*bookMarginLeft), 2)
	rightFrac := bookMarginRight
	if e.bookTextMode() {
		rightFrac = bookTextMarginRight
	}
	marginRight := w - max(int(float64(w)*rightFrac), 1)
	return marginRight - marginLeft
}

// bookPixelWrapInfo holds the pixel-based wrapping parameters needed by the
// graphical book mode cursor movement functions. When non-nil it causes
// sub-row calculations to use bookWrapSegmentsPixel (proportional font) instead
// of bookWrapBody (fixed-width rune count).
type bookPixelWrapInfo struct {
	fs                *bookFontSet
	marginLeft        int
	marginRight       int
	lineH             int
	bodyTextPxFullRow int // pixel width available for body text on a full-width row (marginRight - marginLeft)
}

// bookGetPixelWrapInfo returns the pixel wrapping parameters when in graphical
// book mode, or nil when in text mode.
func (e *Editor) bookGetPixelWrapInfo(c *vt.Canvas) *bookPixelWrapInfo {
	if !e.bookGraphicalMode() {
		return nil
	}
	cellW, cellH := imagepreview.TerminalCellPixels()
	if cellH == 0 {
		cellH = 16
	}
	if cellW == 0 {
		cellW = 8
	}
	_, renderH := bookRenderCellSize(cellW, cellH)
	rows := int(c.Height())
	editRows := int(e.bookEditRows(uint(rows)))
	if editRows <= 0 {
		return nil
	}
	lineH := max(int(renderH), 1)
	pixW := int(uint(c.Width()) * cellW)
	marginLeft := int(float64(pixW) * bookMarginLeft)
	marginRight := pixW - int(float64(pixW)*bookMarginRight)
	fontSize := float64(renderH) * 0.72
	if fontSize < 6 {
		fontSize = 6
	}
	fs, err := bookFaces(fontSize)
	if err != nil {
		return nil
	}
	return &bookPixelWrapInfo{
		fs:                fs,
		marginLeft:        marginLeft,
		marginRight:       marginRight,
		lineH:             lineH,
		bodyTextPxFullRow: marginRight - marginLeft,
	}
}

// pixelBodyAvailPx returns the pixel width available for body text on a
// list/checkbox line, matching the calculation in bookContentImage and
// bookOverlayCursor.
func (pw *bookPixelWrapInfo) pixelBodyAvailPx(pl parsedLine) int {
	switch pl.kind {
	case lineKindUnchecked, lineKindChecked:
		size := max(pw.lineH*55/100, 5)
		prefixEndX := pw.marginLeft + size + 3
		return pw.marginRight - prefixEndX
	case lineKindBullet, lineKindNumbered:
		prefixEndX := pw.marginLeft + measureStringFB(pw.fs.regular, pl.prefix).Round()
		return pw.marginRight - prefixEndX
	default:
		return pw.bodyTextPxFullRow
	}
}

// bookCursorSubRow returns (subRow, totalSubRows) for the current cursor
// within a soft-wrapped line. For non-wrapping line kinds (header, code,
// blank, rule, image) it returns (0, 1). When pw is non-nil, pixel-based
// wrapping (proportional font) is used instead of character-count wrapping.
//
// When the cursor's raw position falls exactly at a wrap boundary (shared
// by the end of row N and the start of row N+1), the editor's
// bookCursorAffinity disambiguates: bookAffinityBackward places the cursor
// on row N; bookAffinityForward (the default) places it on row N+1.
func (e *Editor) bookCursorSubRow(textW int, pw *bookPixelWrapInfo) (subRow, totalRows int) {
	rawLine := e.Line(e.DataY())
	rawLine = strings.ReplaceAll(rawLine, "\t", "    ")
	cursorRawX := e.pos.sx + e.pos.offsetX
	pl := parseBookLine(rawLine)

	switch pl.kind {
	case lineKindHeader, lineKindCode, lineKindTable, lineKindBlank, lineKindRule, lineKindImage:
		return 0, 1
	}

	backward := e.bookCursorAffinity == bookAffinityBackward

	if pw != nil {
		// Pixel-based wrapping for graphical mode.
		bodyAvailPx := pw.pixelBodyAvailPx(pl)
		wrows := bookWrapSegmentsPixel(pw.fs, pl.body, bodyAvailPx)

		// Determine the visual body rune offset.
		rawPrefix := rawMarkdownPrefix(rawLine)
		rawBodyStart := len([]rune(rawPrefix))
		bodyRawX := cursorRawX
		if pl.kind != lineKindBody {
			bodyRawX -= rawBodyStart
		}
		if bodyRawX < 0 {
			return 0, len(wrows)
		}
		bodyVisX := rawXToVisualX(pl.body, bodyRawX)
		pos := 0
		for i, wr := range wrows {
			end := pos + wr.runeCount
			last := i == len(wrows)-1
			if bodyVisX < end || last {
				return i, len(wrows)
			}
			if bodyVisX == end && backward {
				return i, len(wrows)
			}
			pos = end
		}
		return 0, len(wrows)
	}

	// Character-count wrapping for text mode.
	switch pl.kind {
	case lineKindBullet, lineKindNumbered, lineKindUnchecked, lineKindChecked:
		pfxLen := len([]rune(pl.prefix))
		rawPrefix := rawMarkdownPrefix(rawLine)
		rawBodyStart := len([]rune(rawPrefix))
		bodyRawX := cursorRawX - rawBodyStart
		bodyAvailW := textW - pfxLen
		if bodyAvailW <= 0 {
			bodyAvailW = 1
		}
		wsegs := bookWrapBody(pl.body, bodyAvailW)
		if bodyRawX < 0 {
			return 0, len(wsegs)
		}
		pos := 0
		for i, ws := range wsegs {
			segLen := len([]rune(ws.text))
			end := pos + segLen
			last := i == len(wsegs)-1
			if bodyRawX < end || last {
				return i, len(wsegs)
			}
			if bodyRawX == end && backward {
				return i, len(wsegs)
			}
			pos = end
		}
		return 0, len(wsegs)
	default: // lineKindBody
		wsegs := bookWrapBody(pl.body, textW)
		pos := 0
		for i, ws := range wsegs {
			segLen := len([]rune(ws.text))
			end := pos + segLen
			last := i == len(wsegs)-1
			if cursorRawX < end || last {
				return i, len(wsegs)
			}
			if cursorRawX == end && backward {
				return i, len(wsegs)
			}
			pos = end
		}
		return 0, len(wsegs)
	}
}

// bookLineDisplayRows returns the number of display rows a data line occupies
// given the current wrap context. When pw is non-nil, pixel-based wrapping is
// used (graphical book mode); otherwise rune-based wrapping is used (text
// book mode).
func (e *Editor) bookLineDisplayRows(dl int, textW int, pw *bookPixelWrapInfo) int {
	rl := e.Line(LineIndex(dl))
	rl = strings.ReplaceAll(rl, "\t", "    ")
	pl := parseBookLine(rl)
	if pl.kind == lineKindImage {
		// In text mode (pw == nil) an image line renders as a single
		// "[image: url]" row, matching bookTextModeRender and
		// bookWrapRowCount. Calling bookImageRows here with lineH=0
		// would clamp to lineH=1 and return a bogus pixel-scaled row
		// count, which broke PgUp/PgDn and cursor scroll past images.
		if pw == nil {
			return 1
		}
		return e.bookImageRows(pl.body, pw.lineH, pw.bodyTextPxFullRow)
	}
	if pw != nil {
		return bookPixelRowCount(pw.fs, pl, pw.lineH, pw.marginLeft, pw.marginRight)
	}
	return bookWrapRowCount(pl, textW)
}

// bookVisibleDataLines scans forward from startDoc and returns how many data
// lines fit within editRows display rows. A partially-visible line at the
// bottom is NOT counted. The minimum return value is 1 so PgDn always makes
// progress, even when the top line alone would overflow the edit area.
func (e *Editor) bookVisibleDataLines(startDoc, editRows, textW int, pw *bookPixelWrapInfo) int {
	totalLines := e.Len()
	rowsUsed := 0
	count := 0
	for dl := startDoc; dl < totalLines; dl++ {
		lineRows := max(e.bookLineDisplayRows(dl, textW, pw), 1)
		if rowsUsed+lineRows > editRows {
			break
		}
		rowsUsed += lineRows
		count++
	}
	if count < 1 {
		count = 1
	}
	return count
}

// bookPgDn scrolls forward by approximately one visible page of display rows,
// honouring soft wrap. It moves both offsetY and the cursor to the first data
// line that was previously just below the visible area. Returns true if the
// viewport changed.
func (e *Editor) bookPgDn(c *vt.Canvas) bool {
	rows := int(c.Height())
	editRows := int(e.bookEditRows(uint(rows)))
	if editRows <= 0 {
		return false
	}
	textW := e.bookWrapWidth(c)
	pw := e.bookGetPixelWrapInfo(c)
	totalLines := e.Len()
	if totalLines == 0 {
		return false
	}
	step := e.bookVisibleDataLines(e.pos.offsetY, editRows, textW, pw)
	newOffset := e.pos.offsetY + step
	if newOffset >= totalLines {
		newOffset = totalLines - 1
	}
	if newOffset == e.pos.offsetY && int(e.DataY()) == newOffset {
		return false
	}
	e.pos.offsetY = newOffset
	e.pos.sy = 0
	e.pos.SetX(c, 0)
	e.bookSavedLocalX = -1
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
	return true
}

// bookPgUp scrolls backward by approximately one visible page of display
// rows, honouring soft wrap. It moves both offsetY and the cursor back by as
// many data lines as fit on the previous page. Returns true if the viewport
// changed.
func (e *Editor) bookPgUp(c *vt.Canvas) bool {
	rows := int(c.Height())
	editRows := int(e.bookEditRows(uint(rows)))
	if editRows <= 0 {
		return false
	}
	if e.pos.offsetY == 0 && int(e.DataY()) == 0 {
		return false
	}
	textW := e.bookWrapWidth(c)
	pw := e.bookGetPixelWrapInfo(c)
	// Walk backwards accumulating display rows until we've covered editRows.
	dl := e.pos.offsetY - 1
	rowsUsed := 0
	for dl >= 0 {
		lineRows := max(e.bookLineDisplayRows(dl, textW, pw), 1)
		if rowsUsed+lineRows > editRows && dl < e.pos.offsetY-1 {
			dl++
			break
		}
		rowsUsed += lineRows
		if dl == 0 {
			break
		}
		dl--
	}
	if dl < 0 {
		dl = 0
	}
	if dl == e.pos.offsetY && int(e.DataY()) == dl {
		return false
	}
	e.pos.offsetY = dl
	e.pos.sy = 0
	e.pos.SetX(c, 0)
	e.bookSavedLocalX = -1
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
	return true
}

// bookCursorDown moves the cursor down one display row. If the current line
// is soft-wrapped and the cursor is not on the last sub-row, it moves within
// the same data line. Otherwise it moves to the next data line.
func (e *Editor) bookCursorDown(c *vt.Canvas, status *StatusBar) bool {
	textW := e.bookWrapWidth(c)
	pw := e.bookGetPixelWrapInfo(c)
	sub, total := e.bookCursorSubRow(textW, pw)

	// Save the visual X on the first up/down press so that subsequent
	// presses try to land on the same column.
	if e.bookSavedLocalX < 0 {
		e.bookSavedLocalX = e.pos.sx + e.pos.offsetX
	}
	savedX := e.bookSavedLocalX

	if sub < total-1 {
		// Move within the same data line to the next sub-row.
		targetSub := sub + 1
		newX := bookSubRowX(e.Line(e.DataY()), textW, targetSub, savedX, pw)
		e.pos.SetX(c, newX)
		// If the new position lands at the wrap boundary of a non-last
		// sub-row, pin it to the end of targetSub; otherwise forward.
		if targetSub < total-1 && newX == bookSubRowEndX(e.Line(e.DataY()), textW, targetSub, pw) {
			e.bookCursorAffinity = bookAffinityBackward
		} else {
			e.bookCursorAffinity = bookAffinityForward
		}
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
		return true
	}
	// At the last sub-row — move to the next data line. Not using
	// e.CursorDownward because it thinks in canvas cells and triggers a
	// scroll at the wrong time for soft-wrapped book-mode layout.
	if int(e.DataY()) >= e.Len()-1 {
		// Already on the last data line. Surface an EOF message the first time
		// we bump into the end, like CursorDownward does in non-book mode.
		if status != nil && status.Message() != endOfFileMessage {
			status.Clear(c, false)
			status.SetMessage(endOfFileMessage)
			status.ShowNoTimeout(c, e)
		}
		return false
	}
	e.pos.SetY(e.pos.sy + 1)
	// Land on the first sub-row of the new line at the saved visual column.
	newX := bookSubRowX(e.Line(e.DataY()), textW, 0, savedX, pw)
	e.pos.SetX(c, newX)
	// First sub-row of a new line: forward affinity unless it lies on the
	// wrap boundary of a multi-sub-row line (i.e. savedX past end of row 0).
	newTotal := bookLineSubRowCount(e.Line(e.DataY()), textW, pw)
	if newTotal > 1 && newX == bookSubRowEndX(e.Line(e.DataY()), textW, 0, pw) {
		e.bookCursorAffinity = bookAffinityBackward
	} else {
		e.bookCursorAffinity = bookAffinityForward
	}
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
	return true
}

// bookCursorUp moves the cursor up one display row. If the current line
// is soft-wrapped and the cursor is not on the first sub-row, it moves within
// the same data line. Otherwise it moves to the previous data line.
func (e *Editor) bookCursorUp(c *vt.Canvas, status *StatusBar) bool {
	textW := e.bookWrapWidth(c)
	pw := e.bookGetPixelWrapInfo(c)
	sub, total := e.bookCursorSubRow(textW, pw)

	// Save the visual X on the first up/down press.
	if e.bookSavedLocalX < 0 {
		e.bookSavedLocalX = e.pos.sx + e.pos.offsetX
	}
	savedX := e.bookSavedLocalX

	if sub > 0 {
		// Move within the same data line to the previous sub-row.
		targetSub := sub - 1
		newX := bookSubRowX(e.Line(e.DataY()), textW, targetSub, savedX, pw)
		e.pos.SetX(c, newX)
		// If savedX was at or past the end of targetSub, the cursor ends
		// up at the wrap boundary shared with the next sub-row. Pin it to
		// the end of targetSub with backward affinity so it renders on
		// the expected row. Without this, pressing Up from the start of
		// the second sub-row would be a no-op (bug fix).
		if targetSub < total-1 && newX == bookSubRowEndX(e.Line(e.DataY()), textW, targetSub, pw) {
			e.bookCursorAffinity = bookAffinityBackward
		} else {
			e.bookCursorAffinity = bookAffinityForward
		}
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
		return true
	}
	// At the first sub-row — move to the previous data line.
	if e.DataY() == 0 {
		return false
	}
	// Don't use e.CursorUpward (see bookCursorDown for the rationale).
	// Advance DataY directly by decrementing sy; bookModeEnsureCursorVisible
	// will handle scroll-up when sy goes below zero.
	e.pos.SetY(e.pos.sy - 1)
	// Land on the last sub-row of the previous line at the saved visual column.
	lastSub := max(bookLineSubRowCount(e.Line(e.DataY()), textW, pw)-1, 0)
	newX := bookSubRowX(e.Line(e.DataY()), textW, lastSub, savedX, pw)
	e.pos.SetX(c, newX)
	// Landing on the last sub-row: forward affinity (the boundary ambiguity
	// doesn't apply to the last sub-row since there is no row after it).
	e.bookCursorAffinity = bookAffinityForward
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
	return true
}

// bookLineKindAt returns the parsed line kind of the given data line. Handy
// for cursor-movement code that needs to special-case atomic line kinds
// (e.g. image lines, which render as "[image: path]" and share no column
// positions with their raw "![alt](path)" source).
func (e *Editor) bookLineKindAt(dl LineIndex) lineKind {
	rl := e.Line(dl)
	rl = strings.ReplaceAll(rl, "\t", "    ")
	return parseBookLine(rl).kind
}

// bookLineSubRowCount returns the total number of display sub-rows for a data
// line when soft-wrapped at the given textW. When pw is non-nil, pixel-based
// wrapping is used.
func bookLineSubRowCount(rawLine string, textW int, pw *bookPixelWrapInfo) int {
	rawLine = strings.ReplaceAll(rawLine, "\t", "    ")
	pl := parseBookLine(rawLine)
	switch pl.kind {
	case lineKindHeader, lineKindCode, lineKindTable, lineKindBlank, lineKindRule, lineKindImage:
		return 1
	}
	if pw != nil {
		bodyAvailPx := pw.pixelBodyAvailPx(pl)
		n := len(bookWrapSegmentsPixel(pw.fs, pl.body, bodyAvailPx))
		if n < 1 {
			return 1
		}
		return n
	}
	switch pl.kind {
	case lineKindBullet, lineKindNumbered, lineKindUnchecked, lineKindChecked:
		bodyAvailW := textW - len([]rune(pl.prefix))
		if bodyAvailW <= 0 {
			bodyAvailW = 1
		}
		n := len(bookWrapBody(pl.body, bodyAvailW))
		if n < 1 {
			return 1
		}
		return n
	default: // lineKindBody
		n := len(bookWrapBody(pl.body, textW))
		if n < 1 {
			return 1
		}
		return n
	}
}

// bookSubRowX computes the rune position within a data line that corresponds
// to visual column savedX on the given sub-row (0-based). The returned value
// accounts for list prefixes and soft wrapping. When pw is non-nil, pixel-based
// wrapping is used.
func bookSubRowX(rawLine string, textW, targetSub, savedX int, pw *bookPixelWrapInfo) int {
	rawLine = strings.ReplaceAll(rawLine, "\t", "    ")
	pl := parseBookLine(rawLine)

	switch pl.kind {
	case lineKindHeader, lineKindCode, lineKindTable, lineKindBlank, lineKindRule, lineKindImage:
		lineLen := len([]rune(rawLine))
		if savedX > lineLen {
			return lineLen
		}
		return savedX
	}

	rawPfxLen := 0
	if pl.kind != lineKindBody {
		rawPfxLen = len([]rune(rawMarkdownPrefix(rawLine)))
	}

	if pw != nil {
		// Pixel-based wrapping for graphical mode.
		bodyAvailPx := pw.pixelBodyAvailPx(pl)
		wrows := bookWrapSegmentsPixel(pw.fs, pl.body, bodyAvailPx)
		if targetSub < 0 {
			targetSub = 0
		}
		if targetSub >= len(wrows) {
			targetSub = len(wrows) - 1
		}

		// savedX is a raw rune position — convert to visual body rune offset.
		bodyRawX := savedX
		if pl.kind != lineKindBody {
			bodyRawX -= rawPfxLen
		}
		if bodyRawX < 0 {
			bodyRawX = 0
		}
		bodyVisX := rawXToVisualX(pl.body, bodyRawX)

		// Subtract rune counts of sub-rows before the target to get the
		// local visual column within this sub-row.
		localVisX := bodyVisX
		for i := 0; i < targetSub && i < len(wrows); i++ {
			localVisX -= wrows[i].runeCount
		}
		if localVisX < 0 {
			localVisX = 0
		}
		if localVisX > wrows[targetSub].runeCount {
			localVisX = wrows[targetSub].runeCount
		}

		// Convert the visual position back to a raw rune position.
		// The target visual rune count is the sum of prior rows + localVisX.
		targetVisX := 0
		for i := 0; i < targetSub; i++ {
			targetVisX += wrows[i].runeCount
		}
		targetVisX += localVisX
		newX := visualXToRawX(pl.body, targetVisX)
		if pl.kind != lineKindBody {
			newX += rawPfxLen
		}
		return newX
	}

	// Character-count wrapping for text mode.
	pfxLen := 0
	bodyText := rawLine
	bodyAvailW := textW

	switch pl.kind {
	case lineKindBullet, lineKindNumbered, lineKindUnchecked, lineKindChecked:
		pfxLen = len([]rune(pl.prefix))
		bodyText = pl.body
		bodyAvailW = textW - pfxLen
		if bodyAvailW <= 0 {
			bodyAvailW = 1
		}
	}

	wsegs := bookWrapBody(bodyText, bodyAvailW)
	if targetSub < 0 {
		targetSub = 0
	}
	if targetSub >= len(wsegs) {
		targetSub = len(wsegs) - 1
	}

	runeOffset := 0
	for i := 0; i < targetSub; i++ {
		runeOffset += len([]rune(wsegs[i].text))
	}
	segLen := len([]rune(wsegs[targetSub].text))

	localX := savedX
	if pl.kind != lineKindBody {
		localX -= pfxLen
	}
	localX -= runeOffset
	if localX < 0 {
		localX = 0
	}
	if localX > segLen {
		localX = segLen
	}

	newX := runeOffset + localX
	if pl.kind != lineKindBody {
		newX += rawPfxLen
	}
	return newX
}

// bookSubRowEndX returns the raw rune position that corresponds to the end
// (wrap boundary) of the given soft-wrapped sub-row. For the last sub-row
// it returns the raw rune position of the end of the data line. Used to
// decide cursor affinity when placing the cursor at a wrap boundary.
func bookSubRowEndX(rawLine string, textW, sub int, pw *bookPixelWrapInfo) int {
	rawLine = strings.ReplaceAll(rawLine, "\t", "    ")
	pl := parseBookLine(rawLine)
	switch pl.kind {
	case lineKindHeader, lineKindCode, lineKindTable, lineKindBlank, lineKindRule, lineKindImage:
		return len([]rune(rawLine))
	}
	rawPfxLen := 0
	if pl.kind != lineKindBody {
		rawPfxLen = len([]rune(rawMarkdownPrefix(rawLine)))
	}
	clamp := func(n, total int) int {
		if n < 0 {
			return 0
		}
		if n >= total {
			return total - 1
		}
		return n
	}
	if pw != nil {
		bodyAvailPx := pw.pixelBodyAvailPx(pl)
		wrows := bookWrapSegmentsPixel(pw.fs, pl.body, bodyAvailPx)
		if len(wrows) == 0 {
			return rawPfxLen
		}
		sub = clamp(sub, len(wrows))
		visEnd := 0
		for i := 0; i <= sub; i++ {
			visEnd += wrows[i].runeCount
		}
		return rawPfxLen + visualXToRawX(pl.body, visEnd)
	}
	bodyAvailW := textW
	if pl.kind != lineKindBody {
		bodyAvailW = textW - len([]rune(pl.prefix))
		if bodyAvailW <= 0 {
			bodyAvailW = 1
		}
	}
	wsegs := bookWrapBody(pl.body, bodyAvailW)
	if len(wsegs) == 0 {
		return rawPfxLen
	}
	sub = clamp(sub, len(wsegs))
	runeEnd := 0
	for i := 0; i <= sub; i++ {
		runeEnd += len([]rune(wsegs[i].text))
	}
	return rawPfxLen + runeEnd
}

// bookHome moves the cursor to the start of the current display sub-row.
// If the cursor is already at the start of the current sub-row, it goes
// to the very start of the data line (column 0).
func (e *Editor) bookHome(c *vt.Canvas) {
	textW := e.bookWrapWidth(c)
	pw := e.bookGetPixelWrapInfo(c)
	sub, _ := e.bookCursorSubRow(textW, pw)
	if sub == 0 {
		e.Home()
		e.bookCursorAffinity = bookAffinityForward
		return
	}
	// Place cursor at the start of the current sub-row.
	newX := bookSubRowX(e.Line(e.DataY()), textW, sub, 0, pw)
	curX := e.pos.sx + e.pos.offsetX
	if curX == newX {
		e.Home()
		e.bookCursorAffinity = bookAffinityForward
		return
	}
	e.pos.SetX(c, newX)
	// At the start of sub-row N we want forward affinity so the cursor
	// renders on sub-row N (not as the end of sub-row N-1).
	e.bookCursorAffinity = bookAffinityForward
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
}

// bookEnd moves the cursor to the end of the current display sub-row.
// If the cursor is already at the end of the current sub-row, it goes
// to the very end of the data line. Sets backward affinity when the
// cursor is left at a wrap boundary so it renders at the end of the
// current sub-row rather than the start of the next.
func (e *Editor) bookEnd(c *vt.Canvas) {
	textW := e.bookWrapWidth(c)
	pw := e.bookGetPixelWrapInfo(c)
	sub, total := e.bookCursorSubRow(textW, pw)
	if sub >= total-1 {
		e.End(c)
		e.bookCursorAffinity = bookAffinityForward
		return
	}
	// Place cursor at the end of the current sub-row by requesting the
	// start of the next sub-row (which is the end of the current one).
	rawLine := e.Line(e.DataY())
	rawLine = strings.ReplaceAll(rawLine, "\t", "    ")
	pl := parseBookLine(rawLine)

	rawPfxLen := 0
	if pl.kind != lineKindBody {
		rawPfxLen = len([]rune(rawMarkdownPrefix(rawLine)))
	}

	var newX int
	if pw != nil {
		// Pixel wrapping: sum runeCount of sub-rows 0..sub.
		bodyAvailPx := pw.pixelBodyAvailPx(pl)
		wrows := bookWrapSegmentsPixel(pw.fs, pl.body, bodyAvailPx)
		visEnd := 0
		for i := 0; i <= sub && i < len(wrows); i++ {
			visEnd += wrows[i].runeCount
		}
		newX = visualXToRawX(pl.body, visEnd)
		if pl.kind != lineKindBody {
			newX += rawPfxLen
		}
	} else {
		// Character wrapping.
		pfxLen := 0
		bodyText := rawLine
		bodyAvailW := textW
		switch pl.kind {
		case lineKindBullet, lineKindNumbered, lineKindUnchecked, lineKindChecked:
			pfxLen = len([]rune(rawMarkdownPrefix(rawLine)))
			bodyText = pl.body
			bodyAvailW = textW - len([]rune(pl.prefix))
			if bodyAvailW <= 0 {
				bodyAvailW = 1
			}
		}
		_ = pfxLen
		wsegs := bookWrapBody(bodyText, bodyAvailW)
		runeOffset := 0
		for i := 0; i <= sub && i < len(wsegs); i++ {
			runeOffset += len([]rune(wsegs[i].text))
		}
		newX = runeOffset
		if pl.kind != lineKindBody {
			newX += rawPfxLen
		}
	}

	curX := e.pos.sx + e.pos.offsetX
	if curX == newX && e.bookCursorAffinity == bookAffinityBackward {
		// Already at the wrap boundary rendered as end-of-sub — go to line end.
		e.End(c)
		e.bookCursorAffinity = bookAffinityForward
		return
	}
	e.pos.SetX(c, newX)
	// Pin the cursor to the end of the current sub-row rather than the
	// start of the next.
	e.bookCursorAffinity = bookAffinityBackward
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
}

// bookMarkerLenAt returns the length in runes of an inline Markdown marker
// starting at index i in runes, or 0 if there is no marker there. Recognised
// markers: ***, **, *, __, ~~, `. This mirrors the logic used by
// rawXToVisualX/visualXToRawX so the three stay in sync.
func bookMarkerLenAt(runes []rune, i int) int {
	n := len(runes)
	if i < 0 || i >= n {
		return 0
	}
	if i+2 < n && runes[i] == '*' && runes[i+1] == '*' && runes[i+2] == '*' {
		return 3
	}
	if i+1 < n && runes[i] == '*' && runes[i+1] == '*' {
		return 2
	}
	if runes[i] == '*' {
		return 1
	}
	if i+1 < n && runes[i] == '_' && runes[i+1] == '_' {
		return 2
	}
	if i+1 < n && runes[i] == '~' && runes[i+1] == '~' {
		return 2
	}
	if runes[i] == '`' {
		return 1
	}
	return 0
}

// bookSkipMarkersForward advances rawX past any inline Markdown markers at
// rawX so the cursor lands just after them.
func bookSkipMarkersForward(runes []rune, rawX int) int {
	for rawX < len(runes) {
		ml := bookMarkerLenAt(runes, rawX)
		if ml == 0 {
			break
		}
		rawX += ml
	}
	return rawX
}

// bookSkipMarkersBackward retreats rawX past any inline Markdown marker that
// ends exactly at rawX so the cursor lands just before the marker rather
// than inside or right after it.
func bookSkipMarkersBackward(runes []rune, rawX int) int {
	for rawX > 0 {
		switch {
		case rawX >= 3 && runes[rawX-1] == '*' && runes[rawX-2] == '*' && runes[rawX-3] == '*':
			rawX -= 3
		case rawX >= 2 && runes[rawX-1] == '*' && runes[rawX-2] == '*':
			rawX -= 2
		case rawX >= 2 && runes[rawX-1] == '_' && runes[rawX-2] == '_':
			rawX -= 2
		case rawX >= 2 && runes[rawX-1] == '~' && runes[rawX-2] == '~':
			rawX -= 2
		case rawX >= 1 && runes[rawX-1] == '*':
			rawX--
		case rawX >= 1 && runes[rawX-1] == '`':
			rawX--
		default:
			return rawX
		}
	}
	return rawX
}

// bookCursorForward moves one visible column to the right, skipping over any
// Markdown inline markers (*, **, ***, __, ~~, `) so that each key-press
// actually advances the visible cursor position — matching the behaviour of
// WYSIWYG word processors.
func (e *Editor) bookCursorForward(c *vt.Canvas, status *StatusBar) bool {
	// Image lines render as "[image: path]" which shares no positions
	// with the raw "![alt](path)" source. Moving the cursor rune-by-rune
	// over the raw source therefore produces visually erratic jumps.
	// Treat the line atomically: Right jumps straight to the next line.
	if e.bookLineKindAt(e.DataY()) == lineKindImage {
		beforeY := e.DataY()
		if int(beforeY)+1 >= e.Len() {
			return false
		}
		e.pos.Down(c)
		e.Home()
		e.bookCursorAffinity = bookAffinityForward
		e.redrawCursor.Store(true)
		e.SaveX(true)
		return e.DataY() != beforeY
	}
	// At a wrap boundary with backward affinity, Right should flip the
	// affinity to forward (visible movement: cursor jumps from end of
	// sub-row N to start of sub-row N+1) without actually moving rawX.
	if e.bookCursorAffinity == bookAffinityBackward {
		textW := e.bookWrapWidth(c)
		pw := e.bookGetPixelWrapInfo(c)
		sub, total := e.bookCursorSubRow(textW, pw)
		rawX := e.pos.sx + e.pos.offsetX
		if sub < total-1 && rawX == bookSubRowEndX(e.Line(e.DataY()), textW, sub, pw) {
			e.bookCursorAffinity = bookAffinityForward
			e.redrawCursor.Store(true)
			e.SaveX(true)
			return true
		}
	}
	runes := []rune(e.Line(e.DataY()))
	rawX := e.pos.sx + e.pos.offsetX
	// First, skip any marker the cursor is sitting at/inside.
	snapped := bookSkipMarkersForward(runes, rawX)
	if snapped != rawX {
		e.pos.SetX(c, snapped)
		e.bookCursorAffinity = bookAffinityForward
		e.redrawCursor.Store(true)
		e.SaveX(true)
		return true
	}
	// No leading marker: advance one raw char, then skip over any trailing
	// markers so the cursor stops on the next visible position.
	beforeY, beforeX := e.DataY(), e.pos.sx
	e.CursorForward(c, status)
	runes = []rune(e.Line(e.DataY()))
	rawX = e.pos.sx + e.pos.offsetX
	snapped = bookSkipMarkersForward(runes, rawX)
	if snapped != rawX {
		e.pos.SetX(c, snapped)
		e.redrawCursor.Store(true)
		e.SaveX(true)
	}
	e.bookCursorAffinity = bookAffinityForward
	return e.DataY() != beforeY || e.pos.sx != beforeX
}

// bookCursorBackward moves one visible column to the left, skipping over any
// Markdown inline markers so each key-press advances visibly.
func (e *Editor) bookCursorBackward(c *vt.Canvas, status *StatusBar) bool {
	// Atomic image lines: Left from the start of an image line jumps to
	// the end of the previous line, and from anywhere else on the image
	// line snaps back to the line's start.
	if e.bookLineKindAt(e.DataY()) == lineKindImage {
		rawX := e.pos.sx + e.pos.offsetX
		if rawX > 0 {
			e.Home()
			e.bookCursorAffinity = bookAffinityForward
			e.redrawCursor.Store(true)
			e.SaveX(true)
			return true
		}
		beforeY := e.DataY()
		if beforeY == 0 {
			return false
		}
		e.pos.Up()
		e.End(c)
		e.bookCursorAffinity = bookAffinityForward
		e.redrawCursor.Store(true)
		e.SaveX(true)
		return e.DataY() != beforeY
	}
	// At a wrap boundary with forward affinity (cursor rendered at start
	// of sub-row N+1 sharing rawX with end of sub-row N), Left should flip
	// affinity to backward without moving rawX. This matches word
	// processors where pressing Left from column 0 of a visually-wrapped
	// line lands the cursor at the end of the previous visual line.
	if e.bookCursorAffinity == bookAffinityForward {
		textW := e.bookWrapWidth(c)
		pw := e.bookGetPixelWrapInfo(c)
		sub, _ := e.bookCursorSubRow(textW, pw)
		rawX := e.pos.sx + e.pos.offsetX
		if sub > 0 && rawX == bookSubRowEndX(e.Line(e.DataY()), textW, sub-1, pw) {
			e.bookCursorAffinity = bookAffinityBackward
			e.redrawCursor.Store(true)
			e.SaveX(true)
			return true
		}
	}
	runes := []rune(e.Line(e.DataY()))
	rawX := e.pos.sx + e.pos.offsetX
	// First, if the cursor is right after a marker, step past it in one go.
	snapped := bookSkipMarkersBackward(runes, rawX)
	if snapped != rawX {
		e.pos.SetX(c, snapped)
		e.bookCursorAffinity = bookAffinityForward
		e.redrawCursor.Store(true)
		e.SaveX(true)
		return true
	}
	// Otherwise retreat one raw char, then skip any marker we landed in.
	beforeY, beforeX := e.DataY(), e.pos.sx
	e.CursorBackward(c, status)
	runes = []rune(e.Line(e.DataY()))
	rawX = e.pos.sx + e.pos.offsetX
	snapped = bookSkipMarkersBackward(runes, rawX)
	if snapped != rawX {
		e.pos.SetX(c, snapped)
		e.redrawCursor.Store(true)
		e.SaveX(true)
	}
	e.bookCursorAffinity = bookAffinityForward
	return e.DataY() != beforeY || e.pos.sx != beforeX
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
	e.MarkChanged()
	e.redraw.Store(true)
}

// bookScreenshot opens the given Markdown file in graphical book mode, renders
// the first visible page to a PNG image and writes it to outPath. Headless
// operation used for visual debugging. When outPath ends with ".png", a single
// screenshot is taken; otherwise per-line screenshots are written as
// outPath_0.png, outPath_1.png, …
func bookScreenshot(mdPath, outPath string) error {
	// Simulated terminal dimensions (columns × rows).
	const termCols = 120
	const termRows = 40
	// Simulated cell size in pixels.
	const cellW = 8
	const cellH = 18

	// Create a minimal editor in book mode.
	t := NewLightVSTheme()
	e := NewCustomEditor(mode.DefaultTabsSpaces, 1, mode.Markdown, t, false, false, false, false, false, false, false)
	e.filename = mdPath

	if err := e.ReadFileAndProcessLines(mdPath); err != nil {
		return fmt.Errorf("reading %s: %w", mdPath, err)
	}

	// Compute image dimensions.
	editRows := uint(termRows - 1)
	renderW, renderH := bookRenderCellSize(cellW, cellH)
	pixW := int(uint(termCols) * renderW)
	pixH := int(editRows * renderH)

	// Generate screenshots with the cursor on each visible line.
	totalLines := e.Len()
	maxLines := min(totalLines, int(editRows))

	for curLine := range maxLines {
		// Position cursor at the start of this line.
		e.pos.sy = curLine
		e.pos.sx = 0
		e.pos.offsetX = 0
		e.pos.offsetY = 0

		// Force fresh content render.
		bookContentCache = nil

		img := e.bookContentImage(pixW, pixH, int(editRows), renderH)
		e.bookOverlayCursor(img, pixW, pixH, int(editRows), renderH)

		// Apply top corners. Outside-fill is terminal chrome (black).
		cornerRadius := max(min(pixW, pixH)/40, 6)
		// Use the same logic as the page itself: fade top corners to black (terminal chrome)
		bookRoundedCorners(img, cornerRadius, true, false, color.NRGBA{0x00, 0x00, 0x00, 0xff})

		// Draw row guides.
		bookScreenshotAnnotate(img, e, pixW, pixH, int(editRows), renderH)

		path := fmt.Sprintf("%s_%d.png", strings.TrimSuffix(outPath, ".png"), curLine)
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			return err
		}
		f.Close()

		rl := e.Line(LineIndex(curLine))
		preview := strings.ReplaceAll(rl, "\t", "    ")
		if len(preview) > 60 {
			preview = preview[:60] + "…"
		}
		fmt.Printf("Line %2d: %s → %s\n", curLine, preview, path)
	}

	fmt.Printf("\nGenerated %d screenshots (%dx%d px, %d cols × %d rows)\n",
		maxLines, pixW, pixH, termCols, termRows)
	return nil
}

// bookScreenshotAnnotate draws thin red horizontal guides at each display row
// boundary and a small red dot at the cursor position for visual debugging.
func bookScreenshotAnnotate(img *image.RGBA, e *Editor, pixW, pixH, editRows int, cellH uint) {
	lineH := int(float64(cellH) * bookLineHeightMul)
	marginTop := int(float64(pixH) * bookMarginTop)
	marginBottom := int(float64(pixH) * bookMarginBottom)
	marginLeft := int(float64(pixW) * bookMarginLeft)
	marginRight := pixW - int(float64(pixW)*bookMarginRight)

	maxLines := min((pixH-marginTop-marginBottom)/lineH, editRows)

	guideClr := color.NRGBA{0xFF, 0x00, 0x00, 0x60} // semi-transparent red

	// Horizontal guides at each row boundary.
	for row := 0; row <= maxLines; row++ {
		y := marginTop + row*lineH
		if y >= pixH {
			break
		}
		for x := marginLeft; x < marginRight; x += 2 { // dashed
			img.Set(x, y, guideClr)
		}
	}

	// Mark cursor position with a small crosshair.
	fontSize := float64(cellH) * 0.72
	if fontSize < 6 {
		fontSize = 6
	}
	fs, err := bookFaces(fontSize)
	if err != nil {
		return
	}

	ascent := faceAscent(fs.regular, fontSize)
	textW := marginRight - marginLeft

	cursorDataY := int(e.DataY())
	cursorRawX := e.pos.sx + e.pos.offsetX
	startLine := e.pos.offsetY
	totalLines := e.Len()

	// Compute cursor display row using pixel-based counting.
	inFence := e.fenceStateAtLine(startLine)
	cursorDisplayRow := -1
	{
		dl := startLine
		for row := 0; row < maxLines && dl < totalLines; {
			rl := e.Line(LineIndex(dl))
			rl = strings.ReplaceAll(rl, "\t", "    ")
			if isFencedCodeMarker(rl) {
				inFence = !inFence
				if dl == cursorDataY {
					cursorDisplayRow = row
					break
				}
				row++
				dl++
				continue
			}
			pl := parseBookLine(rl)
			if inFence {
				pl = parsedLine{kind: lineKindCode, body: rl}
			}
			if dl == cursorDataY {
				cursorDisplayRow = row
				break
			}
			row += bookPixelRowCount(fs, pl, lineH, marginLeft, marginRight)
			dl++
		}
	}
	if cursorDisplayRow < 0 {
		return
	}

	// Compute cursor pixel X.
	rawLine := e.Line(LineIndex(cursorDataY))
	rawLine = strings.ReplaceAll(rawLine, "\t", "    ")
	pl := parseBookLine(rawLine)
	var cursorPx int
	switch pl.kind {
	case lineKindHeader:
		hFace := fs.headerForLevel(pl.headerLevel)
		prefixLen := pl.headerLevel + 1
		adjRawX := max(cursorRawX-prefixLen, 0)
		bodyRunes := []rune(pl.body)
		if adjRawX > len(bodyRunes) {
			adjRawX = len(bodyRunes)
		}
		cursorPx = marginLeft + measureStringFB(hFace, string(bodyRunes[:adjRawX])).Round()
	default:
		visX := rawXToVisualX(rawLine, cursorRawX)
		_ = visX
		cursorPx = marginLeft + measureStringFB(fs.regular, string([]rune(rawLine)[:min(cursorRawX, len([]rune(rawLine)))])).Round()
	}

	cursorY := marginTop + cursorDisplayRow*lineH + (lineH-ascent)/2 + ascent/2

	// Draw crosshair.
	crossClr := color.NRGBA{0xFF, 0x00, 0x00, 0xFF}
	for dx := -4; dx <= 4; dx++ {
		img.Set(cursorPx+dx, cursorY, crossClr)
	}
	for dy := -4; dy <= 4; dy++ {
		img.Set(cursorPx, cursorY+dy, crossClr)
	}

	// Label: row number and cursor position.
	fmt.Printf("  Cursor: dataY=%d rawX=%d displayRow=%d cursorPx=%d cursorY=%d\n",
		cursorDataY, cursorRawX, cursorDisplayRow, cursorPx, cursorY)
	fmt.Printf("  Layout: marginTop=%d lineH=%d ascent=%d textW=%d\n",
		marginTop, lineH, ascent, textW)

	// Print row map for debugging.
	inFence2 := e.fenceStateAtLine(startLine)
	dl := startLine
	for row := 0; row < maxLines && dl < totalLines; {
		rl := e.Line(LineIndex(dl))
		rl = strings.ReplaceAll(rl, "\t", "    ")
		if isFencedCodeMarker(rl) {
			inFence2 = !inFence2
			fmt.Printf("  Row %2d: doc=%d (fence marker)\n", row, dl)
			row++
			dl++
			continue
		}
		pl2 := parseBookLine(rl)
		if inFence2 {
			pl2 = parsedLine{kind: lineKindCode, body: rl}
		}
		nrows := bookPixelRowCount(fs, pl2, lineH, marginLeft, marginRight)
		marker := " "
		if dl == cursorDataY {
			marker = "*"
		}
		preview := rl
		if len(preview) > 50 {
			preview = preview[:50] + "…"
		}
		fmt.Printf("  Row %2d: doc=%d rows=%d kind=%d %s %q\n", row, dl, nrows, pl2.kind, marker, preview)
		row += nrows
		dl++
	}
}

// bookParagraphStart returns the LineIndex of the first line of the
// paragraph containing y. A paragraph is a maximal run of non-blank lines.
// If y itself is blank, returns y unchanged.
func (e *Editor) bookParagraphStart(y LineIndex) LineIndex {
	if strings.TrimSpace(e.Line(y)) == "" {
		return y
	}
	for y > 0 && strings.TrimSpace(e.Line(y-1)) != "" {
		y--
	}
	return y
}

// bookParagraphEnd returns the LineIndex of the last line of the paragraph
// containing y (inclusive). A paragraph is a maximal run of non-blank lines.
// If y itself is blank, returns y unchanged.
func (e *Editor) bookParagraphEnd(y LineIndex) LineIndex {
	if strings.TrimSpace(e.Line(y)) == "" {
		return y
	}
	last := LineIndex(e.Len() - 1)
	for y < last && strings.TrimSpace(e.Line(y+1)) != "" {
		y++
	}
	return y
}

// bookToggleParagraphIndent toggles a 4-space indent on the first line of
// the paragraph containing the cursor. If the first line already starts
// with 4 spaces, the leading 4 spaces are removed; otherwise 4 spaces are
// prepended. When the edit happens on the cursor's own line, the cursor
// shifts by 4 raw columns to stay on the same character.
func (e *Editor) bookToggleParagraphIndent(c *vt.Canvas) {
	cursorY := e.DataY()
	startY := e.bookParagraphStart(cursorY)
	if strings.TrimSpace(e.Line(startY)) == "" {
		return
	}
	line := e.Line(startY)
	indented := strings.HasPrefix(line, "    ")
	var newLine string
	delta := 0
	if indented {
		newLine = line[4:]
		delta = -4
	} else {
		newLine = "    " + line
		delta = 4
	}
	e.SetLine(startY, newLine)
	e.MarkChanged()
	bookBumpContentGen()
	if startY == cursorY {
		newX := max(e.pos.sx+e.pos.offsetX+delta, 0)
		e.pos.SetX(c, newX)
	}
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
}

// bookSplitTableCells splits a raw Markdown table row like "| a | b | c |"
// into the trimmed cell strings ["a", "b", "c"]. Leading and trailing
// pipes are stripped. Pipes escaped with a backslash ("\|") are literal.
func bookSplitTableCells(row string) []string {
	row = strings.TrimSpace(row)
	if strings.HasPrefix(row, "|") {
		row = row[1:]
	}
	if strings.HasSuffix(row, "|") {
		row = row[:len(row)-1]
	}
	var cells []string
	var cur strings.Builder
	runes := []rune(row)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\\' && i+1 < len(runes) && runes[i+1] == '|' {
			cur.WriteRune('|')
			i++
			continue
		}
		if r == '|' {
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteRune(r)
	}
	cells = append(cells, strings.TrimSpace(cur.String()))
	return cells
}

// bookIsTableSeparator reports whether the given raw table row is a
// separator like "|---|:-:|---:|". Each cell must contain only "-", ":"
// and spaces, and have at least one "-".
func bookIsTableSeparator(row string) bool {
	cells := bookSplitTableCells(row)
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		c = strings.TrimSpace(c)
		if c == "" {
			return false
		}
		hasDash := false
		for _, r := range c {
			switch r {
			case '-':
				hasDash = true
			case ':', ' ':
			default:
				return false
			}
		}
		if !hasDash {
			return false
		}
	}
	return true
}

const (
	tableAlignLeft   = 0
	tableAlignCenter = 1
	tableAlignRight  = 2
)

// bookTableCellAlign returns the alignment implied by a separator cell
// such as ":---", "---:", ":-:" or "---".
func bookTableCellAlign(cell string) int {
	cell = strings.TrimSpace(cell)
	left := strings.HasPrefix(cell, ":")
	right := strings.HasSuffix(cell, ":")
	switch {
	case left && right:
		return tableAlignCenter
	case right:
		return tableAlignRight
	default:
		return tableAlignLeft
	}
}

// bookTableLayout returns per-column alignments and equal-division pixel
// widths (summing to exactly avail) for a table block. Column count is
// the maximum across non-separator rows. Widths are distributed equally
// regardless of content length so all rows align; overflow within a cell
// is handled by wrapping in bookDrawTableRow.
func bookTableLayout(fs *bookFontSet, block []string, avail int) (aligns []int, widths []int) {
	nCols := 0
	rows := make([][]string, len(block))
	for i, r := range block {
		rows[i] = bookSplitTableCells(r)
		if len(rows[i]) > nCols {
			nCols = len(rows[i])
		}
	}
	aligns = make([]int, nCols)
	widths = make([]int, nCols)
	for i, r := range block {
		if bookIsTableSeparator(r) {
			for c := 0; c < nCols; c++ {
				if c < len(rows[i]) {
					aligns[c] = bookTableCellAlign(rows[i][c])
				}
			}
			break
		}
	}
	if nCols == 0 || avail <= 0 {
		return aligns, widths
	}
	// Reserve a single pixel column for the right border. The outer
	// renderer paints a white rectangle starting at rightMargin after
	// the table is drawn; if the table's right vertical line were to
	// land exactly on rightMargin that fill would overwrite it and the
	// table would appear to be missing its right border. Keeping the
	// table one pixel narrower than avail guarantees the border sits at
	// rightMargin-1 and survives.
	layoutAvail := avail
	if layoutAvail > 1 {
		layoutAvail--
	}
	each := layoutAvail / nCols
	for i := range widths {
		widths[i] = each
	}
	widths[nCols-1] += layoutAvail - each*nCols
	return aligns, widths
}

// bookTableRowHeight returns the number of pixel sub-rows a single table
// row needs when each cell is wrapped into its equal-width column.
// Separator rows take zero rows — the top border of the row below (and
// the bottom border of the header above) already paint the visual divider,
// so rendering a separate empty row just adds dead space.
func bookTableRowHeight(fs *bookFontSet, body string, marginLeft, rightMargin int) int {
	if bookIsTableSeparator(body) {
		return 0
	}
	cells := bookSplitTableCells(body)
	if len(cells) == 0 {
		return 1
	}
	avail := rightMargin - marginLeft
	if avail <= 0 {
		return 1
	}
	// Match bookTableLayout, which reserves one pixel for the right border.
	if avail > 1 {
		avail--
	}
	each := avail / len(cells)
	const pad = 10
	inner := max(each-2*pad, 1)
	maxSub := 1
	for _, cell := range cells {
		rows := bookWrapSegmentsPixel(fs, cell, inner)
		if len(rows) > maxSub {
			maxSub = len(rows)
		}
	}
	return maxSub
}

// bookDrawTableRow draws a single table row inside a block using the
// pre-computed column widths and alignments. Long cells are soft-wrapped
// to additional sub-rows; rowPixelH is the total vertical space this row
// occupies (typically subRows*lineH, from bookTableRowHeight). Header rows
// render with a dark background (#101010) and white sans-serif text; odd
// data rows use altBg for readability; inline `code`, *italic* and **bold**
// are parsed per cell.
func bookDrawTableRow(img *image.RGBA, fs *bookFontSet, rawRow string, block []string,
	rowInBlock, headerRow int, aligns, widths []int,
	marginLeft, rightMargin, cellTop, lineH, rowPixelH int, fg, altBg, headerBg, headerFg color.Color) {
	if len(widths) == 0 || rowPixelH <= 0 {
		return
	}
	const pad = 10
	colW := make([]int, len(widths))
	copy(colW, widths)
	totalW := 0
	for _, w := range colW {
		totalW += w
	}
	isSep := bookIsTableSeparator(rawRow)
	isHeader := rowInBlock == headerRow && headerRow >= 0

	// Data-row index (skips separators and the header) — used for the
	// alternating background band
	dataIdx := -1
	if !isSep && !isHeader {
		n := 0
		for i := 0; i < rowInBlock && i < len(block); i++ {
			if i == headerRow {
				continue
			}
			if bookIsTableSeparator(block[i]) {
				continue
			}
			n++
		}
		dataIdx = n
	}

	// Header face: Montserrat Bold at the body size, falling back to
	// the regular body face if it can't be loaded
	headerBoldFace := fs.regular
	if parsedMontserratBold != nil {
		if hf, err := newFace(parsedMontserratBold, fs.baseSize*0.9); err == nil {
			headerBoldFace = hf
		}
	}

	if isHeader {
		draw.Draw(img, image.Rect(marginLeft, cellTop, marginLeft+totalW, cellTop+rowPixelH),
			image.NewUniform(headerBg), image.Point{}, draw.Src)
	} else if dataIdx >= 0 && dataIdx%2 == 1 {
		draw.Draw(img, image.Rect(marginLeft, cellTop, marginLeft+totalW, cellTop+rowPixelH),
			image.NewUniform(altBg), image.Point{}, draw.Src)
	}

	borderClr := color.NRGBA{0x88, 0x88, 0x88, 0xff}
	right := marginLeft + totalW
	drawHLine := func(y int) {
		for x := marginLeft; x <= right; x++ {
			img.Set(x, y, borderClr)
		}
	}
	drawVLine := func(x int) {
		for y := cellTop; y <= cellTop+rowPixelH; y++ {
			img.Set(x, y, borderClr)
		}
	}
	// Top + bottom borders on every row so the divider between adjacent
	// rows survives the next row's background paint
	drawHLine(cellTop)
	drawHLine(cellTop + rowPixelH)
	drawVLine(marginLeft)
	x := marginLeft
	for _, w := range colW {
		x += w
		drawVLine(x)
	}
	if isSep {
		return
	}

	cells := bookSplitTableCells(rawRow)
	ascent := faceAscent(fs.regular, float64(lineH)*0.72)
	descent := fs.regular.Metrics().Descent.Round()
	if descent <= 0 {
		descent = int(float64(lineH)*0.72*0.2 + 0.5)
	}
	// Shift text slightly upward so descenders clear the bottom row line
	// by a few pixels instead of sitting right on it
	const cellBottomMargin = 3
	vPad := max((lineH-ascent-descent)/2, 0)
	textClr := fg
	if isHeader {
		textClr = headerFg
	}
	cellX := marginLeft
	for i, w := range colW {
		var text string
		if i < len(cells) {
			text = cells[i]
		}
		align := tableAlignLeft
		if i < len(aligns) {
			align = aligns[i]
		}
		inner := max(w-2*pad, 1)
		// Wrap the cell to the column's inner width
		var rows []wrappedRow
		if isHeader {
			// Header uses Montserrat Bold uniformly, so wrap by measuring
			// that face directly instead of via parseLineSegments
			rows = bookWrapPlainPixel(headerBoldFace, text, inner)
		} else {
			rows = bookWrapSegmentsPixel(fs, text, inner)
		}
		if len(rows) == 0 {
			rows = []wrappedRow{{}}
		}
		for si, wr := range rows {
			subBaseline := cellTop + si*lineH + vPad + ascent - cellBottomMargin
			var tW int
			if isHeader {
				tW = measureStringFB(headerBoldFace, wr.plainText).Round()
			} else {
				tW = measureSegmentsWidth(fs, wr.segs)
			}
			var tx int
			switch align {
			case tableAlignCenter:
				tx = cellX + (w-tW)/2
			case tableAlignRight:
				tx = cellX + w - tW - pad
			default:
				tx = cellX + pad
			}
			if isHeader {
				drawString(img, headerBoldFace, tx, subBaseline, wr.plainText, textClr)
			} else {
				drawSegments(img, fs, tx, subBaseline, wr.segs, textClr)
			}
		}
		cellX += w
	}
}

// measureSegmentsWidth returns the total pixel width of segs using fs
func measureSegmentsWidth(fs *bookFontSet, segs []textSegment) int {
	total := fixed.Int26_6(0)
	for _, seg := range segs {
		face := faceForSeg(fs, seg)
		for _, r := range seg.text {
			adv, _ := faceGlyphAdvance(face, r)
			total += adv
			if seg.bold {
				total += fixed.I(1)
			}
		}
	}
	return total.Round()
}

// bookWrapPlainPixel wraps a plain (non-segmented) string to fit within
// availPx using the given face. Rows carry plainText; segs is left unset.
func bookWrapPlainPixel(face font.Face, body string, availPx int) []wrappedRow {
	if availPx <= 0 {
		availPx = 1
	}
	runes := []rune(body)
	if len(runes) == 0 {
		return []wrappedRow{{plainText: ""}}
	}
	var rows []wrappedRow
	pos := 0
	for pos < len(runes) {
		rowStart := pos
		x := fixed.Int26_6(0)
		lastSpace := -1
		end := pos
		for end < len(runes) {
			adv, _ := faceGlyphAdvance(face, runes[end])
			if (x+adv).Round() > availPx && end > rowStart {
				break
			}
			x += adv
			if runes[end] == ' ' && (end == rowStart || runes[end-1] != ' ') {
				lastSpace = end
			}
			end++
		}
		if end < len(runes) && lastSpace > rowStart {
			end = lastSpace + 1
		}
		rows = append(rows, wrappedRow{plainText: string(runes[rowStart:end])})
		pos = end
	}
	return rows
}
