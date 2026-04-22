package gfx

import "testing"

func TestPalettes(t *testing.T) {
	for _, tc := range []struct {
		Name       string
		Palette    Palette
		ColorCount int
	}{
		{"15PDX", Palette15PDX, 15},
		{"1Bit", Palette1Bit, 2},
		{"20PDX", Palette20PDX, 20},
		{"2BitGrayScale", Palette2BitGrayScale, 4},
		{"3Bit", Palette3Bit, 8},
		{"AAP16", PaletteAAP16, 16},
		{"AAP64", PaletteAAP64, 64},
		{"ARQ4", PaletteARQ4, 4},
		{"Ammo8", PaletteAmmo8, 8},
		{"Arne16", PaletteArne16, 16},
		{"CGA", PaletteCGA, 16},
		{"EDG16", PaletteEDG16, 16},
		{"EDG32", PaletteEDG32, 32},
		{"EDG36", PaletteEDG36, 36},
		{"EDG64", PaletteEDG64, 64},
		{"EDG8", PaletteEDG8, 8},
		{"EN4", PaletteEN4, 4},
		{"Famicube", PaletteFamicube, 64},
		{"Ink", PaletteInk, 5},
		{"NYX8", PaletteNYX8, 8},
		{"Night16", PaletteNight16, 16},
		{"PICO8", PalettePICO8, 16},
		{"Splendor128", PaletteSplendor128, 128},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			if got, want := len(tc.Palette), tc.ColorCount; got != want {
				t.Fatalf("unexpected number of colors: %d, want %d", got, want)
			}

			for n, want := range tc.Palette {
				if got := tc.Palette.Color(n); got != want {
					t.Fatalf("Color(%d) = %v, want %v", n, got, want)
				}
			}
		})
	}
}

func TestPalettesByNameAndCount(t *testing.T) {
	var nc int

	for _, p := range PalettesByNumberOfColors {
		nc += len(p)
	}

	pc := len(PaletteByName)

	if nc != pc {
		t.Fatalf("nc = %d, want %d", nc, pc)
	}
}
