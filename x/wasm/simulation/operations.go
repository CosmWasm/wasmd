package simulation

import (
	_ "embed"
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/CosmWasm/wasmd/app/params"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

//go:embed testdata/reflect.wasm
var reflectContract []byte

// Simulation operation weights constants
//nolint:gosec
const (
	OpWeightMsgStoreCode = "op_weight_msg_store_code"
)

// ParameterSource is a subset of the wasm keeper used by simulations
type ParameterSource interface {
	GetParams(ctx sdk.Context) types.Params
}

// WeightedOperations returns all the operations from the module with their respective weights
func WeightedOperations(
	simstate *module.SimulationState,
	ak types.AccountKeeper,
	bk simulation.BankKeeper,
	wasmParamSource ParameterSource,
) simulation.WeightedOperations {
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
			SimulateMsgStoreCode(ak, bk, wasmParamSource),
		),
	}
}

// SimulateMsgStoreCode generates a MsgStoreCode with random values
func SimulateMsgStoreCode(ak types.AccountKeeper, bk simulation.BankKeeper, wasmParamSource ParameterSource) simtypes.Operation {
	return func(
		r *rand.Rand,
		app *baseapp.BaseApp,
		ctx sdk.Context,
		accs []simtypes.Account,
		chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if wasmParamSource.GetParams(ctx).CodeUploadAccess.Permission != types.AccessTypeEverybody {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgStoreCode{}.Type(), "no chain permission"), nil, nil
		}

		config := &types.AccessConfig{
			Permission: types.AccessTypeEverybody,
		}

		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := types.MsgStoreCode{
			Sender:                simAccount.Address.String(),
			WASMByteCode:          reflectContract,
			InstantiatePermission: config,
		}

		txCtx := simulation.OperationInput{
			R:             r,
			App:           app,
			TxGen:         simappparams.MakeTestEncodingConfig().TxConfig,
			Cdc:           nil,
			Msg:           &msg,
			MsgType:       msg.Type(),
			Context:       ctx,
			SimAccount:    simAccount,
			AccountKeeper: ak,
			Bankkeeper:    bk,
			ModuleName:    types.ModuleName,
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}
