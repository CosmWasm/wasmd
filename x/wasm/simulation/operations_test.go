package simulation

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/line/lbm-sdk/types/module"
	"github.com/line/lbm-sdk/x/simulation"

	wasmappparams "github.com/line/wasmd/app/params"
	"github.com/line/wasmd/x/wasm/keeper"
	"github.com/line/wasmd/x/wasm/types"
)

func TestWeightedOperations(t *testing.T) {
	type args struct {
		simstate   *module.SimulationState
		ak         types.AccountKeeper
		bk         BankKeeper
		wasmKeeper WasmKeeper
		wasmBz     []byte
	}

	params := args{
		simstate:   &module.SimulationState{},
		wasmKeeper: makeKeeper(t).WasmKeeper,
	}

	tests := []struct {
		name string
		args args
		want simulation.WeightedOperations
	}{
		{
			name: "execute success",
			args: args{
				simstate: &module.SimulationState{},
			},
			want: simulation.WeightedOperations{
				simulation.NewWeightedOperation(
					wasmappparams.DefaultWeightMsgStoreCode,
					SimulateMsgStoreCode(params.ak, params.bk, params.wasmKeeper, params.wasmBz, 5_000_000)),
				simulation.NewWeightedOperation(
					wasmappparams.DefaultWeightMsgInstantiateContract,
					SimulateMsgInstantiateContract(params.ak, params.bk, params.wasmKeeper, DefaultSimulationCodeIDSelector)),
				simulation.NewWeightedOperation(
					wasmappparams.DefaultWeightMsgExecuteContract,
					SimulateMsgExecuteContract(params.ak, params.bk, params.wasmKeeper,
						DefaultSimulationExecuteContractSelector, DefaultSimulationExecuteSenderSelector,
						DefaultSimulationExecutePayloader)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WeightedOperations(tt.args.simstate, tt.args.ak, tt.args.bk, tt.args.wasmKeeper)
			for i := range got {
				require.Equal(t, tt.want[i].Weight(), got[i].Weight(), "WeightedOperations().Weight()")

				expected := reflect.TypeOf(tt.want[i].Op()).String()
				actual := reflect.TypeOf(got[i].Op()).String()

				require.Equal(t, expected, actual, "return value type should be the same")
			}
		})
	}
}

// Copy from keeper_test.go
const SupportedFeatures = "iterator,staking,stargate"

// Copy from keeper_test.go
func makeKeeper(t *testing.T) keeper.TestKeepers {
	_, keepers := keeper.CreateTestInput(t, false, SupportedFeatures)
	return keepers
}
