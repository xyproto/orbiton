package main

import (
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xyproto/burnfont"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// SystemFonts holds file paths for each font role used in graphical book mode.
// A field is empty when no suitable font was found for that role.
type SystemFonts struct {
	Regular string // serif, regular -- body text
	Italic  string // serif, italic -- body emphasis
	Bold    string // sans-serif, bold -- headings
	Light   string // sans-serif, light/regular -- status bar
	Mono    string // monospace, bold -- inline code and code blocks
	Unicode string // wide-coverage sans -- unicode glyph fallback (optional)
}

// Any reports whether at least one font path was resolved.
func (sf *SystemFonts) Any() bool {
	return sf.Regular != "" || sf.Italic != "" || sf.Bold != "" ||
		sf.Light != "" || sf.Mono != ""
}

// FindSystemFonts searches the host for a set of OTF/TTF fonts suitable for
// book mode. On Linux and BSD it uses fontconfig (fc-match) when available.
// On macOS fontconfig is tried first. On Windows the system Fonts directory
// is checked. A directory scan is the last resort when nothing else succeeds.
func FindSystemFonts() *SystemFonts {
	sf := &SystemFonts{}
	if isLinux || isBSD || isDarwin {
		fillViaFcMatch(sf)
	}
	if isWindows {
		fillViaWindowsPaths(sf)
	}
	if !sf.Any() {
		fillViaDirScan(sf)
	}
	return sf
}

// fillViaFcMatch populates sf using fc-match. A small number of preferred
// families are tried first; a generic fontconfig query (serif / sans /
// monospace) is used as the last resort for each still-empty role.
// The wantFamily guard prevents accepting an unrelated fontconfig fallback.
func fillViaFcMatch(sf *SystemFonts) {
	type q struct {
		dest       *string
		pattern    string
		wantFamily string // when non-empty the returned family must contain this
	}

	// Georgia is preferred for serif: it was designed for screen readability
	// and is widely available. EB Garamond is an elegant alternative. Liberation
	// Serif is metrically compatible with Times New Roman and very common on
	// Linux/BSD. Gentium Plus has excellent language coverage.
	// Cantarell is the GNOME default and has a true Light weight.
	// Source Code Pro Bold and Fira Code Bold are the top code-font choices.
	// Generic fallbacks let fontconfig pick the distro's configured default.
	queries := []q{
		{pattern: "Georgia:style=Regular", wantFamily: "Georgia", dest: &sf.Regular},
		{pattern: "EB Garamond:style=Regular", wantFamily: "EB Garamond", dest: &sf.Regular},
		{pattern: "Liberation Serif:style=Regular", wantFamily: "Liberation Serif", dest: &sf.Regular},
		{pattern: "Gentium Plus:style=Regular", wantFamily: "Gentium", dest: &sf.Regular},
		{pattern: "Noto Serif:style=Regular", wantFamily: "Noto Serif", dest: &sf.Regular},
		{pattern: "serif", dest: &sf.Regular},

		{pattern: "Georgia:style=Italic", wantFamily: "Georgia", dest: &sf.Italic},
		{pattern: "EB Garamond:style=Italic", wantFamily: "EB Garamond", dest: &sf.Italic},
		{pattern: "Liberation Serif:style=Italic", wantFamily: "Liberation Serif", dest: &sf.Italic},
		{pattern: "Gentium Plus:style=Italic", wantFamily: "Gentium", dest: &sf.Italic},
		{pattern: "Noto Serif:style=Italic", wantFamily: "Noto Serif", dest: &sf.Italic},
		{pattern: "serif:slant=italic", dest: &sf.Italic},

		{pattern: "Cantarell:style=Bold", wantFamily: "Cantarell", dest: &sf.Bold},
		{pattern: "Source Sans Pro:style=Bold", wantFamily: "Source Sans", dest: &sf.Bold},
		{pattern: "Liberation Sans:style=Bold", wantFamily: "Liberation Sans", dest: &sf.Bold},
		{pattern: "sans:bold", dest: &sf.Bold},

		{pattern: "Cantarell:style=Light", wantFamily: "Cantarell", dest: &sf.Light},
		{pattern: "Cantarell:style=Regular", wantFamily: "Cantarell", dest: &sf.Light},
		{pattern: "Source Sans Pro:style=Light", wantFamily: "Source Sans", dest: &sf.Light},
		{pattern: "Liberation Sans:style=Regular", wantFamily: "Liberation Sans", dest: &sf.Light},
		{pattern: "sans", dest: &sf.Light},

		{pattern: "Source Code Pro:style=Bold", wantFamily: "Source Code Pro", dest: &sf.Mono},
		{pattern: "Fira Code:style=Bold", wantFamily: "Fira Code", dest: &sf.Mono},
		{pattern: "monospace:bold", dest: &sf.Mono},
		{pattern: "monospace", dest: &sf.Mono},

		{pattern: "DejaVu Sans:style=Book", wantFamily: "DejaVu Sans", dest: &sf.Unicode},
		{pattern: "sans", dest: &sf.Unicode},
	}
	for _, query := range queries {
		if *query.dest != "" {
			continue
		}
		if p := fcMatchFile(query.pattern, query.wantFamily); p != "" {
			*query.dest = p
		}
	}
}

// fcMatchFile runs fc-match and returns the resolved font file path. When
// wantFamily is non-empty the result is rejected if the returned family name
// does not contain it (case-insensitive), guarding against silent fontconfig
// substitutions for unavailable families.
func fcMatchFile(pattern, wantFamily string) string {
	out, err := exec.Command("fc-match", "--format=%{family}|%{file}", pattern).Output()
	if err != nil {
		return ""
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), "|", 2)
	if len(parts) != 2 {
		return ""
	}
	gotFamily, path := parts[0], parts[1]
	if wantFamily != "" && !strings.Contains(strings.ToLower(gotFamily), strings.ToLower(wantFamily)) {
		return "" // fontconfig substituted an unrelated family -- reject
	}
	return validFontPath(path)
}

// fillViaWindowsPaths checks the default Windows font directory for a curated
// set of filenames present in typical Windows 10/11 installations.
func fillViaWindowsPaths(sf *SystemFonts) {
	winDir := os.Getenv("WINDIR")
	if winDir == "" {
		winDir = `C:\Windows`
	}
	dir := filepath.Join(winDir, "Fonts")
	type role struct {
		dest  *string
		names []string
	}
	for _, r := range []role{
		{dest: &sf.Regular, names: []string{"georgia.ttf", "times.ttf"}},
		{dest: &sf.Italic, names: []string{"georgiai.ttf", "timesi.ttf"}},
		{dest: &sf.Bold, names: []string{"segoeuib.ttf", "arialbd.ttf", "calibrib.ttf"}},
		{dest: &sf.Light, names: []string{"segoeui.ttf", "arial.ttf", "calibri.ttf"}},
		{dest: &sf.Mono, names: []string{"consolab.ttf", "consola.ttf", "courbd.ttf", "cour.ttf"}},
		{dest: &sf.Unicode, names: []string{"arialuni.ttf"}},
	} {
		if *r.dest != "" {
			continue
		}
		for _, name := range r.names {
			if p := validFontPath(filepath.Join(dir, name)); p != "" {
				*r.dest = p
				break
			}
		}
	}
}

// fillViaDirScan walks the standard font directories and matches files by
// lower-cased base-name fragments. Used when fontconfig is unavailable.
func fillViaDirScan(sf *SystemFonts) {
	type rule struct {
		dest  *string
		frags []string // all fragments must appear in the lower-cased basename
	}
	rules := []rule{
		{dest: &sf.Regular, frags: []string{"georgia"}},
		{dest: &sf.Regular, frags: []string{"ebgaramond", "regular"}},
		{dest: &sf.Regular, frags: []string{"liberationserif", "regular"}},
		{dest: &sf.Regular, frags: []string{"notoserif", "regular"}},
		{dest: &sf.Italic, frags: []string{"georgiai"}},
		{dest: &sf.Italic, frags: []string{"ebgaramond", "italic"}},
		{dest: &sf.Italic, frags: []string{"liberationserif", "italic"}},
		{dest: &sf.Italic, frags: []string{"notoserif", "italic"}},
		{dest: &sf.Bold, frags: []string{"cantarell", "bold"}},
		{dest: &sf.Bold, frags: []string{"sourcesans", "bold"}},
		{dest: &sf.Bold, frags: []string{"liberationsans", "bold"}},
		{dest: &sf.Light, frags: []string{"cantarell", "regular"}},
		{dest: &sf.Light, frags: []string{"sourcesans", "light"}},
		{dest: &sf.Light, frags: []string{"liberationsans", "regular"}},
		{dest: &sf.Mono, frags: []string{"sourcecodepro", "bold"}},
		{dest: &sf.Mono, frags: []string{"sourcecodepro", "regular"}},
		{dest: &sf.Unicode, frags: []string{"dejavusans.ttf"}},
	}
	for _, dir := range systemFontDirs() {
		_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || validFontPath(p) == "" {
				return nil
			}
			lower := strings.ToLower(filepath.Base(p))
			for i := range rules {
				if *rules[i].dest != "" {
					continue
				}
				ok := true
				for _, frag := range rules[i].frags {
					if !strings.Contains(lower, frag) {
						ok = false
						break
					}
				}
				if ok {
					*rules[i].dest = p
				}
			}
			return nil
		})
	}
}

// systemFontDirs returns the canonical font search directories for the host OS.
func systemFontDirs() []string {
	home, _ := os.UserHomeDir()
	switch {
	case isLinux:
		dirs := []string{"/usr/share/fonts", "/usr/local/share/fonts"}
		if home != "" {
			dirs = append(dirs,
				filepath.Join(home, ".local", "share", "fonts"),
				filepath.Join(home, ".fonts"))
		}
		return dirs
	case isBSD:
		dirs := []string{
			"/usr/local/share/fonts",
			"/usr/X11R7/lib/X11/fonts",
			"/usr/pkg/share/fonts",
		}
		if home != "" {
			dirs = append(dirs,
				filepath.Join(home, ".local", "share", "fonts"),
				filepath.Join(home, ".fonts"))
		}
		return dirs
	case isDarwin:
		dirs := []string{
			"/Library/Fonts",
			"/System/Library/Fonts",
			"/System/Library/Fonts/Supplemental",
		}
		if home != "" {
			dirs = append(dirs, filepath.Join(home, "Library", "Fonts"))
		}
		return dirs
	}
	return nil
}

// validFontPath returns p when it refers to a readable .otf, .ttf, or .ttc
// file, and returns the empty string otherwise.
func validFontPath(p string) string {
	lower := strings.ToLower(p)
	if !strings.HasSuffix(lower, ".otf") && !strings.HasSuffix(lower, ".ttf") &&
		!strings.HasSuffix(lower, ".ttc") {
		return ""
	}
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

// parseFontFile reads and parses an OTF, TTF, or TTC font file, returning the
// first (or only) font in the file. Returns nil, nil when path is empty.
func parseFontFile(path string) (*opentype.Font, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// ParseCollection handles both single-font files (TTF/OTF) and TTC
	// collections transparently, always giving the first font in the file.
	coll, err := opentype.ParseCollection(data)
	if err != nil {
		return nil, err
	}
	if coll.NumFonts() == 0 {
		return nil, nil
	}
	return coll.Font(0)
}

// newFaceOrBurn creates an opentype face from f at the given pixel size.
// When f is nil or face creation fails it returns a burnFace fallback,
// so callers always receive a non-nil font.Face without extra error handling.
func newFaceOrBurn(f *opentype.Font, size float64) font.Face {
	if f == nil {
		return burnFace{}
	}
	face, err := newFace(f, size)
	if err != nil {
		return burnFace{}
	}
	return face
}

// burnFace wraps burnfont as a font.Face. It is the last-resort fallback when
// no OTF/TTF fonts are found on the host. Glyphs are always 8x8 pixels.
type burnFace struct{}

const (
	burnW = 8 // glyph cell width in pixels
	burnH = 8 // total cell height (6 drawn rows + 2 px descent/leading)
	burnA = 6 // ascent above baseline in pixels
	burnD = 2 // descent below baseline in pixels
)

func (burnFace) Close() error                              { return nil }
func (burnFace) Kern(_, _ rune) fixed.Int26_6              { return 0 }
func (burnFace) GlyphAdvance(_ rune) (fixed.Int26_6, bool) { return fixed.I(burnW), true }

func (burnFace) Metrics() font.Metrics {
	return font.Metrics{
		Height:    fixed.I(burnH),
		Ascent:    fixed.I(burnA),
		Descent:   fixed.I(burnD),
		XHeight:   fixed.I(burnA / 2),
		CapHeight: fixed.I(burnA),
	}
}

func (burnFace) GlyphBounds(_ rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) {
	return fixed.Rectangle26_6{
		Min: fixed.Point26_6{X: 0, Y: fixed.I(-burnA)},
		Max: fixed.Point26_6{X: fixed.I(burnW), Y: fixed.I(burnD)},
	}, fixed.I(burnW), true
}

func (burnFace) Glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	// Draw in white so the NRGBA alpha channel acts as the compositing mask.
	img := image.NewNRGBA(image.Rect(0, 0, burnW, burnH))
	if err := burnfont.Draw(img, r, 0, 0, 255, 255, 255); err != nil {
		// Unknown glyph: return a transparent blank to preserve layout.
		return image.Rectangle{}, image.NewUniform(color.Transparent), image.Point{}, fixed.I(burnW), true
	}
	x := dot.X.Round()
	y := dot.Y.Round() - burnA // top of glyph cell relative to baseline
	return image.Rect(x, y, x+burnW, y+burnH), img, image.Point{}, fixed.I(burnW), true
}

// compile-time check that burnFace satisfies font.Face
var _ font.Face = burnFace{}
