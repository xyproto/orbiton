//go:build !linux

package imagepreview

// CheckSixelSpeed does nothing on platforms where the Sixel speed can not be measured.
func CheckSixelSpeed() {}

// SixelFastEnough returns true on platforms where the Sixel speed can not be measured.
func SixelFastEnough() bool {
	return true
}
