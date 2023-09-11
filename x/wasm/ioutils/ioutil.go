package ioutils

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

var errLimit = fmt.Errorf("exceeds limit")

// Uncompress expects a valid gzip source to unpack or fails. See IsGzip
func Uncompress(gzipSrc []byte, limit int64) ([]byte, error) {
	if int64(len(gzipSrc)) > limit {
		return nil, fmt.Errorf("%s: max %d bytes", errLimit, limit)
	}
	zr, err := gzip.NewReader(bytes.NewReader(gzipSrc))
	if err != nil {
		return nil, err
	}
	zr.Multistream(false)
	defer zr.Close()
	bz, err := io.ReadAll(LimitReader(zr, limit))
	if err == errLimit {
		return nil, fmt.Errorf("%s: max %d bytes", errLimit, limit)
	}
	return bz, err
}

// LimitReader returns a Reader that reads from r
// but stops with "limit error" after n bytes.
// The underlying implementation is a *io.LimitedReader.
func LimitReader(r io.Reader, n int64) io.Reader {
	return &LimitedReader{r: &io.LimitedReader{R: r, N: n}}
}

type LimitedReader struct {
	r *io.LimitedReader
}

func (l *LimitedReader) Read(p []byte) (n int, err error) {
	if l.r.N <= 0 {
		return 0, errLimit
	}
	return l.r.Read(p)
}
