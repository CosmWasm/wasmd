package simulation

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/CosmWasm/wasmd/app/params"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/testdata"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

const (
	WeightStoreCodeProposal                   = "weight_store_code_proposal"
	WeightInstantiateContractProposal         = "weight_instantiate_contract_proposal"
	WeightUpdateAdminProposal                 = "weight_update_admin_proposal"
	WeightExeContractProposal                 = "weight_execute_contract_proposal"
	WeightClearAdminProposal                  = "weight_clear_admin_proposal"
	WeightMigrateContractProposal             = "weight_migrate_contract_proposal"
	WeightSudoContractProposal                = "weight_sudo_contract_proposal"
	WeightPinCodesProposal                    = "weight_pin_codes_proposal"
	WeightUnpinCodesProposal                  = "weight_unpin_codes_proposal"
	WeightUpdateInstantiateConfigProposal     = "weight_update_instantiate_config_proposal"
	WeightStoreAndInstantiateContractProposal = "weight_store_and_instantiate_contract_proposal"
)

func ProposalContents(bk BankKeeper, wasmKeeper WasmKeeper) []simtypes.WeightedProposalContent {
	return []simtypes.WeightedProposalContent{
		// simulation.NewWeightedProposalContent(
		// 	WeightStoreCodeProposal,
		// 	params.DefaultWeightStoreCodeProposal,
		// 	SimulateStoreCodeProposal(wasmKeeper),
		// ),
		simulation.NewWeightedProposalContent(
			WeightInstantiateContractProposal,
			params.DefaultWeightInstantiateContractProposal,
			SimulateInstantiateContractProposal(
				bk,
				wasmKeeper,
				DefaultSimulationCodeIDSelector,
			),
		),
		simulation.NewWeightedProposalContent(
			WeightUpdateAdminProposal,
			params.DefaultWeightUpdateAdminProposal,
			SimulateUpdateAdminProposal(
				wasmKeeper,
				DefaultSimulateUpdateAdminProposalContractSelector,
			),
		),
		simulation.NewWeightedProposalContent(
			WeightExeContractProposal,
			params.DefaultWeightExecuteContractProposal,
			SimulateExecuteContractProposal(
				bk,
				wasmKeeper,
				DefaultSimulationExecuteContractSelector,
				DefaultSimulationExecuteSenderSelector,
				DefaultSimulationExecutePayloader,
			),
		),
		simulation.NewWeightedProposalContent(
			WeightClearAdminProposal,
			params.DefaultWeightClearAdminProposal,
			SimulateClearAdminProposal(
				wasmKeeper,
				DefaultSimulateContractSelector,
			),
		),
		simulation.NewWeightedProposalContent(
			WeightMigrateContractProposal,
			params.DefaultWeightMigrateContractProposal,
			SimulateMigrateContractProposal(
				wasmKeeper,
				DefaultSimulateContractSelector,
				DefaultSimulationCodeIDSelector,
			),
		),
		// simulation.NewWeightedProposalContent(
		//	WeightSudoContractProposal,
		//	params.DefaultWeightSudoContractProposal,
		//	SimulateSudoContractProposal(
		//		wasmKeeper,
		//		DefaultSimulateContractSelector,
		//	),
		// ),
		simulation.NewWeightedProposalContent(
			WeightPinCodesProposal,
			params.DefaultWeightPinCodesProposal,
			SimulatePinContractProposal(
				wasmKeeper,
				DefaultSimulationCodeIDSelector,
			),
		),
		simulation.NewWeightedProposalContent(
			WeightUnpinCodesProposal,
			params.DefaultWeightUnpinCodesProposal,
			SimulateUnpinContractProposal(
				wasmKeeper,
				DefaultSimulationCodeIDSelector,
			),
		),
		simulation.NewWeightedProposalContent(
			WeightUpdateInstantiateConfigProposal,
			params.DefaultWeightUpdateInstantiateConfigProposal,
			SimulateUpdateInstantiateConfigProposal(
				wasmKeeper,
				DefaultSimulationCodeIDSelector,
			),
		),
		// simulation.NewWeightedProposalContent(
		//	WeightStoreAndInstantiateContractProposal,
		//	params.DefaultWeightStoreAndInstantiateContractProposal,
		//	SimulateStoreAndInstantiateContractProposal(
		//		wasmKeeper,
		//	),
		// ),
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
			"",
			"",
			[]byte{},
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

		return types.NewInstantiateContractProposal(
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			simAccount.Address.String(),
			adminAccount.Address.String(),
			codeID,
			simtypes.RandStringOfLength(r, 10),
			[]byte(`{}`),
			sdk.Coins{},
		)
	}
}

// Simulate execute contract proposal
func SimulateExecuteContractProposal(
	bk BankKeeper,
	wasmKeeper WasmKeeper,
	contractSelector MsgExecuteContractSelector,
	senderSelector MsgExecuteSenderSelector,
	payloader MsgExecutePayloader,
) simtypes.ContentSimulatorFn {
	return func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) simtypes.Content {
		ctAddress := contractSelector(ctx, wasmKeeper)
		if ctAddress == nil {
			return nil
		}

		simAccount, err := senderSelector(wasmKeeper, ctx, ctAddress, accs)
		if err != nil {
			return nil
		}

		msg := types.MsgExecuteContract{
			Sender:   simAccount.Address.String(),
			Contract: ctAddress.String(),
			Funds:    sdk.Coins{},
		}

		if err := payloader(&msg); err != nil {
			return nil
		}

		return types.NewExecuteContractProposal(
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			simAccount.Address.String(),
			ctAddress.String(),
			msg.Msg,
			sdk.Coins{},
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

type ClearAdminContractSelector func(sdk.Context, WasmKeeper) sdk.AccAddress

func DefaultSimulateContractSelector(
	ctx sdk.Context,
	wasmKeeper WasmKeeper,
) sdk.AccAddress {
	var contractAddr sdk.AccAddress
	wasmKeeper.IterateContractInfo(ctx, func(address sdk.AccAddress, info types.ContractInfo) bool {
		contractAddr = address
		return true
	})
	return contractAddr
}

// Simulate clear admin proposal
func SimulateClearAdminProposal(wasmKeeper WasmKeeper, contractSelector ClearAdminContractSelector) simtypes.ContentSimulatorFn {
	return func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) simtypes.Content {
		ctAddress := contractSelector(ctx, wasmKeeper)
		if ctAddress == nil {
			return nil
		}

		return types.NewClearAdminProposal(
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			ctAddress.String(),
		)
	}
}

type MigrateContractProposalContractSelector func(sdk.Context, WasmKeeper) sdk.AccAddress

// Simulate migrate contract proposal
func SimulateMigrateContractProposal(wasmKeeper WasmKeeper, contractSelector MigrateContractProposalContractSelector, codeSelector CodeIDSelector) simtypes.ContentSimulatorFn {
	return func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) simtypes.Content {
		ctAddress := contractSelector(ctx, wasmKeeper)
		if ctAddress == nil {
			return nil
		}

		codeID := codeSelector(ctx, wasmKeeper)
		if codeID == 0 {
			return nil
		}

		return types.NewMigrateContractProposal(
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			ctAddress.String(),
			codeID,
			[]byte(`{}`),
		)
	}
}

type SudoContractProposalContractSelector func(sdk.Context, WasmKeeper) sdk.AccAddress

// Simulate sudo contract proposal
func SimulateSudoContractProposal(wasmKeeper WasmKeeper, contractSelector SudoContractProposalContractSelector) simtypes.ContentSimulatorFn {
	return func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) simtypes.Content {
		ctAddress := contractSelector(ctx, wasmKeeper)
		if ctAddress == nil {
			return nil
		}

		return types.NewSudoContractProposal(
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			ctAddress.String(),
			[]byte(`{}`),
		)
	}
}

// Simulate pin contract proposal
func SimulatePinContractProposal(wasmKeeper WasmKeeper, codeSelector CodeIDSelector) simtypes.ContentSimulatorFn {
	return func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) simtypes.Content {
		codeID := codeSelector(ctx, wasmKeeper)
		if codeID == 0 {
			return nil
		}

		return types.NewPinCodesProposal(
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			[]uint64{codeID},
		)
	}
}

// Simulate unpin contract proposal
func SimulateUnpinContractProposal(wasmKeeper WasmKeeper, codeSelector CodeIDSelector) simtypes.ContentSimulatorFn {
	return func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) simtypes.Content {
		codeID := codeSelector(ctx, wasmKeeper)
		if codeID == 0 {
			return nil
		}

		return types.NewUnpinCodesProposal(
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			[]uint64{codeID},
		)
	}
}

// Simulate update instantiate config proposal
func SimulateUpdateInstantiateConfigProposal(wasmKeeper WasmKeeper, codeSelector CodeIDSelector) simtypes.ContentSimulatorFn {
	return func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) simtypes.Content {
		codeID := codeSelector(ctx, wasmKeeper)
		if codeID == 0 {
			return nil
		}

		simAccount, _ := simtypes.RandomAcc(r, accs)
		permission := wasmKeeper.GetParams(ctx).InstantiateDefaultPermission
		config := permission.With(simAccount.Address)

		configUpdate := types.AccessConfigUpdate{
			CodeID:                codeID,
			InstantiatePermission: config,
		}

		return types.NewUpdateInstantiateConfigProposal(
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			configUpdate,
		)
	}
}

func SimulateStoreAndInstantiateContractProposal(wasmKeeper WasmKeeper) simtypes.ContentSimulatorFn {
	return func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) simtypes.Content {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		adminAccount, _ := simtypes.RandomAcc(r, accs)

		wasmBz := testdata.ReflectContractWasm()
		permission := wasmKeeper.GetParams(ctx).InstantiateDefaultPermission.With(simAccount.Address)

		return types.NewStoreAndInstantiateContractProposal(
			simtypes.RandStringOfLength(r, 10),
			simtypes.RandStringOfLength(r, 10),
			simAccount.Address.String(),
			wasmBz,
			"",
			"",
			[]byte{},
			&permission,
			false,
			adminAccount.Address.String(),
			simtypes.RandStringOfLength(r, 10),
			[]byte(`{}`),
			sdk.Coins{},
		)
	}
}
