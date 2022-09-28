package keeper

import (
	"github.com/CosmWasm/wasmd/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)

	for _, genDenom := range genState.GetFactoryDenoms() {
		_, _, err := types.DeconstructDenom(genDenom.GetDenom())
		if err != nil {
			panic(err)
		}
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	genDenoms := []types.GenesisDenom{}

	return &types.GenesisState{
		FactoryDenoms: genDenoms,
		Params:        k.GetParams(ctx),
	}
}
