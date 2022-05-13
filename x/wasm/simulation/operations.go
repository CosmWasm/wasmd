package simulation

import (
	"io/ioutil"
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

// Simulation operation weights constants
//nolint:gosec
const (
	OpWeightMsgStoreCode           = "op_weight_msg_store_code"
	OpWeightMsgInstantiateContract = "op_weight_msg_instantiate_contract"
	OpReflectContractPath          = "op_reflect_contract_path"
)

// WasmKeeper is a subset of the wasm keeper used by simulations
type WasmKeeper interface {
	GetParams(ctx sdk.Context) types.Params
	IterateCodeInfos(ctx sdk.Context, cb func(uint64, types.CodeInfo) bool)
}
type BankKeeper interface {
	simulation.BankKeeper
	IsSendEnabledCoin(ctx sdk.Context, coin sdk.Coin) bool
}

// WeightedOperations returns all the operations from the module with their respective weights
func WeightedOperations(
	simstate *module.SimulationState,
	ak types.AccountKeeper,
	bk BankKeeper,
	wasmKeeper WasmKeeper,
) simulation.WeightedOperations {
	var (
		weightMsgStoreCode           int
		weightMsgInstantiateContract int
		wasmContractPath             string
	)

	simstate.AppParams.GetOrGenerate(simstate.Cdc, OpWeightMsgStoreCode, &weightMsgStoreCode, nil,
		func(_ *rand.Rand) {
			weightMsgStoreCode = params.DefaultWeightMsgStoreCode
		},
	)

	simstate.AppParams.GetOrGenerate(simstate.Cdc, OpWeightMsgInstantiateContract, &weightMsgInstantiateContract, nil,
		func(_ *rand.Rand) {
			weightMsgInstantiateContract = params.DefaultWeightMsgInstantiateContract
		},
	)
	simstate.AppParams.GetOrGenerate(simstate.Cdc, OpReflectContractPath, &wasmContractPath, nil,
		func(_ *rand.Rand) {
			// simulations are run from the `app` folder
			wasmContractPath = "../x/wasm/keeper/testdata/reflect.wasm"
		},
	)

	wasmBz, err := ioutil.ReadFile(wasmContractPath)
	if err != nil {
		panic(err)
	}

	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			weightMsgStoreCode,
			SimulateMsgStoreCode(ak, bk, wasmKeeper, wasmBz, 5_000_000),
		),
		simulation.NewWeightedOperation(
			weightMsgInstantiateContract,
			SimulateMsgInstantiateContract(ak, bk, wasmKeeper, DefaultSimulationCodeIDSelector),
		),
	}
}

// SimulateMsgStoreCode generates a MsgStoreCode with random values
func SimulateMsgStoreCode(ak types.AccountKeeper, bk simulation.BankKeeper, wasmKeeper WasmKeeper, wasmBz []byte, gas uint64) simtypes.Operation {
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

		simAccount, _ := simtypes.RandomAcc(r, accs)

		permission := wasmKeeper.GetParams(ctx).InstantiateDefaultPermission
		config := permission.With(simAccount.Address)

		msg := types.MsgStoreCode{
			Sender:                simAccount.Address.String(),
			WASMByteCode:          wasmBz,
			InstantiatePermission: &config,
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

		return GenAndDeliverTxWithRandFees(txCtx, gas)
	}
}

// CodeIDSelector returns code id to be used in simulations
type CodeIDSelector = func(ctx sdk.Context, wasmKeeper WasmKeeper) uint64

// DefaultSimulationCodeIDSelector picks the first code id
func DefaultSimulationCodeIDSelector(ctx sdk.Context, wasmKeeper WasmKeeper) uint64 {
	var codeID uint64
	wasmKeeper.IterateCodeInfos(ctx, func(u uint64, info types.CodeInfo) bool {
		if info.InstantiateConfig.Permission != types.AccessTypeEverybody {
			return false
		}
		codeID = u
		return true
	})
	return codeID
}

// SimulateMsgInstantiateContract generates a MsgInstantiateContract with random values
func SimulateMsgInstantiateContract(ak types.AccountKeeper, bk BankKeeper, wasmKeeper WasmKeeper, codeSelector CodeIDSelector) simtypes.Operation {
	return func(
		r *rand.Rand,
		app *baseapp.BaseApp,
		ctx sdk.Context,
		accs []simtypes.Account,
		chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		codeID := codeSelector(ctx, wasmKeeper)
		if codeID == 0 {
			return simtypes.NoOpMsg(types.ModuleName, types.MsgInstantiateContract{}.Type(), "no codes with permission available"), nil, nil
		}
		deposit := sdk.Coins{}
		spendableCoins := bk.SpendableCoins(ctx, simAccount.Address)
		for _, v := range spendableCoins {
			if bk.IsSendEnabledCoin(ctx, v) {
				deposit = deposit.Add(simtypes.RandSubsetCoins(r, sdk.NewCoins(v))...)
			}
		}

		msg := types.MsgInstantiateContract{
			Sender: simAccount.Address.String(),
			Admin:  simtypes.RandomAccounts(r, 1)[0].Address.String(),
			CodeID: codeID,
			Label:  simtypes.RandStringOfLength(r, 10),
			Msg:    []byte(`{}`),
			Funds:  deposit,
		}

		txCtx := simulation.OperationInput{
			R:               r,
			App:             app,
			TxGen:           simappparams.MakeTestEncodingConfig().TxConfig,
			Cdc:             nil,
			Msg:             &msg,
			MsgType:         msg.Type(),
			Context:         ctx,
			SimAccount:      simAccount,
			AccountKeeper:   ak,
			Bankkeeper:      bk,
			ModuleName:      types.ModuleName,
			CoinsSpentInMsg: deposit,
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}
