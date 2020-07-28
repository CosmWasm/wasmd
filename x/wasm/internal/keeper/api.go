package keeper

import (
	"fmt"

	cosmwasm "github.com/CosmWasm/go-cosmwasm"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type cosmwasmAPIImpl struct {
	gasMultiplier uint64
}

func (a cosmwasmAPIImpl) humanAddress(canon []byte) (string, uint64, error) {
	gas := 5 * a.gasMultiplier
	if len(canon) != sdk.AddrLen {
		return "", gas, fmt.Errorf("Expected %d byte address", sdk.AddrLen)
	}
	return sdk.AccAddress(canon).String(), gas, nil
}

func (a cosmwasmAPIImpl) canonicalAddress(human string) ([]byte, uint64, error) {
	bz, err := sdk.AccAddressFromBech32(human)
	return bz, 4 * a.gasMultiplier, err
}

func (k Keeper) cosmwasmAPI(ctx sdk.Context) cosmwasm.GoAPI {
	x := cosmwasmAPIImpl{
		gasMultiplier: k.getGasMultiplier(ctx),
	}
	return cosmwasm.GoAPI{
		HumanAddress:     x.humanAddress,
		CanonicalAddress: x.canonicalAddress,
	}
}
