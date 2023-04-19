package keeper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/Finschia/finschia-sdk/types"

	"github.com/Finschia/wasmd/x/wasmplus/types"
)

func TestQueryInactiveContracts(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)
	keeper := keepers.WasmKeeper

	example1 := InstantiateHackatomExampleContract(t, ctx, keepers)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	example2 := InstantiateHackatomExampleContract(t, ctx, keepers)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// set inactive
	err := keeper.deactivateContract(ctx, example1.Contract)
	require.NoError(t, err)
	err = keeper.deactivateContract(ctx, example2.Contract)
	require.NoError(t, err)

	q := Querier(keeper)
	rq := types.QueryInactiveContractsRequest{}
	res, err := q.InactiveContracts(sdk.WrapSDKContext(ctx), &rq)
	require.NoError(t, err)
	expect := []string{example1.Contract.String(), example2.Contract.String()}
	for _, exp := range expect {
		assert.Contains(t, res.Addresses, exp)
	}
}

func TestQueryIsInactiveContract(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)
	keeper := keepers.WasmKeeper

	example := InstantiateHackatomExampleContract(t, ctx, keepers)

	q := Querier(keeper)
	rq := types.QueryInactiveContractRequest{Address: example.Contract.String()}
	res, err := q.InactiveContract(sdk.WrapSDKContext(ctx), &rq)
	require.NoError(t, err)
	require.False(t, res.Inactivated)

	// set inactive
	err = keeper.deactivateContract(ctx, example.Contract)
	require.NoError(t, err)

	rq = types.QueryInactiveContractRequest{Address: example.Contract.String()}
	res, err = q.InactiveContract(sdk.WrapSDKContext(ctx), &rq)
	require.NoError(t, err)
	require.True(t, res.Inactivated)
}
