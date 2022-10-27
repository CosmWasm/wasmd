package simulation

import (
	"math/rand"

	"github.com/CosmWasm/wasmd/app/params"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/testdata"
	"github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

const (
	OpWeightStoreCodeProposal           = "op_weight_store_code_proposal"
	OpWeightInstantiateContractProposal = "op_weight_instantiate_contract_proposal"
	OpWeightUpdateAdminProposal         = "op_weight_update_admin_proposal"
	OpWeightExecuteContractProposal     = "op_weight_execute_contract_proposal"
)

func ProposalContents(bk BankKeeper, wasmKeeper WasmKeeper) []simtypes.WeightedProposalContent {
	return []simtypes.WeightedProposalContent{
		// simulation.NewWeightedProposalContent(
		// 	OpWeightStoreCodeProposal,
		// 	params.DefaultWeightStoreCodeProposal,
		// 	SimulateStoreCodeProposal(wasmKeeper),
		// ),
		simulation.NewWeightedProposalContent(
			OpWeightInstantiateContractProposal,
			params.DefaultWeightInstantiateContractProposal,
			SimulateInstantiateContractProposal(
				bk,
				wasmKeeper,
				DefaultSimulationCodeIDSelector,
			),
		),
		simulation.NewWeightedProposalContent(
			OpWeightUpdateAdminProposal,
			params.DefaultWeightUpdateAdminProposal,
			SimulateUpdateAdminProposal(
				wasmKeeper,
				DefaultSimulateUpdateAdminProposalContractSelector,
			),
		),
		simulation.NewWeightedProposalContent(
			OpWeightExecuteContractProposal,
			params.DefaultWeightExecuteContractProposal,
			SimulateExecuteContractProposal(
				bk,
				wasmKeeper,
				DefaultSimulationExecuteContractSelector,
			),
		),
	}
}

// simulate store code proposal (unused now)
// Current problem: out of gas (defaul gaswanted config of gov SimulateMsgSubmitProposal is 10_000_000)
// but this proposal may need more than it
func SimulateStoreCodeProposal(wasmKeeper WasmKeeper) simtypes.ContentSimulatorFn {
	return func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) simtypes.Content {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		wasmBz := testdata.ReflectContractWasm()

		permission := wasmKeeper.GetParams(ctx).InstantiateDefaultPermission.With(simAccount.Address)

		return types.NewStoreCodeProposal(
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			simAccount.Address.String(),
			wasmBz,
			&permission,
			false,
		)
	}
}

// Simulate instantiate contract proposal
func SimulateInstantiateContractProposal(bk BankKeeper, wasmKeeper WasmKeeper, codeSelector CodeIDSelector) simtypes.ContentSimulatorFn {
	return func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) simtypes.Content {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		// admin
		adminAccount, _ := simtypes.RandomAcc(r, accs)
		// get codeID
		codeID := codeSelector(ctx, wasmKeeper)
		if codeID == 0 {
			return nil
		}

		deposit := sdk.Coins{}
		spendableCoins := bk.SpendableCoins(ctx, simAccount.Address)
		for _, v := range spendableCoins {
			if bk.IsSendEnabledCoin(ctx, v) {
				deposit = deposit.Add(simtypes.RandSubsetCoins(r, sdk.NewCoins(v))...)
			}
		}

		return types.NewInstantiateContractProposal(
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			simAccount.Address.String(),
			adminAccount.Address.String(),
			codeID,
			simtypes.RandStringOfLength(r, 10),
			[]byte(`{}`),
			deposit,
		)
	}
}

func SimulateExecuteContractProposal(bk BankKeeper, wasmKeeper WasmKeeper, contractSelector MsgExecuteContractSelector) simtypes.ContentSimulatorFn {
	return func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) simtypes.Content {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		ctAddress := contractSelector(ctx, wasmKeeper)

		deposit := sdk.Coins{}
		spendableCoins := bk.SpendableCoins(ctx, simAccount.Address)
		for _, v := range spendableCoins {
			if bk.IsSendEnabledCoin(ctx, v) {
				deposit = deposit.Add(simtypes.RandSubsetCoins(r, sdk.NewCoins(v))...)
			}
		}

		return types.NewExecuteContractProposal(
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			simAccount.Address.String(),
			ctAddress.String(),
			[]byte(`{}`),
			deposit,
		)
	}
}

type UpdateAdminContractSelector func(sdk.Context, WasmKeeper, string) (sdk.AccAddress, types.ContractInfo)

func DefaultSimulateUpdateAdminProposalContractSelector(
	ctx sdk.Context,
	wasmKeeper WasmKeeper,
	adminAddress string,
) (sdk.AccAddress, types.ContractInfo) {
	var contractAddr sdk.AccAddress
	var contractInfo types.ContractInfo
	wasmKeeper.IterateContractInfo(ctx, func(address sdk.AccAddress, info types.ContractInfo) bool {
		if info.Admin != adminAddress {
			return false
		}
		contractAddr = address
		contractInfo = info
		return true
	})
	return contractAddr, contractInfo
}

// Simulate update admin contract proposal
func SimulateUpdateAdminProposal(wasmKeeper WasmKeeper, contractSelector UpdateAdminContractSelector) simtypes.ContentSimulatorFn {
	return func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) simtypes.Content {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		ctAddress, _ := contractSelector(ctx, wasmKeeper, simAccount.Address.String())
		if ctAddress == nil {
			return nil
		}

		return types.NewUpdateAdminProposal(
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandomAccounts(r, 1)[0].Address.String(),
			ctAddress.String(),
		)
	}
}
