package keeper

import (
	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/Finschia/finschia-sdk/types"
	sdkerrors "github.com/Finschia/finschia-sdk/types/errors"

	wasmkeeper "github.com/Finschia/wasmd/x/wasm/keeper"
	"github.com/Finschia/wasmd/x/wasmplus/types"
)

func InitGenesis(
	ctx sdk.Context,
	keeper *Keeper,
	data types.GenesisState,
	stakingKeeper wasmkeeper.ValidatorSetSource,
	msgHandler sdk.Handler,
) ([]abci.ValidatorUpdate, error) {
	result, err := wasmkeeper.InitGenesis(ctx, &keeper.Keeper, data.RawWasmState(), stakingKeeper, msgHandler)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "wasm")
	}

	// set InactiveContractAddresses
	for i, contractAddr := range data.InactiveContractAddresses {
		inactiveContractAddr := sdk.MustAccAddressFromBech32(contractAddr)
		err = keeper.deactivateContract(ctx, inactiveContractAddr)
		if err != nil {
			return nil, sdkerrors.Wrapf(err, "contract number %d", i)
		}
	}

	return result, nil
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, keeper *Keeper) *types.GenesisState {
	wasmState := wasmkeeper.ExportGenesis(ctx, &keeper.Keeper)

	genState := types.GenesisState{
		Params:    wasmState.Params,
		Codes:     wasmState.Codes,
		Contracts: wasmState.Contracts,
		Sequences: wasmState.Sequences,
		GenMsgs:   wasmState.GenMsgs,
	}

	keeper.IterateInactiveContracts(ctx, func(contractAddr sdk.AccAddress) (stop bool) {
		genState.InactiveContractAddresses = append(genState.InactiveContractAddresses, contractAddr.String())
		return false
	})

	return &genState
}
