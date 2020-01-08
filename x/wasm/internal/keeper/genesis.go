package keeper

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmwasm/wasmd/x/wasm/internal/types"
	// authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	// "github.com/cosmwasm/wasmd/x/wasm/internal/types"
)

// InitGenesis sets supply information for genesis.
//
// CONTRACT: all types of accounts must have been already initialized/created
func InitGenesis(ctx sdk.Context, keeper Keeper, data types.GenesisState) {
	for id, info := range data.CodeInfos {
		bytecode := data.CodesBytes[id]
		newId, err := keeper.Create(ctx, info.Creator, bytecode)
		if err != nil {
			panic(err)
		}
		newInfo := keeper.GetCodeInfo(ctx, newId)
		if !bytes.Equal(info.CodeHash, newInfo.CodeHash) {
			panic("code hashes not same")
		}
	}

	for i, addr := range data.ContractAddresses {
		info := data.ContractInfos[i]
		state := data.ContractStates[i]

		keeper.setContractInfo(ctx, addr, info)
		keeper.setContractState(ctx, addr, state)
	}

}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, keeper Keeper) types.GenesisState {
	var genState types.GenesisState

	maxCodeID := keeper.GetNextCodeID(ctx, types.KeyLastCodeID)
	for i := uint64(1); i < maxCodeID; i++ {
		genState.CodeInfos = append(genState.CodeInfos, *keeper.GetCodeInfo(ctx, i))
		bytecode, err := keeper.GetByteCode(ctx, i)
		if err != nil {
			panic(err)
		}
		genState.CodesBytes = append(genState.CodesBytes, bytecode)
	}

	keeper.ListContractInfo(ctx, func(addr sdk.AccAddress, contract types.Contract) bool {
		genState.ContractAddresses = append(genState.ContractAddresses, addr)
		genState.ContractInfos = append(genState.ContractInfos, contract)

		contractStateIterator := keeper.GetContractState(ctx, addr)
		var state []types.Model
		for ; contractStateIterator.Valid(); contractStateIterator.Next() {
			m := types.Model{
				Key:   string(contractStateIterator.Key()),
				Value: string(contractStateIterator.Value()),
			}
			state = append(state, m)
		}
		genState.ContractStates = append(genState.ContractStates, state)

		return false
	})

	return genState
}
