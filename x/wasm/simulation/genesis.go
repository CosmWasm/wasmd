package simulation

import (
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// RandomizeGenState generates a random GenesisState for wasm
func RandomizedGenState(simstate *module.SimulationState) {
	wasmGenesis := types.GenesisState{
		Params: types.Params{
			CodeUploadAccess:             RandomizeAccessConfig(simstate.Rand),
			InstantiateDefaultPermission: RandomizeAccessType(simstate.Rand),
			MaxWasmCodeSize:              RandomizeMaxWasmCodeSize(simstate.Rand),
		},
		Codes:     nil,
		Contracts: nil,
		Sequences: nil,
		GenMsgs:   nil,
	}

	_, err := simstate.Cdc.MarshalJSON(&wasmGenesis)
	if err != nil {
		panic(err)
	}

	simstate.GenState[types.ModuleName] = simstate.Cdc.MustMarshalJSON(&wasmGenesis)
}
