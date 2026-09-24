package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xyproto/vt"
)

// chunkReader hands out at most chunk bytes per Read, splitting a paste
// across several reads the way a terminal does.
type chunkReader struct {
	data  []byte
	chunk int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := min(r.chunk, len(p))
	if n > len(r.data) {
		n = len(r.data)
	}
	copy(p, r.data[:n])
	r.data = r.data[n:]
	return n, nil
}

func longJSONLine(pairs int) string {
	var sb strings.Builder
	sb.WriteString(`{"items":[`)
	for i := range pairs {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(`{"id":`)
		sb.WriteString(strings.Repeat("9", 4))
		sb.WriteString(`,"name":"item","value":"`)
		sb.WriteString(strings.Repeat("x", 20))
		sb.WriteString(`"}`)
	}
	sb.WriteString(`]}`)
	return sb.String()
}

// TestCollectPasteLongLine checks that a very long single-line paste is
// collected in full, however it is split across reads.
func TestCollectPasteLongLine(t *testing.T) {
	if isWindows {
		t.Skip("mock TTY key reading is not supported on Windows")
	}
	payload := longJSONLine(4000)
	if len(payload) < 100000 {
		t.Fatalf("test payload is only %d bytes", len(payload))
	}
	script := []byte(pasteStartMarker + payload + pasteEndMarker)

	for _, tc := range []struct {
		name  string
		chunk int
	}{
		{"one read", len(script)},
		{"4096 byte chunks", 4096},
		{"64 byte chunks", 64},
		{"7 byte chunks", 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tty := vt.NewTTYFromReader(&chunkReader{data: append([]byte{}, script...), chunk: tc.chunk})
			defer tty.Close()

			if got := tty.ReadKey(); got != pasteStartMarker {
				t.Fatalf("expected the paste start marker, got %q", got)
			}
			got := collectPaste(tty, time.Second)
			if got != payload {
				t.Errorf("collected %d bytes, want %d", len(got), len(payload))
				for i := 0; i < len(got) && i < len(payload); i++ {
					if got[i] != payload[i] {
						t.Errorf("first difference at byte %d", i)
						break
					}
				}
			}
		})
	}
}

// TestCollectPasteStopsAtEndMarker checks that keys typed after the paste are
// left for the editor.
func TestCollectPasteStopsAtEndMarker(t *testing.T) {
	if isWindows {
		t.Skip("mock TTY key reading is not supported on Windows")
	}
	script := []byte(pasteStartMarker + `{"a":1}` + pasteEndMarker + "Z")
	tty := vt.NewTTYFromReader(bytes.NewReader(script))
	defer tty.Close()

	if got := tty.ReadKey(); got != pasteStartMarker {
		t.Fatalf("expected the paste start marker, got %q", got)
	}
	if got := collectPaste(tty, time.Second); got != `{"a":1}` {
		t.Errorf("got %q, want %q", got, `{"a":1}`)
	}
	if got := tty.ReadKey(); got != "Z" {
		t.Errorf("key after the paste: got %q, want %q", got, "Z")
	}
}

// TestCollectPasteKeepsTabsAndNewlines checks that tabs and newlines survive.
func TestCollectPasteKeepsTabsAndNewlines(t *testing.T) {
	if isWindows {
		t.Skip("mock TTY key reading is not supported on Windows")
	}
	payload := "{\n\t\"a\": 1,\n\t\"b\": 2\n}"
	script := []byte(pasteStartMarker + payload + pasteEndMarker)
	tty := vt.NewTTYFromReader(&chunkReader{data: script, chunk: 3})
	defer tty.Close()

	if got := tty.ReadKey(); got != pasteStartMarker {
		t.Fatalf("expected the paste start marker, got %q", got)
	}
	got := collectPaste(tty, time.Second)
	want := strings.ReplaceAll(payload, "\n", "\r")
	if got != payload && got != want {
		t.Errorf("got %q, want %q", got, payload)
	}
}
