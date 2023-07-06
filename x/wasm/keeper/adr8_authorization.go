package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/xadr8"
)

var _ xadr8.Authorization = &AcceptOnlyContracts{}

type AcceptOnlyContracts struct {
	k *Keeper
}

func NewAcceptOnlyContracts(k *Keeper) *AcceptOnlyContracts {
	return &AcceptOnlyContracts{k: k}
}

func (a AcceptOnlyContracts) IsAuthorized(ctx sdk.Context, actor sdk.AccAddress) (bool, error) {
	return a.k.HasContractInfo(ctx, actor), nil
}
