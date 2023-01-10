package wasm

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/line/lbm-sdk/types"
)

// ensure store code returns the expected response
func assertStoreCodeResponse(t *testing.T, data []byte, expected uint64) {
	var pStoreResp MsgStoreCodeResponse
	require.NoError(t, pStoreResp.Unmarshal(data))
	require.Equal(t, pStoreResp.CodeID, expected)
}

// ensure execution returns the expected data
func assertExecuteResponse(t *testing.T, data []byte, expected []byte) {
	var pExecResp MsgExecuteContractResponse
	require.NoError(t, pExecResp.Unmarshal(data))
	require.Equal(t, pExecResp.Data, expected)
}

// ensures this returns a valid bech32 address and returns it
func parseInitResponse(t *testing.T, data []byte) string {
	var pInstResp MsgInstantiateContractResponse
	require.NoError(t, pInstResp.Unmarshal(data))
	require.NotEmpty(t, pInstResp.Address)
	addr := pInstResp.Address
	// ensure this is a valid sdk address
	_, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	return addr
}

// ensures this returns a valid codeID and bech32 address and returns it
func parseStoreAndInitResponse(t *testing.T, data []byte) (uint64, string) {
	var res MsgStoreCodeAndInstantiateContractResponse
	require.NoError(t, res.Unmarshal(data))
	require.NotEmpty(t, res.CodeID)
	require.NotEmpty(t, res.Address)
	addr := res.Address
	codeID := res.CodeID
	// ensure this is a valid sdk address
	_, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	return codeID, addr
}
