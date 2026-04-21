package imagepreview

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"sort"
)

// sixelMaxColors is the maximum number of palette entries for Sixel output.
// Most terminals support at least 256 colours in Sixel mode.
const sixelMaxColors = 256

// sixelRGBA32 is an 8-bit-per-channel colour used during Sixel quantization.
type sixelRGBA32 struct{ r, g, b, a uint8 }

// SixelEncode writes img to w as a Sixel escape sequence.
// The image is quantised to at most sixelMaxColors and rendered in bands of
// 6 pixel rows, which is how the Sixel protocol works.
func SixelEncode(w io.Writer, img image.Image) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return
	}

	// Build a colour palette by sampling the image. We use a simple
	// popularity-based approach: collect all unique colours, then keep
	// the most frequent ones up to sixelMaxColors.
	palette, indexed := sixelQuantize(img, width, height)

	// Sixel header: DCS q (with raster attributes).
	// P1=0 (aspect ratio default), P2=0 (background default), P3=q (Sixel mode).
	// Raster attributes: "width;height
	fmt.Fprintf(w, "\033Pq\"1;1;%d;%d", width, height)

	// Define palette colours.
	for i, c := range palette {
		r, g, b, _ := c.RGBA()
		// Sixel colour components are percentages 0-100.
		pr := int(r>>8) * 100 / 255
		pg := int(g>>8) * 100 / 255
		pb := int(b>>8) * 100 / 255
		fmt.Fprintf(w, "#%d;2;%d;%d;%d", i, pr, pg, pb)
	}

	// Render sixel bands (6 rows each).
	for bandY := 0; bandY < height; bandY += 6 {
		// Collect which colours are used in this band and their sixel data.
		type bandEntry struct {
			colorIdx int
			data     []byte // one byte per column: bits 0-5 = pixel mask
		}
		bandMap := make(map[int]*bandEntry)

		for dy := 0; dy < 6; dy++ {
			py := bandY + dy
			if py >= height {
				break
			}
			bit := byte(1 << dy)
			for x := 0; x < width; x++ {
				ci := indexed[py*width+x]
				be, ok := bandMap[ci]
				if !ok {
					be = &bandEntry{colorIdx: ci, data: make([]byte, width)}
					bandMap[ci] = be
				}
				be.data[x] |= bit
			}
		}

		// Sort colour indices for deterministic output.
		indices := make([]int, 0, len(bandMap))
		for ci := range bandMap {
			indices = append(indices, ci)
		}
		sort.Ints(indices)

		// Write each colour's sixel row.
		for i, ci := range indices {
			be := bandMap[ci]
			fmt.Fprintf(w, "#%d", be.colorIdx)
			// RLE-encode the sixel data.
			sixelWriteRLE(w, be.data)
			if i < len(indices)-1 {
				// Carriage return (stay on same band for next colour).
				w.Write([]byte("$"))
			}
		}
		// Line feed: advance to next 6-pixel band.
		if bandY+6 < height {
			w.Write([]byte("-"))
		}
	}

	// Sixel terminator (ST = String Terminator).
	fmt.Fprint(w, "\033\\")
}

// sixelWriteRLE writes sixel data with simple run-length encoding.
func sixelWriteRLE(w io.Writer, data []byte) {
	n := len(data)
	i := 0
	for i < n {
		ch := data[i] + 63 // Sixel character: 63 ('?') + bit pattern
		j := i + 1
		for j < n && data[j] == data[i] {
			j++
		}
		runLen := j - i
		if runLen >= 4 {
			fmt.Fprintf(w, "!%d%c", runLen, ch)
		} else {
			for range runLen {
				w.Write([]byte{ch})
			}
		}
		i = j
	}
}

// sixelQuantize reduces the image to at most sixelMaxColors colours and
// returns the palette and an indexed pixel array (row-major, one byte per pixel).
func sixelQuantize(img image.Image, width, height int) ([]color.Color, []int) {
	// Count colour frequencies using a map keyed on RGBA32.
	freq := make(map[sixelRGBA32]int)
	pixels := make([]sixelRGBA32, width*height)

	for y := range height {
		for x := range width {
			r, g, b, a := img.At(x+img.Bounds().Min.X, y+img.Bounds().Min.Y).RGBA()
			c := sixelRGBA32{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
			freq[c]++
			pixels[y*width+x] = c
		}
	}

	// If <= sixelMaxColors unique colours, use them directly.
	if len(freq) <= sixelMaxColors {
		palette := make([]color.Color, 0, len(freq))
		lookup := make(map[sixelRGBA32]int, len(freq))
		for c := range freq {
			lookup[c] = len(palette)
			palette = append(palette, color.NRGBA{c.r, c.g, c.b, c.a})
		}
		indexed := make([]int, len(pixels))
		for i, p := range pixels {
			indexed[i] = lookup[p]
		}
		return palette, indexed
	}

	// Too many colours: use median-cut quantization.
	return sixelMedianCut(pixels, freq, width, height)
}

// sixelMedianCut performs a simple median-cut colour quantization.
func sixelMedianCut(pixels []sixelRGBA32, freq map[sixelRGBA32]int, width, height int) ([]color.Color, []int) {
	// Collect all unique colours.
	type colorEntry struct {
		c    sixelRGBA32
		freq int
	}
	entries := make([]colorEntry, 0, len(freq))
	for c, f := range freq {
		entries = append(entries, colorEntry{c, f})
	}

	// Recursively split the colour space.
	type bucket struct {
		entries []colorEntry
	}

	// Find the channel with the widest range.
	channelRange := func(es []colorEntry) (ch int) {
		var rMin, gMin, bMin uint8 = 255, 255, 255
		var rMax, gMax, bMax uint8
		for _, e := range es {
			if e.c.r < rMin {
				rMin = e.c.r
			}
			if e.c.r > rMax {
				rMax = e.c.r
			}
			if e.c.g < gMin {
				gMin = e.c.g
			}
			if e.c.g > gMax {
				gMax = e.c.g
			}
			if e.c.b < bMin {
				bMin = e.c.b
			}
			if e.c.b > bMax {
				bMax = e.c.b
			}
		}
		dr := int(rMax) - int(rMin)
		dg := int(gMax) - int(gMin)
		db := int(bMax) - int(bMin)
		if dg >= dr && dg >= db {
			return 1
		}
		if db >= dr && db >= dg {
			return 2
		}
		return 0
	}

	buckets := []bucket{{entries}}
	for len(buckets) < sixelMaxColors {
		// Find the largest bucket (by pixel count) to split.
		best := -1
		bestSize := 0
		for i, b := range buckets {
			if len(b.entries) < 2 {
				continue
			}
			s := 0
			for _, e := range b.entries {
				s += e.freq
			}
			if s > bestSize {
				bestSize = s
				best = i
			}
		}
		if best < 0 {
			break
		}
		b := buckets[best]
		ch := channelRange(b.entries)
		sort.Slice(b.entries, func(i, j int) bool {
			switch ch {
			case 1:
				return b.entries[i].c.g < b.entries[j].c.g
			case 2:
				return b.entries[i].c.b < b.entries[j].c.b
			default:
				return b.entries[i].c.r < b.entries[j].c.r
			}
		})
		mid := len(b.entries) / 2
		buckets[best] = bucket{b.entries[:mid]}
		buckets = append(buckets, bucket{b.entries[mid:]})
	}

	// Build palette: average colour of each bucket.
	palette := make([]color.Color, len(buckets))
	lookup := make(map[sixelRGBA32]int)
	for i, b := range buckets {
		var rSum, gSum, bSum, aSum, total int
		for _, e := range b.entries {
			rSum += int(e.c.r) * e.freq
			gSum += int(e.c.g) * e.freq
			bSum += int(e.c.b) * e.freq
			aSum += int(e.c.a) * e.freq
			total += e.freq
		}
		if total == 0 {
			total = 1
		}
		avg := color.NRGBA{
			R: uint8(rSum / total),
			G: uint8(gSum / total),
			B: uint8(bSum / total),
			A: uint8(aSum / total),
		}
		palette[i] = avg
		for _, e := range b.entries {
			lookup[e.c] = i
		}
	}

	indexed := make([]int, len(pixels))
	for i, p := range pixels {
		if idx, ok := lookup[p]; ok {
			indexed[i] = idx
		} else {
			// Nearest-colour fallback (shouldn't happen with median-cut).
			indexed[i] = sixelNearest(palette, p)
		}
	}
	return palette, indexed
}

// sixelNearest finds the closest palette colour to c by Euclidean distance.
func sixelNearest(palette []color.Color, c sixelRGBA32) int {
	best := 0
	bestDist := 1<<30 - 1
	for i, pc := range palette {
		pr, pg, pb, _ := pc.RGBA()
		dr := int(pr>>8) - int(c.r)
		dg := int(pg>>8) - int(c.g)
		db := int(pb>>8) - int(c.b)
		d := dr*dr + dg*dg + db*db
		if d < bestDist {
			bestDist = d
			best = i
		}
	}
	return best
}
