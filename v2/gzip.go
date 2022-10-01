package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
)

// withoutGZ removes the trailing ".gz" suffix
func withoutGZ(filename string) string {
	return strings.TrimSuffix(filename, ".gz")
}

// gUnzipData uncompressed gzip data
func gUnzipData(data []byte) ([]byte, error) {
	var (
		b    = bytes.NewBuffer(data)
		r    io.Reader
		resB bytes.Buffer
		err  error
	)
	r, err = gzip.NewReader(b)
	if err != nil {
		return nil, err
	}
	if _, err = resB.ReadFrom(r); err != nil {
		return nil, err
	}
	return resB.Bytes(), nil
}

// gZipData compresses data with gzip
func gZipData(data []byte) ([]byte, error) {
	var (
		b  bytes.Buffer
		gz = gzip.NewWriter(&b)
	)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Flush(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
