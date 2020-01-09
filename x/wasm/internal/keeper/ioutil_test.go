package keeper

import (
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUncompress(t *testing.T) {
	wasmRaw, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	wasmGzipped, err := ioutil.ReadFile("./testdata/contract.wasm.gzip")
	require.NoError(t, err)

	specs := map[string]struct {
		src       []byte
		expError  error
		expResult []byte
	}{
		"handle wasm uncompressed": {
			src:       wasmRaw,
			expResult: wasmRaw,
		},
		"handle wasm compressed": {
			src:       wasmGzipped,
			expResult: wasmRaw,
		},
		"handle nil slice": {
			src:       nil,
			expResult: nil,
		},
		"handle short unidentified": {
			src:       []byte{0x1, 0x2},
			expResult: []byte{0x1, 0x2},
		},
		"handle big slice": {
			src:       []byte(strings.Repeat("a", maxSize+1)),
			expResult: []byte(strings.Repeat("a", maxSize+1)),
		},
		"handle gzip identifier only": {
			src:      gzipIdent,
			expError: io.ErrUnexpectedEOF,
		},
		"handle broken gzip": {
			src:      append(gzipIdent, byte(0x1)),
			expError: io.ErrUnexpectedEOF,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			r, err := uncompress(spec.src)
			require.True(t, errors.Is(spec.expError, err), "exp %+v got %+v", spec.expError, err)
			assert.Equal(t, spec.expResult, r)
		})
	}

}
