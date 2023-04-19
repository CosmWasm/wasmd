package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/Finschia/finschia-sdk/types"

	"github.com/Finschia/wasmd/x/wasmplus/types"
)

func TestValidateDeactivateContractProposal(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, "staking")
	govKeeper, wasmKeeper := keepers.GovKeeper, keepers.WasmKeeper

	example := InstantiateHackatomExampleContract(t, ctx, keepers)

	src := types.DeactivateContractProposal{
		Title:       "Foo",
		Description: "Bar",
		Contract:    example.Contract.String(),
	}

	em := sdk.NewEventManager()

	// when stored
	storedProposal, err := govKeeper.SubmitProposal(ctx, &src)
	require.NoError(t, err)

	// proposal execute
	handler := govKeeper.Router().GetRoute(storedProposal.ProposalRoute())
	err = handler(ctx.WithEventManager(em), storedProposal.GetContent())
	require.NoError(t, err)

	// then
	isInactive := wasmKeeper.IsInactiveContract(ctx, example.Contract)
	require.True(t, isInactive)
}

func TestActivateContractProposal(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, "staking")
	govKeeper, wasmKeeper := keepers.GovKeeper, keepers.WasmKeeper

	example := InstantiateHackatomExampleContract(t, ctx, keepers)

	// set deactivate
	err := wasmKeeper.deactivateContract(ctx, example.Contract)
	require.NoError(t, err)

	src := types.ActivateContractProposal{
		Title:       "Foo",
		Description: "Bar",
		Contract:    example.Contract.String(),
	}

	em := sdk.NewEventManager()

	// when stored
	storedProposal, err := govKeeper.SubmitProposal(ctx, &src)
	require.NoError(t, err)

	// proposal execute
	handler := govKeeper.Router().GetRoute(storedProposal.ProposalRoute())
	err = handler(ctx.WithEventManager(em), storedProposal.GetContent())
	require.NoError(t, err)

	// then
	isInactive := wasmKeeper.IsInactiveContract(ctx, example.Contract)
	require.False(t, isInactive)
}
