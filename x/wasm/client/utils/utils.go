package utils

import (
	"bytes"
	"compress/gzip"
	"io"
)

var (
	gzipIdent = []byte("\x1F\x8B\x08")
	wasmIdent = []byte("\x00\x61\x73\x6D")
)

// limit max bytes read to prevent gzip bombs
const maxSize = 400 * 1024

// IsGzip returns checks if the file contents are gzip compressed
func IsGzip(input []byte) bool {
	if len(input) < 3 {
		return false
	}

	in := io.LimitReader(bytes.NewReader(input), maxSize)
	buf := make([]byte, 3)

	if _, err := io.ReadAtLeast(in, buf, 3); err != nil {
		return false
	}

	return bytes.Equal(gzipIdent, buf)
}

// IsWasm checks if the file contents are of wasm binary
func IsWasm(input []byte) bool {
	if len(input) < 3 {
		return false
	}

	in := io.LimitReader(bytes.NewReader(input), maxSize)
	buf := make([]byte, 4)

	if _, err := io.ReadAtLeast(in, buf, 4); err != nil {
		return false
	}

	if bytes.Equal(wasmIdent, buf) {
		return true
	}

	return false
}

// GzipIt compresses the input ([]byte)
func GzipIt(input []byte) ([]byte, error) {
	// Create gzip writer.
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write(input)
	if err != nil {
		return nil, err
	}
	err = w.Close() // You must close this first to flush the bytes to the buffer.
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
