package main

import (
	"bytes"
	"compress/gzip"
	"strings"
)

// stripGZ removes the trailing ".gz" suffix
func stripGZ(filename string) string {
	return strings.TrimSuffix(filename, ".gz")
}

// gUnzipData uncompresses gzip data
func gUnzipData(data []byte) ([]byte, error) {
	b := bytes.NewBuffer(data)
	r, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var resB bytes.Buffer
	if _, err = resB.ReadFrom(r); err != nil {
		return nil, err
	}
	return resB.Bytes(), nil
}

// gZipData compresses data with gzip
func gZipData(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	defer gz.Close()

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
