package simulation

import (
	"math/rand"
	"os"

	"github.com/CosmWasm/wasmd/app/params"
	"github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

const (
	OpWeightStoreCodeProposal = "op_weight_store_code_proposal"
)

func ProposalContents(wasmKeeper WasmKeeper) []simtypes.WeightedProposalContent {
	return []simtypes.WeightedProposalContent{
		simulation.NewWeightedProposalContent(
			OpWeightStoreCodeProposal,
			params.DefaultWeightStoreCodeProposal,
			SimulatetoreCodeProposal(wasmKeeper),
		),
	}
}

func SimulatetoreCodeProposal(wasmKeeper WasmKeeper) simtypes.ContentSimulatorFn {
	return func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) simtypes.Content {
		simAccount, _ := simtypes.RandomAcc(r, accs)

		var wasmBz []byte
		var err error
		wasmBz, err = os.ReadFile("./../x/wasm/keeper/testdata/reflect_1_1.wasm")
		if err != nil {
			panic(err)
		}

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
