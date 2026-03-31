package megafile

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/xyproto/files"
	"github.com/xyproto/imagepreview"
	"github.com/xyproto/mode"
	"github.com/xyproto/syntax"
	"github.com/xyproto/vt"
)

// previewPaneBounds returns the 1-indexed terminal column/row and the cell dimensions
// (cols, rows) of the preview pane.  A separator column sits at splitX (canvas, 0-indexed),
// so the preview starts one column to the right of it.
func (s *State) previewPaneBounds() (col, row, cols, rows uint) {
	W := s.canvas.W()
	H := s.canvas.H()
	var half uint
	if s.showPreviewPane() && s.splitX > 0 {
		half = s.splitX
	} else {
		half = W / 2
	}
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
	imagepreview.DeleteInlineImages()
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

// loadImageAsync decodes an image file via imagepreview.LoadAndEncode and sends the
// result to s.previewResultChan. Must be called as a goroutine; never writes to stdout.
func (s *State) loadImageAsync(ctx context.Context, path string, panePixW, panePixH uint) {
	result, err := imagepreview.LoadAndEncode(ctx, path, panePixW, panePixH)
	if err != nil || ctx.Err() != nil {
		return
	}
	select {
	case s.previewResultChan <- result:
	case <-ctx.Done():
	}
}

// applyPreviewResult stores the data from a finished async load into the state cache.
// Returns true if the result belongs to the currently selected file.
func (s *State) applyPreviewResult(result imagepreview.PreviewResult) bool {
	if result.Path != s.currentPreviewPath {
		return false
	}
	s.currentPreviewEncoded = result.Encoded
	s.currentPreviewImgW = result.ImgW
	s.currentPreviewImgH = result.ImgH
	return true
}

// flushImageFromCache writes the cached PNG image to the preview pane using the
// Kitty graphics protocol (f=100) or iTerm2 inline image protocol.
// The caller must ensure currentPreviewEncoded is non-empty.
func (s *State) flushImageFromCache(col, row, cols, rows uint) {
	dispCols, dispRows := imagepreview.AspectRatioCells(s.currentPreviewImgW, s.currentPreviewImgH, cols, rows)
	fmt.Fprintf(os.Stdout, "\033[%d;%dH", row, col)
	imagepreview.FlushImage(os.Stdout, s.currentPreviewEncoded, dispCols, dispRows)
}

// showPreview updates the preview pane for the given file path.
// For image files it returns immediately after starting a background goroutine
// (unless the image is already cached). The goroutine sends its result to
// s.previewResultChan, which is consumed by the main event loop.
// Non-image previews (text, directory, binary) are rendered synchronously.
func (s *State) showPreview(path string) {
	if !s.showPreviewPane() {
		return
	}
	col, row, cols, rows := s.previewPaneBounds()
	if cols == 0 || rows == 0 {
		return
	}

	if path != s.currentPreviewPath {
		// New file selected: cancel any stale load and clear the pane.
		s.cancelPreviewLoad()
		imagepreview.DeleteInlineImages()
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
	case imagepreview.IsImageExt(path):
		if imagepreview.HasGraphics {
			if s.currentPreviewEncoded != "" {
				// Cache hit: write immediately (no goroutine needed).
				s.flushImageFromCache(col, row, cols, rows)
			} else if s.previewCancel == nil {
				// No goroutine running yet: start one.
				cellW, cellH := imagepreview.TerminalCellPixels()
				ctx, cancel := context.WithCancel(context.Background())
				s.previewCancel = cancel
				go s.loadImageAsync(ctx, path, cols*cellW, rows*cellH)
			}
		} else {
			drawRune := imagepreview.BlockRune
			if envVT {
				drawRune = imagepreview.ASCIIRune
			}
			imagepreview.DrawTextImage(s.canvas, path, col, row, cols, rows, drawRune)
			s.canvas.Draw()
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
	if !s.showPreviewPane() {
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

// drawTextPreview renders the first rows lines of a text file into the preview pane
// with syntax highlighting using the same method as Orbiton.
func (s *State) drawTextPreview(path string, col, row, cols, rows uint) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	// Detect the file mode and adjust keywords for syntax highlighting.
	m := mode.Detect(path)
	syntax.AdjustKeywords(m)
	syntax.SetDefaultTextConfigFromEnv()
	tout := vt.New()

	sc := bufio.NewScanner(f)
	for r := uint(0); r < rows && sc.Scan(); r++ {
		line := sc.Text()
		line = strings.ReplaceAll(line, "\t", "    ")
		runes := []rune(line)
		if uint(len(runes)) >= cols {
			runes = runes[:cols-1]
		}
		truncated := string(runes)
		tagged, err := syntax.AsText([]byte(truncated), m)
		if err != nil {
			fmt.Fprintf(os.Stdout, "\033[%d;%dH%s", row+r, col, truncated)
		} else {
			fmt.Fprintf(os.Stdout, "\033[%d;%dH%s\033[0m", row+r, col, tout.DarkTags(string(tagged)))
		}
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
