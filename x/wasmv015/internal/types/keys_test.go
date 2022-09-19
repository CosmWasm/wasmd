package types

import (
	"bytes"
	"testing"

	v040auth "github.com/cosmos/cosmos-sdk/x/auth/legacy/v040"
	"github.com/stretchr/testify/assert"
)

func TestGetContractByCodeIDSecondaryIndexPrefix(t *testing.T) {
	specs := map[string]struct {
		src uint64
		exp []byte
	}{
		"small number": {src: 1,
			exp: []byte{6, 0, 0, 0, 0, 0, 0, 0, 1},
		},
		"big number": {src: 1 << (8 * 7),
			exp: []byte{6, 1, 0, 0, 0, 0, 0, 0, 0},
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			got := GetContractByCodeIDSecondaryIndexPrefix(spec.src)
			assert.Equal(t, spec.exp, got)
		})
	}
}

func TestGetContractByCreatedSecondaryIndexKey(t *testing.T) {
	c := &ContractInfo{
		CodeID:  1,
		Created: &AbsoluteTxPosition{2 + 1<<(8*7), 3 + 1<<(8*7)},
	}
	addr := bytes.Repeat([]byte{4}, v040auth.AddrLen)
	got := GetContractByCreatedSecondaryIndexKey(addr, c)
	exp := []byte{6, // prefix
		0, 0, 0, 0, 0, 0, 0, 1, // codeID
		1, 0, 0, 0, 0, 0, 0, 2, // height
		1, 0, 0, 0, 0, 0, 0, 3, // index
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, // address
	}
	assert.Equal(t, exp, got)
}
