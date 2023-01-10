package simulation

import (
	"encoding/json"
	"fmt"

	"github.com/line/lbm-sdk/types/module"

	"github.com/line/wasmd/x/wasm/types"
)

// RandomizedGenState RandomizeGenState generates a random GenesisState for wasm
func RandomizedGenState(simState *module.SimulationState) {
	params := RandomParams(simState.Rand)
	wasmGenesis := types.GenesisState{
		Params:    params,
		Codes:     nil,
		Contracts: nil,
		Sequences: []types.Sequence{
			{IDKey: types.KeyLastCodeID, Value: simState.Rand.Uint64()},
			{IDKey: types.KeyLastInstanceID, Value: simState.Rand.Uint64()},
		},
		GenMsgs: nil,
	}

	bz, err := json.MarshalIndent(&wasmGenesis.Params, "", " ")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Selected randomly generated wasm parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&wasmGenesis)
}
