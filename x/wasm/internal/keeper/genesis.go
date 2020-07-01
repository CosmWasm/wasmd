package keeper

import (
	"bytes"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	// authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	// "github.com/CosmWasm/wasmd/x/wasm/internal/types"
)

// InitGenesis sets supply information for genesis.
//
// CONTRACT: all types of accounts must have been already initialized/created
func InitGenesis(ctx sdk.Context, keeper Keeper, data types.GenesisState) error {
	for i, code := range data.Codes {
		newId, err := keeper.Create(ctx, code.CodeInfo.Creator, code.CodesBytes, code.CodeInfo.Source, code.CodeInfo.Builder)
		if err != nil {
			return sdkerrors.Wrapf(err, "code number %d", i)

		}
		newInfo := keeper.GetCodeInfo(ctx, newId)
		if !bytes.Equal(code.CodeInfo.CodeHash, newInfo.CodeHash) {
			return sdkerrors.Wrap(types.ErrInvalid, "code hashes not same")
		}
	}

	for i, contract := range data.Contracts {
		err := keeper.importContract(ctx, contract.ContractAddress, &contract.ContractInfo, contract.ContractState)
		if err != nil {
			return sdkerrors.Wrapf(err, "contract number %d", i)
		}
	}

	for i, seq := range data.Sequences {
		err := keeper.importAutoIncrementID(ctx, seq.IDKey, seq.Value)
		if err != nil {
			return sdkerrors.Wrapf(err, "sequence number %d", i)
		}
	}
	return nil
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, keeper Keeper) types.GenesisState {
	var genState types.GenesisState

	maxCodeID := keeper.GetNextCodeID(ctx)
	for i := uint64(1); i < maxCodeID; i++ {
		bytecode, err := keeper.GetByteCode(ctx, i)
		if err != nil {
			panic(err)
		}
		genState.Codes = append(genState.Codes, types.Code{
			CodeInfo:   *keeper.GetCodeInfo(ctx, i),
			CodesBytes: bytecode,
		})
	}

	keeper.ListContractInfo(ctx, func(addr sdk.AccAddress, contract types.ContractInfo) bool {
		contractStateIterator := keeper.GetContractState(ctx, addr)
		var state []types.Model
		for ; contractStateIterator.Valid(); contractStateIterator.Next() {
			m := types.Model{
				Key:   contractStateIterator.Key(),
				Value: contractStateIterator.Value(),
			}
			state = append(state, m)
		}

		genState.Contracts = append(genState.Contracts, types.Contract{
			ContractAddress: addr,
			ContractInfo:    contract,
			ContractState:   state,
		})

		return false
	})

	// types.KeyLastCodeID is updated via keeper create
	for _, k := range [][]byte{types.KeyLastInstanceID} {
		genState.Sequences = append(genState.Sequences, types.Sequence{
			IDKey: k,
			Value: keeper.peekAutoIncrementID(ctx, k),
		})
	}

	return genState
}
