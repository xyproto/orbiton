package gfx

import (
	"encoding/json"
	"image"
	"io"
	"os"
)

// SavePNG saves an image using the provided file name.
func SavePNG(fn string, src image.Image) error {
	if src == nil || src.Bounds().Empty() {
		return Error("SavePNG: empty image provided")
	}

	w, err := CreateFile(fn)
	if err != nil {
		return err
	}
	defer w.Close()

	return EncodePNG(w, src)
}

// MustOpenImage decodes an image using the provided file name. Panics on error.
func MustOpenImage(fn string) image.Image {
	m, err := OpenImage(fn)
	if err != nil {
		panic(err)
	}

	return m
}

// OpenImage decodes an image using the provided file name.
func OpenImage(fn string) (image.Image, error) {
	r, err := OpenFile(fn)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return DecodeImage(r)
}

// OpenFile opens the named file for reading.
func OpenFile(fn string) (*os.File, error) {
	return os.Open(fn)
}

// CreateFile creates or truncates the named file.
func CreateFile(fn string) (*os.File, error) {
	return os.Create(fn)
}

// ReadFile opens a file and calls the given ReadFunc.
func ReadFile(fn string, rf ReadFunc) error {
	f, err := OpenFile(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	return rf(f)
}

// ReadJSON opens and decodes a JSON file.
func ReadJSON(fn string, v interface{}) error {
	return ReadFile(fn, DecodeJSONFunc(v))
}

// ReadFunc is a func that takes a io.Reader and returns an error.
type ReadFunc func(r io.Reader) error

// DecodeJSONFunc returns a function that takes a reader, and decodes into the given value.
func DecodeJSONFunc(v interface{}) ReadFunc {
	return func(r io.Reader) error {
		return json.NewDecoder(r).Decode(v)
	}
}
