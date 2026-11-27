package simulation_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/CosmWasm/wasmd/x/wasm/simulation"
)

func TestWeightedOperationsWithAppParams(t *testing.T) {
	specs := map[string]struct {
		appParams simtypes.AppParams
		exp       []int
	}{
		"defaults": {
			appParams: simtypes.AppParams{},
			exp: []int{
				simulation.DefaultWeightMsgStoreCode,
				simulation.DefaultWeightMsgInstantiateContract,
				simulation.DefaultWeightMsgExecuteContract,
				simulation.DefaultWeightMsgUpdateAdmin,
				simulation.DefaultWeightMsgClearAdmin,
				simulation.DefaultWeightMsgMigrateContract,
			},
		},
		"custom weights": {
			appParams: simtypes.AppParams{
				simulation.OpWeightMsgStoreCode:           json.RawMessage("1"),
				simulation.OpWeightMsgInstantiateContract: json.RawMessage("2"),
				simulation.OpWeightMsgExecuteContract:     json.RawMessage("3"),
				simulation.OpWeightMsgUpdateAdmin:         json.RawMessage("4"),
				simulation.OpWeightMsgClearAdmin:          json.RawMessage("5"),
				simulation.OpWeightMsgMigrateContract:     json.RawMessage("6"),
			},
			exp: []int{1, 2, 3, 4, 5, 6},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			ops := simulation.WeightedOperations(spec.appParams, nil, nil, nil)
			require.Len(t, ops, len(spec.exp))
			for i, op := range ops {
				assert.Equal(t, spec.exp[i], op.Weight(), "operation %d", i)
			}
		})
	}
}
