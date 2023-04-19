package keeper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/Finschia/finschia-sdk/types"
)

func TestActivateContract(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)

	k := keepers.WasmKeeper
	example := InstantiateHackatomExampleContract(t, ctx, keepers)
	em := sdk.NewEventManager()

	// request no contract address -> fail
	err := k.activateContract(ctx, example.CreatorAddr)
	require.Error(t, err, fmt.Sprintf("no contract %s", example.CreatorAddr))

	// try to activate an activated contract -> fail
	err = k.activateContract(ctx.WithEventManager(em), example.Contract)
	require.Error(t, err, fmt.Sprintf("no inactivate contract %s", example.Contract))

	// add to inactive contract
	err = k.deactivateContract(ctx, example.Contract)
	require.NoError(t, err)

	// try to activate an inactivated contract -> success
	err = k.activateContract(ctx, example.Contract)
	require.NoError(t, err)
}

func TestDeactivateContract(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)

	k := keepers.WasmKeeper
	example := InstantiateHackatomExampleContract(t, ctx, keepers)
	em := sdk.NewEventManager()

	// request no contract address -> fail
	err := k.deactivateContract(ctx, example.CreatorAddr)
	require.Error(t, err, fmt.Sprintf("no contract %s", example.CreatorAddr))

	// success case
	err = k.deactivateContract(ctx, example.Contract)
	require.NoError(t, err)

	// already inactivate contract -> fail
	err = k.deactivateContract(ctx.WithEventManager(em), example.Contract)
	require.Error(t, err, fmt.Sprintf("already inactivate contract %s", example.Contract))
}

func TestIterateInactiveContracts(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)
	k := keepers.WasmKeeper

	example1 := InstantiateHackatomExampleContract(t, ctx, keepers)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	example2 := InstantiateHackatomExampleContract(t, ctx, keepers)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	err := k.deactivateContract(ctx, example1.Contract)
	require.NoError(t, err)
	err = k.deactivateContract(ctx, example2.Contract)
	require.NoError(t, err)

	var inactiveContracts []sdk.AccAddress
	k.IterateInactiveContracts(ctx, func(contractAddress sdk.AccAddress) (stop bool) {
		inactiveContracts = append(inactiveContracts, contractAddress)
		return false
	})
	assert.Equal(t, 2, len(inactiveContracts))
	expectList := []sdk.AccAddress{example1.Contract, example2.Contract}
	assert.ElementsMatch(t, expectList, inactiveContracts)
}
