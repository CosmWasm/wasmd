package types

import (
	sdk "github.com/line/lbm-sdk/types"

	wasmtypes "github.com/line/wasmd/x/wasm/types"
)

const (
	// ModuleName is the name of this module.
	ModuleName = wasmtypes.ModuleName

	// RouterKey is used to route governance proposals
	RouterKey = wasmtypes.RouterKey

	// StoreKey is the prefix under which we store this module's data
	StoreKey = wasmtypes.StoreKey
)

var (
	InactiveContractPrefix = []byte{0x90}
)

func GetInactiveContractKey(contractAddress sdk.AccAddress) []byte {
	key := make([]byte, len(InactiveContractPrefix)+len(contractAddress))
	copy(key, InactiveContractPrefix)
	copy(key[len(InactiveContractPrefix):], contractAddress)
	return key
}
