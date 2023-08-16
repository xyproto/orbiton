package ico

import "image"

// ICO represents the possibly multiple images stored in a ICO file.
type ICO struct {
	Num   int           // Total number of images
	Image []image.Image // The images themselves
}

type entry struct {
	Width   uint8
	Height  uint8
	Palette uint8
	_       uint8 // Reserved byte
	Plane   uint16
	Bits    uint16
	Size    uint32
	Offset  uint32
}
