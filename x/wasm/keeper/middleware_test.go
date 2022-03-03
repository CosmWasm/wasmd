package keeper_test

import (
	"context"
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/x/auth/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"
)

func TestCountTxHandle(t *testing.T) {
	keyWasm := sdk.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyWasm, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())
	const myCurrentBlockHeight = 100

	specs := map[string]struct {
		setupDB        func(t *testing.T, ctx sdk.Context)
		simulate       bool
		nextAssertAnte func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error)
		expErr         bool
	}{
		"no initial counter set": {
			setupDB: func(t *testing.T, ctx sdk.Context) {},
			nextAssertAnte: func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
				gotCounter, ok := types.TXCounter(ctx)
				require.True(t, ok)
				assert.Equal(t, uint32(0), gotCounter)
				// and stored +1
				bz := ctx.MultiStore().GetKVStore(keyWasm).Get(types.TXCounterPrefix)
				assert.Equal(t, []byte{0, 0, 0, 0, 0, 0, 0, myCurrentBlockHeight, 0, 0, 0, 1}, bz)
				return ctx, nil
			},
		},
		"persistent counter incremented - big endian": {
			setupDB: func(t *testing.T, ctx sdk.Context) {
				bz := []byte{0, 0, 0, 0, 0, 0, 0, myCurrentBlockHeight, 1, 0, 0, 2}
				ctx.MultiStore().GetKVStore(keyWasm).Set(types.TXCounterPrefix, bz)
			},
			nextAssertAnte: func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
				gotCounter, ok := types.TXCounter(ctx)
				require.True(t, ok)
				assert.Equal(t, uint32(1<<24+2), gotCounter)
				// and stored +1
				bz := ctx.MultiStore().GetKVStore(keyWasm).Get(types.TXCounterPrefix)
				assert.Equal(t, []byte{0, 0, 0, 0, 0, 0, 0, myCurrentBlockHeight, 1, 0, 0, 3}, bz)
				return ctx, nil
			},
		},
		"old height counter replaced": {
			setupDB: func(t *testing.T, ctx sdk.Context) {
				previousHeight := byte(myCurrentBlockHeight - 1)
				bz := []byte{0, 0, 0, 0, 0, 0, 0, previousHeight, 0, 0, 0, 1}
				ctx.MultiStore().GetKVStore(keyWasm).Set(types.TXCounterPrefix, bz)
			},
			nextAssertAnte: func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
				gotCounter, ok := types.TXCounter(ctx)
				require.True(t, ok)
				assert.Equal(t, uint32(0), gotCounter)
				// and stored +1
				bz := ctx.MultiStore().GetKVStore(keyWasm).Get(types.TXCounterPrefix)
				assert.Equal(t, []byte{0, 0, 0, 0, 0, 0, 0, myCurrentBlockHeight, 0, 0, 0, 1}, bz)
				return ctx, nil
			},
		},
		"simulation not persisted": {
			setupDB: func(t *testing.T, ctx sdk.Context) {
			},
			simulate: true,
			nextAssertAnte: func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
				_, ok := types.TXCounter(ctx)
				assert.False(t, ok)
				require.True(t, simulate)
				// and not stored
				assert.False(t, ctx.MultiStore().GetKVStore(keyWasm).Has(types.TXCounterPrefix))
				return ctx, nil
			},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			ctx := sdk.NewContext(ms.CacheMultiStore(), tmproto.Header{
				Height: myCurrentBlockHeight,
				Time:   time.Date(2021, time.September, 27, 12, 0, 0, 0, time.UTC),
			}, false, log.NewNopLogger())

			spec.setupDB(t, ctx)
			var anyTx sdk.Tx
			txHandler := middleware.ComposeMiddlewares(noopTxHandler, keeper.CountTxMiddleware(keyWasm))

			// test DeliverTx
			_, _, gotErr := txHandler.CheckTx(ctx, txtypes.Request{Tx: anyTx}, txtypes.RequestCheckTx{})
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
		})
	}
}

func TestLimitSimulationGasMiddleware(t *testing.T) {
	var (
		hundred sdk.Gas = 100
		zero    sdk.Gas = 0
	)
	specs := map[string]struct {
		customLimit *sdk.Gas
		consumeGas  sdk.Gas
		maxBlockGas int64
		simulation  bool
		expErr      interface{}
	}{
		"custom limit set": {
			customLimit: &hundred,
			consumeGas:  hundred + 1,
			maxBlockGas: -1,
			simulation:  true,
			expErr:      sdk.ErrorOutOfGas{Descriptor: "testing"},
		},
		"block limit set": {
			maxBlockGas: 100,
			consumeGas:  hundred + 1,
			simulation:  true,
			expErr:      sdk.ErrorOutOfGas{Descriptor: "testing"},
		},
		"no limits set": {
			maxBlockGas: -1,
			consumeGas:  hundred + 1,
			simulation:  true,
		},
		"both limits set, custom applies": {
			customLimit: &hundred,
			consumeGas:  hundred - 1,
			maxBlockGas: 10,
			simulation:  true,
		},
		"not a simulation": {
			customLimit: &hundred,
			consumeGas:  hundred + 1,
			simulation:  false,
		},
		"zero custom limit": {
			customLimit: &zero,
			simulation:  true,
			expErr:      "gas limit must not be zero",
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			ctx := sdk.Context{}.
				WithGasMeter(sdk.NewInfiniteGasMeter()).
				WithConsensusParams(&tmproto.ConsensusParams{
					Block: &tmproto.BlockParams{MaxGas: spec.maxBlockGas}})

			//setting TxHandler
			var anyTx sdk.Tx

			txHandler := middleware.ComposeMiddlewares(
				noopTxHandler,
				keeper.LimitSimulationGasMiddleware(spec.customLimit),
				cosumeGasTxMiddleware(spec.consumeGas),
			)

			if spec.expErr != nil {
				require.PanicsWithValue(t, spec.expErr, func() {
					txHandler.SimulateTx(ctx, txtypes.Request{Tx: anyTx})
				})
				return
			}
		})
	}
}

// customTxHandler is a test middleware that will run a custom function.
type customTxHandler struct {
	fn func(context.Context, tx.Request) (tx.Response, error)
}

var _ tx.Handler = customTxHandler{}

func (h customTxHandler) DeliverTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	return h.fn(ctx, req)
}
func (h customTxHandler) CheckTx(ctx context.Context, req tx.Request, _ tx.RequestCheckTx) (tx.Response, tx.ResponseCheckTx, error) {
	res, err := h.fn(ctx, req)
	return res, tx.ResponseCheckTx{}, err
}
func (h customTxHandler) SimulateTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	return h.fn(ctx, req)
}

// noopTxHandler is a test middleware that returns an empty response.
var noopTxHandler = customTxHandler{func(_ context.Context, _ tx.Request) (tx.Response, error) {
	return tx.Response{}, nil
}}

// customTxHandler is a test middleware that will run a custom function.
type cosumeGasTxHandler struct {
	gasToConsume sdk.Gas
	next         tx.Handler
}

var _ tx.Handler = cosumeGasTxHandler{}

func (h cosumeGasTxHandler) DeliverTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	return h.next.DeliverTx(ctx, req)
}
func (h cosumeGasTxHandler) CheckTx(ctx context.Context, req tx.Request, checkReq tx.RequestCheckTx) (tx.Response, tx.ResponseCheckTx, error) {
	return h.next.CheckTx(ctx, req, checkReq)
}
func (h cosumeGasTxHandler) SimulateTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.GasMeter().ConsumeGas(h.gasToConsume, "testing")
	return h.next.SimulateTx(ctx, req)
}

func cosumeGasTxMiddleware(gas sdk.Gas) tx.Middleware {
	return func(txh tx.Handler) tx.Handler {
		return cosumeGasTxHandler{
			gasToConsume: gas,
			next:         txh,
		}
	}
}
