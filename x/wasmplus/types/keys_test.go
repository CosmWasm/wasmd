package types

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetInactiveContractKey(t *testing.T) {
	addr := bytes.Repeat([]byte{4}, 20)
	got := GetInactiveContractKey(addr)
	exp := []byte{
		0x90,                         // prefix
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4, // address 20 bytes
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	}
	assert.Equal(t, exp, got)
}
