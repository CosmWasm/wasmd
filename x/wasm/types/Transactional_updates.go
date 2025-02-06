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

	// Begin a transaction
	txn := store.NewTransaction()
	defer txn.Discard() // Ensure the transaction is discarded if not committed

	var txCounter uint32 // start with 0
	// load counter when exists
	bz, err := txn.Get(types.TXCounterPrefix)
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
	err = txn.Set(types.TXCounterPrefix, encodeHeightCounter(currentHeight, newTxCounter))
	if err != nil {
		return ctx, errorsmod.Wrap(err, "store tx counter")
	}

	// Commit the transaction
	err = txn.Commit()
	if err != nil {
		return ctx, errorsmod.Wrap(err, "commit transaction")
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
