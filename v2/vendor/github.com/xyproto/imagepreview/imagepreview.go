package imagepreview

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/palgen"
	"github.com/xyproto/vt"
)

var (
	// Sync is a global mutex to coordinate terminal output between text and graphics.
	Sync sync.Mutex

	// IsKitty is true when TERM=xterm-kitty, enabling Kitty graphics protocol features.
	IsKitty = env.Str("TERM") == "xterm-kitty"

	// IsITerm2 is true when running inside iTerm2 (which sets TERM_PROGRAM=iTerm.app).
	IsITerm2 = env.Str("TERM_PROGRAM") == "iTerm.app"

	// IsVT is true when TERM starts with vt100 or vt220.
	IsVT = strings.HasPrefix(env.Str("TERM"), "vt100") || strings.HasPrefix(env.Str("TERM"), "vt220")

	// HasGraphics is true when the terminal supports an inline image protocol.
	// It is disabled if TERM is vt100/vt220 or if NO_COLOR is set.
	HasGraphics = (IsKitty || IsITerm2) && !IsVT && !env.Bool("NO_COLOR")

	// ImageExts lists file extensions handled by the image preview path.
	ImageExts = map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".svg": true, ".qoi": true, ".ico": true, ".jxl": true,
		".bmp": true, ".webp": true,
	}

	// PaletteColorMap maps RGB values to VT100 attribute colors for fast lookup.
	PaletteColorMap map[[3]uint8]vt.AttributeColor
)

func init() {
	vtColors := []vt.AttributeColor{
		vt.Black, vt.Red, vt.Green, vt.Yellow,
		vt.Blue, vt.Magenta, vt.Cyan, vt.LightGray,
		vt.DarkGray, vt.LightRed, vt.LightGreen, vt.LightYellow,
		vt.LightBlue, vt.LightMagenta, vt.LightCyan, vt.White,
	}
	PaletteColorMap = make(map[[3]uint8]vt.AttributeColor, len(palgen.BasicPalette16))
	for i, rgb := range palgen.BasicPalette16 {
		if i < len(vtColors) {
			PaletteColorMap[rgb] = vtColors[i]
		}
	}
}

// IsImageExt checks if the file extension indicates a supported image file.
func IsImageExt(path string) bool {
	return ImageExts[strings.ToLower(filepath.Ext(path))]
}

// PreviewResult holds a fully-prepared image preview ready to be displayed.
// Encoded is always base64-encoded PNG data (Kitty format f=100).
type PreviewResult struct {
	Path    string
	Encoded string // base64-encoded PNG
	ImgW    uint   // pixel width of the source image
	ImgH    uint   // pixel height of the source image
}

// BeginSync locks the global Sync mutex and emits the terminal's begin synchronized
// update escape sequence. Pair this with EndSync.
func BeginSync() {
	Sync.Lock()
	vt.BeginSyncUpdate()
}

// EndSync emits the terminal's end synchronized update escape sequence and unlocks
// the global Sync mutex. Pair this with BeginSync.
func EndSync() {
	vt.EndSyncUpdate()
	Sync.Unlock()
}
