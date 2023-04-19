package simulation

import (
	"github.com/Finschia/finschia-sdk/types/module"

	"github.com/Finschia/wasmd/x/wasm/types"
)

// RandomizedGenState generates a random GenesisState for wasm
func RandomizedGenState(simstate *module.SimulationState) {
	params := types.DefaultParams()
	wasmGenesis := types.GenesisState{
		Params:    params,
		Codes:     nil,
		Contracts: nil,
		Sequences: []types.Sequence{
			{IDKey: types.KeyLastCodeID, Value: simstate.Rand.Uint64()},
		},
		GenMsgs: nil,
	}

	_, err := simstate.Cdc.MarshalJSON(&wasmGenesis)
	if err != nil {
		panic(err)
	}

	simstate.GenState[types.ModuleName] = simstate.Cdc.MustMarshalJSON(&wasmGenesis)
}
