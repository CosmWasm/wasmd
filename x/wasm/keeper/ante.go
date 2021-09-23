package keeper

import (
	"github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type CountTXDecorator struct {
	storeKey sdk.StoreKey
}

func NewCountTXDecorator(storeKey sdk.StoreKey) *CountTXDecorator {
	return &CountTXDecorator{storeKey: storeKey}
}

func (a CountTXDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(types.WithTXCounter(ctx, 0), tx, simulate)
	}
	currentHeight := ctx.BlockHeight()
	store := ctx.KVStore(a.storeKey)
	var txCounter uint64 = 0
	if bz := store.Get(types.TXCounterPrefix); bz != nil {
		lastHeight, val := decodeHeightCounter(bz)
		if currentHeight == lastHeight {
			txCounter = val
		}
	}
	store.Set(types.TXCounterPrefix, encodeHeightCounter(currentHeight, txCounter+1))

	return next(types.WithTXCounter(ctx, txCounter), tx, simulate)
}

func encodeHeightCounter(height int64, counter uint64) []byte {
	return append(sdk.Uint64ToBigEndian(uint64(height)), sdk.Uint64ToBigEndian(counter)...)
}

func decodeHeightCounter(bz []byte) (int64, uint64) {
	return int64(sdk.BigEndianToUint64(bz[0:8])), sdk.BigEndianToUint64(bz[8:])
}
