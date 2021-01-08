package wasm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

// ensure store code returns the expected response
func assertStoreCodeResponse(t *testing.T, data []byte, expected int64) {
	var pStoreResp MsgStoreCodeResponse
	require.NoError(t, pStoreResp.Unmarshal(data))
	// TODO: change this when it we store int natively
	require.Equal(t, pStoreResp.CodeID, strconv.FormatInt(expected, 10))
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
