package main

import (
	"bytes"
	"os"
	"testing"
)

func TestBinaryLoadSaveRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"trailing newline", []byte("\x00\x01\x02hello\nworld\xff\xfe\x00\n")},
		{"no trailing newline", []byte("\x7fELF\x02\x01\x01\x00\x00\x00")},
		{"all NUL", []byte{0, 0, 0, 0}},
		{"single byte", []byte{0xeb}},
	}
	for _, tc := range cases {
		t.Run(tc.name+" (LoadBytes)", func(t *testing.T) {
			e := &Editor{}
			e.lines = make(map[int][]rune)
			e.LoadBytes(tc.data)
			if !e.binaryFile {
				t.Fatalf("expected binaryFile=true")
			}
			got := e.binaryBytes()
			if !bytes.Equal(got, tc.data) {
				t.Fatalf("want % x, got % x", tc.data, got)
			}
		})
		t.Run(tc.name+" (ReadFileAndProcessLines)", func(t *testing.T) {
			tmp, err := os.CreateTemp("", "orbiton-bin-*.bin")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmp.Name())
			if _, err := tmp.Write(tc.data); err != nil {
				t.Fatal(err)
			}
			tmp.Close()
			e := &Editor{}
			e.lines = make(map[int][]rune)
			if err := e.ReadFileAndProcessLines(tmp.Name()); err != nil {
				t.Fatal(err)
			}
			if !e.binaryFile {
				t.Fatalf("expected binaryFile=true")
			}
			got := e.binaryBytes()
			if !bytes.Equal(got, tc.data) {
				t.Fatalf("want % x, got % x", tc.data, got)
			}
		})
	}
}
