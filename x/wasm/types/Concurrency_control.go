import (
	"encoding/binary"
	"sync"

	corestoretypes "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// CountTXDecorator ante handler to count the tx position in a block.
type CountTXDecorator struct {
	storeService corestoretypes.KVStoreService
	mu           sync.Mutex // Mutex for concurrency control
}

// NewCountTXDecorator constructor
func NewCountTXDecorator(s corestoretypes.KVStoreService) *CountTXDecorator {
	return &CountTXDecorator{storeService: s}
}

// AnteHandle handler stores a tx counter with current height encoded in the store to let the app handle
// global rollback behavior instead of keeping state in the handler itself.
// The ante handler passes the counter value via sdk.Context upstream. See `types.TXCounter(ctx)` to read the value.
// Simulations don't get a tx counter value assigned.
func (a CountTXDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}
	store := a.storeService.OpenKVStore(ctx)
	currentHeight := ctx.BlockHeight()

	var txCounter uint32 // start with 0

	a.mu.Lock()         // Lock for concurrency control
	defer a.mu.Unlock() // Unlock after the function completes

	// load counter when exists
	bz, err := store.Get(types.TXCounterPrefix)
	if err != nil {
		return ctx, errorsmod.Wrap(err, "read tx counter")
	}
	if bz != nil {
		lastHeight, val := decodeHeightCounter(bz)
		if currentHeight == lastHeight {
			// then use stored counter
			txCounter = val
		} // else use `0` from above to start with
	}
	// store next counter value for current height
	newTxCounter := txCounter + 1
	err = store.Set(types.TXCounterPrefix, encodeHeightCounter(currentHeight, newTxCounter))
	if err != nil {
		return ctx, errorsmod.Wrap(err, "store tx counter")
	}
	return next(types.WithTXCounter(ctx, newTxCounter), tx, simulate)
}

func encodeHeightCounter(height int64, counter uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, counter)
	return append(sdk.Uint64ToBigEndian(uint64(height)), b...)
}

func decodeHeightCounter(bz []byte) (int64, uint32) {
	return int64(sdk.BigEndianToUint64(bz[0:8])), binary.BigEndian.Uint32(bz[8:])
}

distrbuted_counters.go

package keeper

import (
	"encoding/binary"

	corestoretypes "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// CountTXDecorator ante handler to count the tx position in a block.
type CountTXDecorator struct {
	storeService corestoretypes.KVStoreService
	counter      *DistributedCounter // Distributed counter
}

// NewCountTXDecorator constructor
func NewCountTXDecorator(s corestoretypes.KVStoreService, counter *DistributedCounter) *CountTXDecorator {
	return &CountTXDecorator{storeService: s, counter: counter}
}

// AnteHandle handler stores a tx counter with current height encoded in the store to let the app handle
// global rollback behavior instead of keeping state in the handler itself.
// The ante handler passes the counter value via sdk.Context upstream. See `types.TXCounter(ctx)` to read the value.
// Simulations don't get a tx counter value assigned.
func (a CountTXDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}
	store := a.storeService.OpenKVStore(ctx)
	currentHeight := ctx.BlockHeight()

	txCounter, err := a.counter.Increment(currentHeight)
	if err != nil {
		return ctx, errorsmod.Wrap(err, "increment tx counter")
	}

	// store next counter value for current height
	err = store.Set(types.TXCounterPrefix, encodeHeightCounter(currentHeight, txCounter))
	if err != nil {
		return ctx, errorsmod.Wrap(err, "store tx counter")
	}
	return next(types.WithTXCounter(ctx, txCounter), tx, simulate)
}

func encodeHeightCounter(height int64, counter uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, counter)
	return append(sdk.Uint64ToBigEndian(uint64(height)), b...)
}

func decodeHeightCounter(bz []byte) (int64, uint32) {
	return int64(sdk.BigEndianToUint64(bz[0:8])), binary.BigEndian.Uint32(bz[8:])
}

// DistributedCounter is a placeholder for a distributed counter implementation.
type DistributedCounter struct {
	// Implementation details for the distributed counter
}

func (dc *DistributedCounter) Increment(height int64) (uint32, error) {
	// Increment logic for the distributed counter
	return 0, nil
}
