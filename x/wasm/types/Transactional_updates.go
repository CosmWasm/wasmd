package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	corestoretypes "cosmossdk.io/core/store"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// MockKVStoreService and related implementations should be defined to simulate the store behavior.

func TestCountTXDecorator_SameBlock(t *testing.T) {
	// Create a mock store service and context
	storeService := corestoretypes.NewMockKVStoreService() // Assuming you have a mock implementation
	ctx := sdk.Context{}.WithBlockHeight(100)
	// Create the decorator
	decorator := NewCountTXDecorator(storeService)

	// Define a dummy next handler that simply returns the context and nil error.
	nextHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	}

	// Execute the handler twice in the same block.
	ctx, err := decorator.AnteHandle(ctx, nil, false, nextHandler)
	require.NoError(t, err)
	counter1 := types.TXCounter(ctx) // Implement retrieval as needed

	ctx, err = decorator.AnteHandle(ctx, nil, false, nextHandler)
	require.NoError(t, err)
	counter2 := types.TXCounter(ctx)

	// Expect counter2 to be counter1 + 1.
	require.Equal(t, counter1+1, counter2)
}

func TestCountTXDecorator_NewBlock(t *testing.T) {
	// Create a mock store service and context for block height 100.
	storeService := corestoretypes.NewMockKVStoreService() // Replace with your mock implementation
	ctx := sdk.Context{}.WithBlockHeight(100)
	decorator := NewCountTXDecorator(storeService)

	nextHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	}

	// Execute the handler.
	ctx, err := decorator.AnteHandle(ctx, nil, false, nextHandler)
	require.NoError(t, err)
	counter := types.TXCounter(ctx)
	require.Equal(t, uint32(1), counter)

	// Simulate a new block.
	ctx = ctx.WithBlockHeight(101)
	ctx, err = decorator.AnteHandle(ctx, nil, false, nextHandler)
	require.NoError(t, err)
	newCounter := types.TXCounter(ctx)
	// Since it's a new block, counter should reset to 1.
	require.Equal(t, uint32(1), newCounter)
}

