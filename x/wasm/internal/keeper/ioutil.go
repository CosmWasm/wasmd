package keeper

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
)

// magic bytes to identify gzip.
// See https://www.ietf.org/rfc/rfc1952.txt
// and https://github.com/golang/go/blob/master/src/net/http/sniff.go#L186
var gzipIdent = []byte("\x1F\x8B\x08")

// limit max bytes read to prevent gzip bombs
const maxSize = 400 * 1024

// uncompress returns gzip uncompressed content or given src when not gzip.
func uncompress(src []byte) ([]byte, error) {
	if len(src) < 3 {
		return src, nil
	}
	if !bytes.Equal(gzipIdent, src[0:3]) {
		return src, nil
	}
	zr, err := gzip.NewReader(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	zr.Multistream(false)
	defer zr.Close()
	return ioutil.ReadAll(LimitReader(zr, maxSize))
}

// LimitReader returns a Reader that reads from r
// but stops with types.ErrLimit after n bytes.
// The underlying implementation is a *io.LimitedReader.
func LimitReader(r io.Reader, n int64) io.Reader {
	return &LimitedReader{r: &io.LimitedReader{R: r, N: n}}
}

type LimitedReader struct {
	r *io.LimitedReader
}

func (l *LimitedReader) Read(p []byte) (n int, err error) {
	if l.r.N <= 0 {
		return 0, types.ErrLimit
	}
	return l.r.Read(p)
}
