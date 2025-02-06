package keeper

import (
	"encoding/binary"

	corestoretypes "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// --- Begin added definitions to fix missing references ---
// These definitions provide the missing TX counter key and context helpers.
// In your production code, you may already have these in your types package.
const txCounterKey = "tx_counter"

// TXCounterPrefix is the key under which the tx counter and height are stored.
var TXCounterPrefix = []byte(txCounterKey)

// WithTXCounter returns a new context with the given transaction counter injected.
// This example assumes your sdk.Context supports WithValue. In Cosmos SDK, you might
// instead need to use context annotations or a different mechanism.
func WithTXCounter(ctx sdk.Context, counter uint32) sdk.Context {
	// In a production implementation, you would likely use a proper context
	// mechanism. Here, for simplicity, we use the context's KV store.
	return ctx.WithValue(txCounterKey, counter)
}

// TXCounter retrieves the transaction counter from the context.
// If no value is found, it returns 0.
func TXCounter(ctx sdk.Context) uint32 {
	if counter, ok := ctx.Value(txCounterKey).(uint32); ok {
		return counter
	}
	return 0
}
// --- End added definitions ---

// CountTXDecorator is an ante handler that counts the transaction position within a block.
type CountTXDecorator struct {
	storeService corestoretypes.KVStoreService
}

// NewCountTXDecorator creates a new instance of CountTXDecorator.
func NewCountTXDecorator(s corestoretypes.KVStoreService) *CountTXDecorator {
	return &CountTXDecorator{storeService: s}
}

// AnteHandle stores a transaction counter with the current height encoded in the store.
// It lets the application handle global rollback behavior instead of keeping state in the handler itself.
// The counter is passed upstream via the sdk.Context (retrieved via TXCounter).
// Simulations do not get a transaction counter assigned.
func (a CountTXDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	store := a.storeService.OpenKVStore(ctx)
	currentHeight := ctx.BlockHeight()

	// Begin a new transaction.
	txn := store.NewTransaction()
	// Ensure the transaction is discarded if not committed.
	defer txn.Discard()

	var txCounter uint32 = 0
	// Load the existing counter (if any).
	bz, err := txn.Get(TXCounterPrefix)
	if err != nil {
		return ctx, errorsmod.Wrap(err, "read tx counter")
	}
	if bz != nil {
		lastHeight, val := decodeHeightCounter(bz)
		if currentHeight == lastHeight {
			txCounter = val
		}
	}

	// Increment the counter for the current transaction.
	newTxCounter := txCounter + 1
	if err := txn.Set(TXCounterPrefix, encodeHeightCounter(currentHeight, newTxCounter)); err != nil {
		return ctx, errorsmod.Wrap(err, "store tx counter")
	}

	// Commit the transaction.
	if err := txn.Commit(); err != nil {
		return ctx, errorsmod.Wrap(err, "commit transaction")
	}

	// Pass the updated counter value via context.
	return next(WithTXCounter(ctx, newTxCounter), tx, simulate)
}

// encodeHeightCounter encodes the block height and transaction counter into a byte slice.
// The first 8 bytes represent the block height (big-endian),
// followed by 4 bytes for the counter.
func encodeHeightCounter(height int64, counter uint32) []byte {
	counterBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(counterBytes, counter)
	return append(sdk.Uint64ToBigEndian(uint64(height)), counterBytes...)
}

// decodeHeightCounter decodes a byte slice into the block height and transaction counter.
func decodeHeightCounter(bz []byte) (int64, uint32) {
	if len(bz) < 12 {
		// Return zero values if the slice is too short.
		return 0, 0
	}
	return int64(sdk.BigEndianToUint64(bz[0:8])), binary.BigEndian.Uint32(bz[8:])
}
