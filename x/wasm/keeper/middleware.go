package keeper

import (
	"context"

	"github.com/CosmWasm/wasmd/x/wasm/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
)

var _ tx.Handler = setCountTxHandler{}

type setCountTxHandler struct {
	storeKey storetypes.StoreKey
	next     tx.Handler
}

// SetCountTxMiddleware sets CountTx in context
func SetCountTxMiddleware(storeKey storetypes.StoreKey) tx.Middleware {
	return func(txh tx.Handler) tx.Handler {
		return setCountTxHandler{
			storeKey: storeKey,
			next:     txh,
		}
	}
}

func (sctm setCountTxHandler) setCountTxHandler(ctx context.Context, req tx.Request, simulate bool) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if simulate {
		return nil
	}
	store := sdkCtx.KVStore(sctm.storeKey)
	currentHeight := sdkCtx.BlockHeight()

	var txCounter uint32 // start with 0
	// load counter when exists
	if bz := store.Get(types.TXCounterPrefix); bz != nil {
		lastHeight, val := decodeHeightCounter(bz)
		if currentHeight == lastHeight {
			// then use stored counter
			txCounter = val
		} // else use `0` from above to start with
	}
	// store next counter value for current height
	store.Set(types.TXCounterPrefix, encodeHeightCounter(currentHeight, txCounter+1))
	return nil
}

func (sctm setCountTxHandler) CheckTx(ctx context.Context, req tx.Request, checkReq tx.RequestCheckTx) (tx.Response, tx.ResponseCheckTx, error) {
	if err := sctm.setCountTxHandler(ctx, req, false); err != nil {
		return tx.Response{}, tx.ResponseCheckTx{}, err
	}

	return sctm.next.CheckTx(ctx, req, checkReq)
}

// DeliverTx implements tx.Handler.DeliverTx.
func (sctm setCountTxHandler) DeliverTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	if err := sctm.setCountTxHandler(ctx, req, false); err != nil {
		return tx.Response{}, err
	}
	return sctm.next.DeliverTx(ctx, req)
}

// SimulateTx implements tx.Handler.SimulateTx.
func (sctm setCountTxHandler) SimulateTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	if err := sctm.setCountTxHandler(ctx, req, true); err != nil {
		return tx.Response{}, err
	}
	return sctm.next.SimulateTx(ctx, req)
}
