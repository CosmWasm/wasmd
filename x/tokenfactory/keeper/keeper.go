package keeper

import (
	"github.com/CosmWasm/wasmd/x/tokenfactory/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

type Keeper struct {
	cdc        codec.Codec
	storeKey   sdk.StoreKey
	paramSpace paramtypes.Subspace

	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
}

func NewKeeper(
	cdc codec.Codec,
	storeKey sdk.StoreKey,
	paramSpace paramtypes.Subspace,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
) Keeper {
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		paramSpace: paramSpace,

		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
	}
}
