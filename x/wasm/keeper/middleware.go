package keeper

import (
	"context"
	"encoding/binary"

	"github.com/CosmWasm/wasmd/x/wasm/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
)

var _ tx.Handler = countTxHandler{}

type countTxHandler struct {
	storeKey storetypes.StoreKey
	next     tx.Handler
}

// CountTxMiddleware sets CountTx in context
func CountTxMiddleware(storeKey storetypes.StoreKey) tx.Middleware {
	return func(txh tx.Handler) tx.Handler {
		return countTxHandler{
			storeKey: storeKey,
			next:     txh,
		}
	}
}

func (ctm countTxHandler) setCountTxHandler(ctx context.Context, req tx.Request, simulate bool) error {
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

// CheckTx implements tx.Handler.CheckTx.
func (ctm countTxHandler) CheckTx(ctx context.Context, req tx.Request, checkReq tx.RequestCheckTx) (tx.Response, tx.ResponseCheckTx, error) {
	if err := ctm.setCountTxHandler(ctx, req, false); err != nil {
		return tx.Response{}, tx.ResponseCheckTx{}, err
	}

	return ctm.next.CheckTx(ctx, req, checkReq)
}

// DeliverTx implements tx.Handler.DeliverTx.
func (ctm countTxHandler) DeliverTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	if err := ctm.setCountTxHandler(ctx, req, false); err != nil {
		return tx.Response{}, err
	}
	return ctm.next.DeliverTx(ctx, req)
}

// SimulateTx implements tx.Handler.SimulateTx.
func (ctm countTxHandler) SimulateTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	if err := ctm.setCountTxHandler(ctx, req, true); err != nil {
		return tx.Response{}, err
	}
	return ctm.next.SimulateTx(ctx, req)
}

var _ tx.Handler = LimitSimulationGasHandler{}

// LimitSimulationGasHandler to limit gas in simulation calls
type LimitSimulationGasHandler struct {
	gasLimit *sdk.Gas
	next     tx.Handler
}

// LimitSimulationGasMiddleware to limit gas in simulation calls
func LimitSimulationGasMiddleware(gas *sdk.Gas) tx.Middleware {
	return func(txh tx.Handler) tx.Handler {
		return LimitSimulationGasHandler{
			gasLimit: gas,
			next:     txh,
		}
	}
}

func (d LimitSimulationGasHandler) setLimitSimulationGas(ctx context.Context, req tx.Request, simulate bool) (context.Context, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if d.gasLimit != nil && *d.gasLimit == 0 {
		panic("gas limit must not be zero")
	}

	if !simulate {
		return ctx, nil
	}

	// apply custom node gas limit
	if d.gasLimit != nil {
		return sdkCtx.WithGasMeter(storetypes.NewGasMeter(*d.gasLimit)), nil
	}

	// default to max block gas when set, to be on the safe side
	if maxGas := sdkCtx.ConsensusParams().GetBlock().MaxGas; maxGas > 0 {
		return sdkCtx.WithGasMeter(sdk.NewGasMeter(sdk.Gas(maxGas))), nil
	}

	return ctx, nil
}

// CheckTx implements tx.Handler.CheckTx.
func (d LimitSimulationGasHandler) CheckTx(ctx context.Context, req tx.Request, checkReq tx.RequestCheckTx) (tx.Response, tx.ResponseCheckTx, error) {
	ctx, err := d.setLimitSimulationGas(ctx, req, false)
	if err != nil {
		return tx.Response{}, tx.ResponseCheckTx{}, err
	}

	return d.next.CheckTx(ctx, req, checkReq)
}

// DeliverTx implements tx.Handler.DeliverTx.
func (d LimitSimulationGasHandler) DeliverTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	ctx, err := d.setLimitSimulationGas(ctx, req, false)
	if err != nil {
		return tx.Response{}, err
	}
	return d.next.DeliverTx(ctx, req)
}

// SimulateTx implements tx.Handler.SimulateTx.
func (d LimitSimulationGasHandler) SimulateTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	ctx, err := d.setLimitSimulationGas(ctx, req, true)
	if err != nil {
		return tx.Response{}, err
	}
	return d.next.SimulateTx(ctx, req)
}

func encodeHeightCounter(height int64, counter uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, counter)
	return append(sdk.Uint64ToBigEndian(uint64(height)), b...)
}

func decodeHeightCounter(bz []byte) (int64, uint32) {
	return int64(sdk.BigEndianToUint64(bz[0:8])), binary.BigEndian.Uint32(bz[8:])
}
