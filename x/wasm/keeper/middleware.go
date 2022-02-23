package keeper

import (
	"context"

	"github.com/CosmWasm/wasmd/x/wasm/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
)

var _ tx.Handler = CountTxHandler{}

type CountTxHandler struct {
	storeKey storetypes.StoreKey
	next     tx.Handler
}

// CountTxMiddleware sets CountTx in context
func CountTxMiddleware(storeKey storetypes.StoreKey) tx.Middleware {
	return func(txh tx.Handler) tx.Handler {
		return CountTxHandler{
			storeKey: storeKey,
			next:     txh,
		}
	}
}

func (ctm CountTxHandler) setCountTxHandler(ctx context.Context, req tx.Request, simulate bool) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if simulate {
		return nil
	}
	store := sdkCtx.KVStore(ctm.storeKey)
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

func (ctm CountTxHandler) CheckTx(ctx context.Context, req tx.Request, checkReq tx.RequestCheckTx) (tx.Response, tx.ResponseCheckTx, error) {
	if err := ctm.setCountTxHandler(ctx, req, false); err != nil {
		return tx.Response{}, tx.ResponseCheckTx{}, err
	}

	return ctm.next.CheckTx(ctx, req, checkReq)
}

// DeliverTx implements tx.Handler.DeliverTx.
func (ctm CountTxHandler) DeliverTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	if err := ctm.setCountTxHandler(ctx, req, false); err != nil {
		return tx.Response{}, err
	}
	return ctm.next.DeliverTx(ctx, req)
}

// SimulateTx implements tx.Handler.SimulateTx.
func (ctm CountTxHandler) SimulateTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	if err := ctm.setCountTxHandler(ctx, req, true); err != nil {
		return tx.Response{}, err
	}
	return ctm.next.SimulateTx(ctx, req)
}
