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
	}
}

// simulate store code proposal (unused now)
// Current problem: out of gas (defaul gas config of gov SimulateMsgSubmitProposal is 10_000_000)
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
