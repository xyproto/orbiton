//go:build !tinygo
// +build !tinygo

package gfx

import (
	"encoding/json"
	"io"
)

// JSONIndent configures the prefix and indent level of a JSON encoder.
func JSONIndent(prefix, indent string) func(*json.Encoder) {
	return func(enc *json.Encoder) {
		enc.SetIndent(prefix, indent)
	}
}

// NewJSONEncoder creates a new JSON encoder for the given io.Writer.
func NewJSONEncoder(w io.Writer, options ...func(*json.Encoder)) *json.Encoder {
	enc := json.NewEncoder(w)

	for _, o := range options {
		o(enc)
	}

	return enc
}

// EncodeJSON creates a new JSON encoder and encodes the provided value.
func EncodeJSON(w io.Writer, v interface{}, options ...func(*json.Encoder)) error {
	return NewJSONEncoder(w, options...).Encode(v)
}
