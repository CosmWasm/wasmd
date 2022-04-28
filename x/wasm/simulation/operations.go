package simulation

import (
	_ "embed"
	"math/rand"

	"github.com/CosmWasm/wasmd/app/params"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

//go:embed testdata/reflect.wasm
var reflectContract []byte

// Simulation operation weights constants
//nolint:gosec
const (
	OpWeightMsgStoreCode = "op_weight_msg_store_code"
)

// WeightedOperations returns all the operations from the module with their respective weights
func WeightedOperations(
	simstate *module.SimulationState, ak types.AccountKeeper) simulation.WeightedOperations {
	var (
		weightMsgStoreCode int
	)

	simstate.AppParams.GetOrGenerate(simstate.Cdc, OpWeightMsgStoreCode, &weightMsgStoreCode, nil,
		func(_ *rand.Rand) {
			weightMsgStoreCode = params.DefaultWeightMsgStoreCode
		},
	)

	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			weightMsgStoreCode,
			SimulateMsgStoreCode(ak),
		),
	}
}

// SimulateMsgStoreCode generates a MsgStoreCode with random values
func SimulateMsgStoreCode(ak types.AccountKeeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {

		simAccount, _ := simtypes.RandomAcc(r, accs)

		config := &types.AccessConfig{
			Permission: 3,
			Address:    keeper.RandomBech32AccountAddress(nil),
		}

		msg := types.MsgStoreCode{
			Sender:                keeper.RandomBech32AccountAddress(nil),
			WASMByteCode:          reflectContract,
			InstantiatePermission: config,
		}

		txCtx := simulation.OperationInput{
			App:           app,
			TxGen:         simappparams.MakeTestEncodingConfig().TxConfig,
			Cdc:           nil,
			Msg:           &msg,
			MsgType:       msg.Type(),
			Context:       ctx,
			SimAccount:    simAccount,
			AccountKeeper: ak,
			ModuleName:    types.ModuleName,
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}
