//go:build !tinygo
// +build !tinygo

package gfx

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
)

// DefaultAnimationDelay is the default animation delay, in 100ths of a second.
var DefaultAnimationDelay = 50

// Animation represents multiple images.
type Animation struct {
	Frames   []image.Image   // The successive images.
	Palettes []color.Palette // The successive palettes.

	Delay int // Delay between each of the frames.

	// LoopCount controls the number of times an animation will be
	// restarted during display.
	// A LoopCount of 0 means to loop forever.
	// A LoopCount of -1 means to show each frame only once.
	// Otherwise, the animation is looped LoopCount+1 times.
	LoopCount int
}

// AddPalettedImage adds a frame and palette to the animation.
func (a *Animation) AddPalettedImage(frame PalettedImage) {
	a.Frames = append(a.Frames, frame)
	a.Palettes = append(a.Palettes, frame.ColorPalette())
}

// AddFrame adds a frame to the animation.
func (a *Animation) AddFrame(frame image.Image, palette color.Palette) {
	a.Frames = append(a.Frames, frame)
	a.Palettes = append(a.Palettes, palette)
}

// SaveGIF saves the animation to a GIF using the provided file name.
func (a *Animation) SaveGIF(fn string) error {
	w, err := CreateFile(fn)
	if err != nil {
		return err
	}
	defer w.Close()

	return a.EncodeGIF(w)
}

// EncodeGIF writes the animation to w in GIF format with the
// given loop count and delay between frames.
func (a *Animation) EncodeGIF(w io.Writer) error {
	if len(a.Frames) != len(a.Palettes) {
		return Error("Animation: the number of Frames and Palettes does not match")
	}

	if a.Delay < 1 {
		a.Delay = DefaultAnimationDelay
	}

	var frames []*image.Paletted
	var delays []int
	var disposal []byte

	for i, src := range a.Frames {
		dst := image.NewPaletted(src.Bounds(), a.Palettes[i])

		draw.Draw(dst, dst.Bounds(), src, image.ZP, draw.Src)

		frames = append(frames, dst)
		delays = append(delays, a.Delay)
		disposal = append(disposal, gif.DisposalBackground)
	}

	return gif.EncodeAll(w, &gif.GIF{
		Image:     frames,
		Delay:     delays,
		LoopCount: a.LoopCount,
		Disposal:  disposal,
	})
}
