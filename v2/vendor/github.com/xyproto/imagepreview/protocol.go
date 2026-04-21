package imagepreview

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
)

// DeleteInlineImages sends the appropriate protocol command to delete all
// previously placed inline images. For Kitty it issues an explicit delete;
// iTerm2 and Sixel don't require one (overwriting the cells is sufficient).
func DeleteInlineImages() {
	if IsKitty {
		fmt.Fprintf(os.Stdout, "\033_Ga=d,d=A,q=2\033\\")
	}
}

// DeleteKittyImageByID deletes a previously placed Kitty image by its ID.
// This is much cheaper than DeleteInlineImages which deletes ALL images.
func DeleteKittyImageByID(id uint32) {
	if IsKitty {
		fmt.Fprintf(os.Stdout, "\033_Ga=d,d=i,i=%d,q=2\033\\", id)
	}
}

// FlushImage writes an image to w using the appropriate terminal graphics
// protocol: Kitty (f=100 PNG), iTerm2 (inline image), or Sixel.
// For Kitty and iTerm2, encoded is base64-encoded PNG data.
// For Sixel, encoded is base64-encoded PNG data that will be decoded and
// re-encoded as Sixel.
// dispCols and dispRows specify the display size in terminal cells.
func FlushImage(w io.Writer, encoded string, dispCols, dispRows uint) {
	if IsITerm2 {
		fmt.Fprintf(w, "\033]1337;File=inline=1;width=%d;height=%d;preserveAspectRatio=1:%s\a",
			dispCols, dispRows, encoded)
		return
	}

	if IsSixel {
		flushSixelFromEncoded(w, encoded)
		return
	}

	// Kitty graphics protocol with chunked transmission.
	const chunkSize = 4096
	total := len(encoded)
	for i := 0; i < total; i += chunkSize {
		end := min(i+chunkSize, total)
		chunk := encoded[i:end]
		isLast := end >= total
		isFirst := i == 0

		switch {
		case isFirst && isLast:
			fmt.Fprintf(w, "\033_Ga=T,f=100,q=2,C=1,c=%d,r=%d;%s\033\\", dispCols, dispRows, chunk)
		case isFirst:
			fmt.Fprintf(w, "\033_Ga=T,f=100,q=2,C=1,m=1,c=%d,r=%d;%s\033\\", dispCols, dispRows, chunk)
		case isLast:
			fmt.Fprintf(w, "\033_Gm=0;%s\033\\", chunk)
		default:
			fmt.Fprintf(w, "\033_Gm=1;%s\033\\", chunk)
		}
	}
}

// FlushImageWithID writes a base64-encoded PNG image to w using the Kitty
// graphics protocol with an explicit image ID, or the iTerm2 inline protocol
// (which ignores the ID). The image is transmitted and displayed immediately.
// For Sixel terminals, the ID is ignored (Sixel has no image ID concept).
func FlushImageWithID(w io.Writer, encoded string, dispCols, dispRows uint, id uint32) {
	if IsITerm2 || IsSixel {
		// iTerm2 has no image ID concept; just overwrite. Same for Sixel.
		FlushImage(w, encoded, dispCols, dispRows)
		return
	}

	// Kitty graphics protocol with image ID and chunked transmission.
	const chunkSize = 4096
	total := len(encoded)
	for i := 0; i < total; i += chunkSize {
		end := min(i+chunkSize, total)
		chunk := encoded[i:end]
		isLast := end >= total
		isFirst := i == 0

		switch {
		case isFirst && isLast:
			fmt.Fprintf(w, "\033_Ga=T,f=100,q=2,C=1,i=%d,c=%d,r=%d;%s\033\\", id, dispCols, dispRows, chunk)
		case isFirst:
			fmt.Fprintf(w, "\033_Ga=T,f=100,q=2,C=1,m=1,i=%d,c=%d,r=%d;%s\033\\", id, dispCols, dispRows, chunk)
		case isLast:
			fmt.Fprintf(w, "\033_Gm=0;%s\033\\", chunk)
		default:
			fmt.Fprintf(w, "\033_Gm=1;%s\033\\", chunk)
		}
	}
}

// FlushSixelImage writes an image.Image directly as a Sixel sequence to w.
// This is more efficient than going through base64 PNG encoding when the
// caller already has the image in memory.
func FlushSixelImage(w io.Writer, img image.Image) {
	SixelEncode(w, img)
}

// flushSixelFromEncoded decodes base64 PNG data and re-encodes it as Sixel.
func flushSixelFromEncoded(w io.Writer, encoded string) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return
	}
	SixelEncode(w, img)
}
