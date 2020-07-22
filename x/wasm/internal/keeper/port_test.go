package keeper

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortIDForContract(t *testing.T) {
	for i := 0; i < 25; i++ {
		codeID, instanceID := uint64(rand.Uint32()), uint64(rand.Uint32())
		portID := PortIDForContract(codeID, instanceID)
		require.True(t, len(portID) <= 20, portID)
		gotContractAddr, err := ContractFromPortID(portID)
		require.NoError(t, err)
		assert.Equal(t, contractAddress(codeID, instanceID), gotContractAddr)
	}
}
