package megafile

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "github.com/dkua/go-ico"
	_ "github.com/xfmoulet/qoi"
	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
)

// envKitty is true when TERM=xterm-kitty, enabling Kitty graphics protocol features.
var envKitty = env.Str("TERM") == "xterm-kitty"

// imageExts lists file extensions handled by the image preview path.
var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".svg": true, ".qoi": true,
	".ico": true, ".jxl": true,
}

func isImageExt(path string) bool {
	return imageExts[strings.ToLower(filepath.Ext(path))]
}

// previewResult holds a fully-prepared image preview ready to be displayed.
// It is produced by loadImageAsync and consumed on the main goroutine.
// encoded is always base64-encoded PNG data (Kitty format f=100).
type previewResult struct {
	path    string
	encoded string // base64-encoded PNG
	imgW    uint   // pixel width of the source image
	imgH    uint   // pixel height of the source image
}

// kittyDeleteImages sends the Kitty graphics protocol command to delete all placed images.
func kittyDeleteImages() {
	fmt.Fprintf(os.Stdout, "\033_Ga=d,d=A,q=2\033\\")
}

// previewPaneBounds returns the 1-indexed terminal column/row and the cell dimensions
// (cols, rows) of the preview pane.  A separator column sits at W/2 (canvas, 0-indexed),
// so the preview starts one column to the right of it.
func (s *State) previewPaneBounds() (col, row, cols, rows uint) {
	W := s.canvas.W()
	H := s.canvas.H()
	half := W / 2
	col = half + 2
	row = s.starty + 2
	if W > half+1 {
		cols = W - half - 1
	}
	const bottomMargin = 2
	if H > s.starty+bottomMargin+1 {
		rows = H - s.starty - bottomMargin - 1
	}
	return
}

// cancelPreviewLoad cancels any in-flight image loading goroutine.
func (s *State) cancelPreviewLoad() {
	if s.previewCancel != nil {
		s.previewCancel()
		s.previewCancel = nil
	}
}

// clearPreviewPane erases the preview pane area, deletes any Kitty graphics images,
// and cancels any in-flight image loading goroutine.
func (s *State) clearPreviewPane() {
	s.cancelPreviewLoad()
	kittyDeleteImages()
	col, row, cols, rows := s.previewPaneBounds()
	if cols == 0 || rows == 0 {
		return
	}
	blank := strings.Repeat(" ", int(cols))
	for r := range rows {
		fmt.Fprintf(os.Stdout, "\033[%d;%dH%s", row+r, col, blank)
	}
	s.currentPreviewPath = ""
	s.currentPreviewEncoded = ""

	s.currentPreviewImgW = 0
	s.currentPreviewImgH = 0
}

// loadImageAsync decodes an image file, re-encodes it as PNG, base64-encodes the result,
// and sends it to s.previewResultChan. The goroutine checks ctx.Done() between the
// expensive steps so that fast navigation can cancel stale loads before they write anything.
// This function must be called as a goroutine; it never writes to stdout.
//
// PNG files are sent directly (no re-encode needed). All other formats (JPEG, GIF, …)
// are decoded and re-encoded as PNG because the Kitty graphics protocol only supports
// raw pixel data and PNG (f=100); it has no native JPEG support.
func (s *State) loadImageAsync(ctx context.Context, path string, panePixW, panePixH uint) {
	ext := strings.ToLower(filepath.Ext(path))

	var encoded string
	var imgW, imgH uint

	if ext == ".svg" {
		// SVG: render via inkscape at the pane's pixel dimensions.
		w := panePixW
		if w == 0 {
			w = 800
		}
		enc, iw, ih, err := convertToPNG(ctx, "inkscape",
			"--export-type=png", "--export-filename=-",
			"--export-area-drawing", "--export-width="+fmt.Sprintf("%d", w), path)
		if err != nil || ctx.Err() != nil {
			return
		}
		encoded, imgW, imgH = enc, iw, ih
	} else if ext == ".jxl" {
		// JPEG XL: convert via ImageMagick.
		enc, iw, ih, err := convertToPNG(ctx, "magick", path, "png:-")
		if err != nil || ctx.Err() != nil {
			return
		}
		encoded, imgW, imgH = enc, iw, ih
	} else {
		// Standard bitmap formats via Go's image package.
		f, err := os.Open(path)
		if err != nil {
			return
		}
		defer f.Close()

		// Use DecodeConfig to read dimensions from the header cheaply.
		config, format, err := image.DecodeConfig(f)
		if err != nil || ctx.Err() != nil {
			return
		}
		imgW = uint(config.Width)
		imgH = uint(config.Height)

		if format == "png" && ext == ".png" {
			// PNG can be forwarded verbatim — Kitty accepts it as f=100.
			data, err := os.ReadFile(path)
			if err != nil || ctx.Err() != nil {
				return
			}
			encoded = base64.StdEncoding.EncodeToString(data)
		} else {
			// JPEG, GIF: decode and re-encode as PNG.
			if _, err := f.Seek(0, 0); err != nil {
				return
			}
			img, _, err := image.Decode(f)
			if err != nil || ctx.Err() != nil {
				return
			}
			var buf bytes.Buffer
			if err := png.Encode(&buf, img); err != nil || ctx.Err() != nil {
				return
			}
			encoded = base64.StdEncoding.EncodeToString(buf.Bytes())
		}
	}

	if ctx.Err() != nil {
		return
	}
	result := previewResult{path: path, encoded: encoded, imgW: imgW, imgH: imgH}
	select {
	case s.previewResultChan <- result:
	case <-ctx.Done():
	}
}

// convertToPNG runs an external command that writes PNG data to stdout,
// base64-encodes the result, and returns the encoded string with pixel dimensions.
func convertToPNG(ctx context.Context, args ...string) (encoded string, w, h uint, err error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err = cmd.Run(); err != nil {
		return
	}
	cfg, _, cfgErr := image.DecodeConfig(bytes.NewReader(buf.Bytes()))
	if cfgErr != nil {
		err = cfgErr
		return
	}
	encoded = base64.StdEncoding.EncodeToString(buf.Bytes())
	w, h = uint(cfg.Width), uint(cfg.Height)
	return
}

// applyPreviewResult stores the data from a finished async load into the state cache.
// Returns true if the result belongs to the currently selected file.
func (s *State) applyPreviewResult(result previewResult) bool {
	if result.path != s.currentPreviewPath {
		return false
	}
	s.currentPreviewEncoded = result.encoded
	s.currentPreviewImgW = result.imgW
	s.currentPreviewImgH = result.imgH
	return true
}

// flushImageFromCache writes the cached PNG image to the preview pane using the
// Kitty graphics protocol (f=100). The caller must ensure currentPreviewEncoded is non-empty.
func (s *State) flushImageFromCache(col, row, cols, rows uint) {
	dispCols, dispRows := aspectRatioCells(s.currentPreviewImgW, s.currentPreviewImgH, cols, rows)

	fmt.Fprintf(os.Stdout, "\033[%d;%dH", row, col)

	encoded := s.currentPreviewEncoded
	const chunkSize = 4096
	total := len(encoded)
	for i := 0; i < total; i += chunkSize {
		end := min(i+chunkSize, total)
		chunk := encoded[i:end]
		isLast := end >= total
		isFirst := i == 0

		switch {
		case isFirst && isLast:
			fmt.Fprintf(os.Stdout, "\033_Ga=T,f=100,q=2,c=%d,r=%d;%s\033\\", dispCols, dispRows, chunk)
		case isFirst:
			fmt.Fprintf(os.Stdout, "\033_Ga=T,f=100,q=2,m=1,c=%d,r=%d;%s\033\\", dispCols, dispRows, chunk)
		case isLast:
			fmt.Fprintf(os.Stdout, "\033_Gm=0;%s\033\\", chunk)
		default:
			fmt.Fprintf(os.Stdout, "\033_Gm=1;%s\033\\", chunk)
		}
	}
}

// showPreview updates the preview pane for the given file path.
// For image files it returns immediately after starting a background goroutine
// (unless the image is already cached). The goroutine sends its result to
// s.previewResultChan, which is consumed by the main event loop.
// Non-image previews (text, directory, binary) are rendered synchronously.
func (s *State) showPreview(path string) {
	if !envKitty {
		return
	}
	col, row, cols, rows := s.previewPaneBounds()
	if cols == 0 || rows == 0 {
		return
	}

	if path != s.currentPreviewPath {
		// New file selected: cancel any stale load and clear the pane.
		s.cancelPreviewLoad()
		kittyDeleteImages()
		blank := strings.Repeat(" ", int(cols))
		for r := range rows {
			fmt.Fprintf(os.Stdout, "\033[%d;%dH%s", row+r, col, blank)
		}
		s.currentPreviewPath = path
		s.currentPreviewEncoded = ""

		s.currentPreviewImgW = 0
		s.currentPreviewImgH = 0
	}

	switch {
	case files.IsDir(path):
		s.drawDirPreview(path, col, row, cols, rows)
	case isImageExt(path):
		if s.currentPreviewEncoded != "" {
			// Cache hit: write immediately (no goroutine needed).
			s.flushImageFromCache(col, row, cols, rows)
		} else if s.previewCancel == nil {
			// No goroutine running yet: start one.
			cellW, cellH := terminalCellPixels()
			ctx, cancel := context.WithCancel(context.Background())
			s.previewCancel = cancel
			go s.loadImageAsync(ctx, path, cols*cellW, rows*cellH)
		}
		// If previewCancel != nil and encoded == "": goroutine is already running; wait.
	case !files.BinaryAccurate(path):
		s.drawTextPreview(path, col, row, cols, rows)
	default:
		s.drawBinaryPreview(path, col, row, cols)
	}
}

// redrawPreview refreshes the preview pane to match the current selection state.
// Call this after every c.Draw() to restore preview content erased by the canvas flush.
func (s *State) redrawPreview() {
	if !envKitty {
		return
	}
	if s.selectedIndex() >= 0 && s.selectedIndex() < len(s.fileEntries) {
		if path, err := s.selectedPath(); err == nil {
			s.showPreview(path)
		}
	} else if s.currentPreviewPath != "" {
		s.clearPreviewPane()
	}
}

// aspectRatioCells computes the display size in terminal cells that best fits the given
// image (imgW×imgH pixels) inside the available pane (availCols×availRows cells) while
// preserving the image's pixel aspect ratio.
func aspectRatioCells(imgW, imgH, availCols, availRows uint) (cols, rows uint) {
	if imgW == 0 || imgH == 0 || availCols == 0 || availRows == 0 {
		return availCols, availRows
	}
	cellW, cellH := terminalCellPixels()
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

// drawTextPreview renders the first rows lines of a text file into the preview pane.
func (s *State) drawTextPreview(path string, col, row, cols, rows uint) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for r := uint(0); r < rows && scanner.Scan(); r++ {
		line := scanner.Text()
		line = strings.ReplaceAll(line, "\t", "    ")
		runes := []rune(line)
		if uint(len(runes)) >= cols {
			runes = runes[:cols-1]
		}
		fmt.Fprintf(os.Stdout, "\033[%d;%dH%s", row+r, col, string(runes))
	}
}

// drawDirPreview lists the visible contents of a directory in the preview pane.
func (s *State) drawDirPreview(path string, col, row, cols, rows uint) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}
	r := uint(0)
	for _, e := range entries {
		if r >= rows {
			break
		}
		if !s.ShowHidden && strings.HasPrefix(e.Name(), ".") {
			continue
		}
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		runes := []rune(name)
		if uint(len(runes)) >= cols {
			runes = runes[:cols-1]
		}
		fmt.Fprintf(os.Stdout, "\033[%d;%dH%s", row+r, col, string(runes))
		r++
	}
}

// drawBinaryPreview shows a one-line description for a binary file.
func (s *State) drawBinaryPreview(path string, col, row, cols uint) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	line := fmt.Sprintf("Binary file (%d bytes)", info.Size())
	runes := []rune(line)
	if uint(len(runes)) >= cols {
		runes = runes[:cols-1]
	}
	fmt.Fprintf(os.Stdout, "\033[%d;%dH%s", row, col, string(runes))
}
