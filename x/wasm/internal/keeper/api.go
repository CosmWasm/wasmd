package keeper

import (
	"fmt"
	wasmvm "github.com/CosmWasm/wasmvm"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	CostHumanize  = 5 * GasMultiplier
	CostCanonical = 4 * GasMultiplier
)

func humanAddress(canon []byte) (string, uint64, error) {
	if len(canon) != sdk.AddrLen {
		return "", CostHumanize, fmt.Errorf("Expected %d byte address", sdk.AddrLen)
	}
	return sdk.AccAddress(canon).String(), CostHumanize, nil
}

func canonicalAddress(human string) ([]byte, uint64, error) {
	bz, err := sdk.AccAddressFromBech32(human)
	return bz, CostCanonical, err
}

var cosmwasmAPI = wasmvm.GoAPI{
	HumanAddress:     humanAddress,
	CanonicalAddress: canonicalAddress,
}
