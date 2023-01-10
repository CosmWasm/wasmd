package simulation

import (
	"math/rand"

	"github.com/line/lbm-sdk/baseapp"
	sdk "github.com/line/lbm-sdk/types"
	"github.com/line/lbm-sdk/types/module"
	simtypes "github.com/line/lbm-sdk/types/simulation"
	"github.com/line/lbm-sdk/x/simulation"

	wasmappparams "github.com/line/wasmd/app/params"
	"github.com/line/wasmd/x/wasm/keeper/testdata"
	"github.com/line/wasmd/x/wasm/types"
)

// Simulation operation weights constants
const (
	OpWeightMsgStoreCode           = "op_weight_msg_store_code"           //nolint:gosec
	OpWeightMsgInstantiateContract = "op_weight_msg_instantiate_contract" //nolint:gosec
	OpReflectContractPath          = "op_reflect_contract_path"           //nolint:gosec
)

// WasmKeeper is a subset of the wasm keeper used by simulations
type WasmKeeper interface {
	GetParams(ctx sdk.Context) types.Params
	IterateCodeInfos(ctx sdk.Context, cb func(uint64, types.CodeInfo) bool)
}

// WeightedOperations returns all the operations from the module with their respective weights
func WeightedOperations(
	simstate *module.SimulationState,
	ak types.AccountKeeper,
	bk simulation.BankKeeper,
	wasmKeeper WasmKeeper,
) simulation.WeightedOperations {
	var (
		weightMsgStoreCode           int
		weightMsgInstantiateContract int
	)

	simstate.AppParams.GetOrGenerate(simstate.Cdc, OpWeightMsgStoreCode, &weightMsgStoreCode, nil,
		func(_ *rand.Rand) {
			weightMsgStoreCode = wasmappparams.DefaultWeightMsgStoreCode
		},
	)

	simstate.AppParams.GetOrGenerate(simstate.Cdc, OpWeightMsgInstantiateContract, &weightMsgInstantiateContract, nil,
		func(_ *rand.Rand) {
			weightMsgInstantiateContract = wasmappparams.DefaultWeightMsgInstantiateContract
		},
	)

	wasmBz := testdata.ReflectContractWasm()

	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			weightMsgStoreCode,
			SimulateMsgStoreCode(ak, bk, wasmKeeper, wasmBz),
		),
		simulation.NewWeightedOperation(
			weightMsgInstantiateContract,
			SimulateMsgInstantiateContract(ak, bk, wasmKeeper),
		),
	}
}

// SimulateMsgStoreCode generates a MsgStoreCode with random values
func SimulateMsgStoreCode(ak types.AccountKeeper, bk simulation.BankKeeper, wasmKeeper WasmKeeper, wasmBz []byte) simtypes.Operation {
	return func(
		r *rand.Rand,
		app *baseapp.BaseApp,
		ctx sdk.Context,
		accs []simtypes.Account,
		chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if wasmKeeper.GetParams(ctx).CodeUploadAccess.Permission != types.AccessTypeEverybody {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgStoreCode{}.Type(), "no chain permission"), nil, nil
		}

		config := &types.AccessConfig{
			Permission: types.AccessTypeEverybody,
		}

		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := types.MsgStoreCode{
			Sender:                simAccount.Address.String(),
			WASMByteCode:          wasmBz,
			InstantiatePermission: config,
		}

		txCtx := simulation.OperationInput{
			R:             r,
			App:           app,
			TxGen:         wasmappparams.MakeEncodingConfig().TxConfig,
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

// SimulateMsgInstantiateContract generates a MsgInstantiateContract with random values
func SimulateMsgInstantiateContract(ak types.AccountKeeper, bk simulation.BankKeeper, wasmKeeper WasmKeeper) simtypes.Operation {
	return func(
		r *rand.Rand,
		app *baseapp.BaseApp,
		ctx sdk.Context,
		accs []simtypes.Account,
		chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		var codeID uint64
		wasmKeeper.IterateCodeInfos(ctx, func(u uint64, info types.CodeInfo) bool {
			if info.InstantiateConfig.Permission != types.AccessTypeEverybody {
				return false
			}
			codeID = u
			return true
		})

		if codeID == 0 {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgInstantiateContract{}.Type(), "no codes with permission available"), nil, nil
		}

		spendable := bk.SpendableCoins(ctx, simAccount.Address)

		msg := types.MsgInstantiateContract{
			Sender: simAccount.Address.String(),
			Admin:  simtypes.RandomAccounts(r, 1)[0].Address.String(),
			CodeID: codeID,
			Label:  simtypes.RandStringOfLength(r, 10),
			Msg:    []byte(`{}`),
			Funds:  simtypes.RandSubsetCoins(r, spendable),
		}

		txCtx := simulation.OperationInput{
			R:             r,
			App:           app,
			TxGen:         wasmappparams.MakeEncodingConfig().TxConfig,
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
