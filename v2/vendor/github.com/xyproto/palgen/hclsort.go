package palgen

import (
	"github.com/lucasb-eyer/go-colorful"
	"image/color"
	"sort"
)

// HCLSortablePalette is a slice of color.Color that can be sorted with sort.Sort, by h, c and l values
type HCLSortablePalette []color.Color

func (a HCLSortablePalette) Len() int      { return len(a) }
func (a HCLSortablePalette) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a HCLSortablePalette) Less(i, j int) bool {
	// Create colorful.Color structs
	ci, _ := colorful.MakeColor(a[i])
	cj, _ := colorful.MakeColor(a[j])
	// Convert the colors to HCL
	h1, c1, l1 := ci.Hcl()
	h2, c2, l2 := cj.Hcl()
	// Compare the H, C and L values
	if h1 == h2 {
		if c1 == c2 {
			return l1 < l2
		}
		return c1 < c2
	}
	return h1 < h2

}

// Sort the palette by luminance, hue and then chroma
func Sort(pal color.Palette) {
	//tmp := HCLSortablePalette(pal)
	sort.Sort(HCLSortablePalette(pal))
	//return color.Palette(tmp)
}
