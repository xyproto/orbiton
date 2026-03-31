package imagepreview

import (
	"fmt"
	"io"
	"os"
)

// DeleteInlineImages sends the appropriate protocol command to delete all
// previously placed inline images. For Kitty it issues an explicit delete;
// iTerm2 doesn't require one (overwriting the cells is sufficient).
func DeleteInlineImages() {
	if IsKitty {
		fmt.Fprintf(os.Stdout, "\033_Ga=d,d=A,q=2\033\\")
	}
}

// FlushImage writes a base64-encoded PNG image to w using the Kitty graphics
// protocol (f=100) or iTerm2 inline image protocol.
// dispCols and dispRows specify the display size in terminal cells.
func FlushImage(w io.Writer, encoded string, dispCols, dispRows uint) {
	if IsITerm2 {
		fmt.Fprintf(w, "\033]1337;File=inline=1;width=%d;height=%d;preserveAspectRatio=1:%s\a",
			dispCols, dispRows, encoded)
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
			fmt.Fprintf(w, "\033_Ga=T,f=100,q=2,c=%d,r=%d;%s\033\\", dispCols, dispRows, chunk)
		case isFirst:
			fmt.Fprintf(w, "\033_Ga=T,f=100,q=2,m=1,c=%d,r=%d;%s\033\\", dispCols, dispRows, chunk)
		case isLast:
			fmt.Fprintf(w, "\033_Gm=0;%s\033\\", chunk)
		default:
			fmt.Fprintf(w, "\033_Gm=1;%s\033\\", chunk)
		}
	}
}
