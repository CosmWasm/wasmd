package types

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetContractByCodeIDSecondaryIndexPrefix(t *testing.T) {
	specs := map[string]struct {
		src uint64
		exp []byte
	}{
		"small number": {
			src: 1,
			exp: []byte{6, 0, 0, 0, 0, 0, 0, 0, 1},
		},
		"big number": {
			src: 1 << (8 * 7),
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

func TestGetContractCodeHistoryElementPrefix(t *testing.T) {
	// test that contract addresses of 20 length are still supported
	addr := bytes.Repeat([]byte{4}, 20)
	got := GetContractCodeHistoryElementPrefix(addr)
	exp := []byte{
		5,                            // prefix
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4, // address 20 bytes
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	}
	assert.Equal(t, exp, got)

	addr = bytes.Repeat([]byte{4}, ContractAddrLen)
	got = GetContractCodeHistoryElementPrefix(addr)
	exp = []byte{
		5,                            // prefix
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4, // address 32 bytes
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
		4, 4,
	}
	assert.Equal(t, exp, got)
}

func TestGetContractByCreatedSecondaryIndexKey(t *testing.T) {
	e := ContractCodeHistoryEntry{
		CodeID:  1,
		Updated: &AbsoluteTxPosition{2 + 1<<(8*7), 3 + 1<<(8*7)},
	}

	// test that contract addresses of 20 length are still supported
	addr := bytes.Repeat([]byte{4}, 20)
	got := GetContractByCreatedSecondaryIndexKey(addr, e)
	exp := []byte{
		6,                      // prefix
		0, 0, 0, 0, 0, 0, 0, 1, // codeID
		1, 0, 0, 0, 0, 0, 0, 2, // height
		1, 0, 0, 0, 0, 0, 0, 3, // index
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4, // address 32 bytes
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	}
	assert.Equal(t, exp, got)

	addr = bytes.Repeat([]byte{4}, ContractAddrLen)
	got = GetContractByCreatedSecondaryIndexKey(addr, e)
	exp = []byte{
		6,                      // prefix
		0, 0, 0, 0, 0, 0, 0, 1, // codeID
		1, 0, 0, 0, 0, 0, 0, 2, // height
		1, 0, 0, 0, 0, 0, 0, 3, // index
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4, // address 32 bytes
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
		4, 4,
	}
	assert.Equal(t, exp, got)
}

func TestGetContractByCreatorThirdIndexKey(t *testing.T) {
	creatorAddr := bytes.Repeat([]byte{4}, 20)

	// test that contract addresses of 20 length are still supported
	contractAddr := bytes.Repeat([]byte{4}, 20)
	got := GetContractByCreatorThirdIndexKey(creatorAddr, contractAddr)
	exp := []byte{
		9,                            // prefix
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4, // creator address
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4, // address 20 bytes
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	}
	assert.Equal(t, exp, got)

	// test that contract addresses of 32 length are still supported
	contractAddr = bytes.Repeat([]byte{4}, 32)
	got = GetContractByCreatorThirdIndexKey(creatorAddr, contractAddr)
	exp = []byte{
		9,                            // prefix
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4, // creator address
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4, // address 32 bytes
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
		4, 4,
	}
	assert.Equal(t, exp, got)
}
