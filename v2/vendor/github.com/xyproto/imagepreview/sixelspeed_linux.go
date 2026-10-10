//go:build linux

package imagepreview

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xyproto/env/v2"
	"golang.org/x/sys/unix"
)

const (
	// sixelTestWidth and sixelTestHeight give the size of the image that is drawn to measure Sixel speed
	sixelTestWidth  = 400
	sixelTestHeight = 96

	// sixelMinPixelsPerSecond is the slowest acceptable drawing speed, which draws
	// a full 1280x800 screen in about half a second
	sixelMinPixelsPerSecond = 2_000_000

	// sixelReplyTimeout is how long to wait for the terminal to reply, before giving up
	sixelReplyTimeout = 3 * time.Second
)

// CheckSixelSpeed turns off Sixel graphics (IsSixel and HasGraphics) if the terminal
// supports Sixel but draws it too slowly to be useful. See SixelFastEnough.
func CheckSixelSpeed() {
	if IsSixel && !SixelFastEnough() {
		IsSixel = false
		HasGraphics = (IsKitty || IsITerm2) && !IsVT && !env.Bool("NO_COLOR")
	}
}

// SixelFastEnough reports whether the terminal draws Sixel graphics quickly enough.
// It draws a small test image in the top left corner and times how long the terminal
// takes to answer a Device Attributes request sent right after it, since terminals
// handle their input in order. The time it takes to answer without an image is
// subtracted. The result is cached per terminal and CPU, so that the test image is
// only drawn once. Set IMAGEPREVIEW_SIXEL to 1 or 0 to skip the measurement.
// The caller should redraw the screen afterwards.
func SixelFastEnough() bool {
	if v := env.Str("IMAGEPREVIEW_SIXEL"); v != "" {
		return env.Bool("IMAGEPREVIEW_SIXEL")
	}
	cacheFile := sixelSpeedCacheFile()
	if data, err := os.ReadFile(cacheFile); err == nil {
		return strings.TrimSpace(string(data)) != "slow"
	}
	fast, ok := measureSixelSpeed()
	if !ok {
		return true
	}
	result := "fast"
	if !fast {
		result = "slow"
	}
	if os.MkdirAll(filepath.Dir(cacheFile), 0o755) == nil {
		_ = os.WriteFile(cacheFile, []byte(result+"\n"), 0o644)
	}
	return fast
}

// sixelSpeedCacheFile returns the path to the cached measurement for this terminal and CPU
func sixelSpeedCacheFile() string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s\n%s\n%d\n", env.Str("TERM"), env.Str("TERM_PROGRAM"), runtime.NumCPU())
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "model name") {
				fmt.Fprintln(h, line)
				break
			}
		}
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "imagepreview", fmt.Sprintf("sixel-%x", h.Sum64()))
}

// measureSixelSpeed draws a test image and returns true if it was drawn quickly enough.
// The second return value is false if the speed could not be measured.
func measureSixelSpeed() (bool, bool) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false, false
	}
	defer tty.Close()
	fd := int(tty.Fd())
	saved, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return false, false
	}
	raw := *saved
	raw.Lflag &^= unix.ICANON | unix.ECHO
	raw.Cc[unix.VMIN] = 0
	raw.Cc[unix.VTIME] = 1
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &raw); err != nil {
		return false, false
	}
	defer unix.IoctlSetTermios(fd, unix.TCSETS, saved)

	img := image.NewRGBA(image.Rect(0, 0, sixelTestWidth, sixelTestHeight))
	for y := range sixelTestHeight {
		for x := range sixelTestWidth {
			img.Set(x, y, color.RGBA{uint8(x * 255 / sixelTestWidth), uint8(y * 255 / sixelTestHeight), uint8((x ^ y) & 0xff), 255})
		}
	}
	var sixel bytes.Buffer
	SixelEncode(&sixel, img)

	baseline, ok := timeDeviceAttributes(tty, nil)
	if !ok {
		return false, false
	}
	elapsed, ok := timeDeviceAttributes(tty, sixel.Bytes())
	if !ok {
		return false, true
	}
	drawing := max(elapsed-baseline, time.Microsecond)
	pixelsPerSecond := float64(sixelTestWidth*sixelTestHeight) / drawing.Seconds()
	return pixelsPerSecond >= sixelMinPixelsPerSecond, true
}

// timeDeviceAttributes writes the given data at the top left corner, followed by a
// Device Attributes request, and returns how long it took until the reply arrived
func timeDeviceAttributes(tty *os.File, data []byte) (time.Duration, bool) {
	var out bytes.Buffer
	out.WriteString("\x1b7\x1b[1;1H")
	out.Write(data)
	out.WriteString("\x1b[c\x1b8")
	start := time.Now()
	if _, err := tty.Write(out.Bytes()); err != nil {
		return 0, false
	}
	var reply []byte
	buf := make([]byte, 64)
	for time.Since(start) < sixelReplyTimeout {
		n, err := tty.Read(buf)
		if err != nil {
			return 0, false
		}
		reply = append(reply, buf[:n]...)
		if i := bytes.Index(reply, []byte("\x1b[?")); i >= 0 && bytes.IndexByte(reply[i:], 'c') >= 0 {
			return time.Since(start), true
		}
	}
	return 0, false
}
