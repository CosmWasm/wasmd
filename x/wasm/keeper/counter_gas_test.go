package keeper_test

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log/v2"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/store/v2"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func counterBytes(height uint64, counter uint32) []byte {
	bz := make([]byte, 12)
	binary.BigEndian.PutUint64(bz, height)
	binary.BigEndian.PutUint32(bz[8:], counter)
	return bz
}

func counterGasContext(t *testing.T, config storetypes.GasConfig, initial []byte) (sdk.Context, *storetypes.KVStoreKey) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ms := store.NewCommitMultiStore(dbm.NewMemDB(), log.NewTestLogger(t))
	ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, nil)
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms.CacheMultiStore(), cmtproto.Header{Height: 100}, false, log.NewNopLogger()).
		WithGasMeter(storetypes.NewInfiniteGasMeter()).WithKVGasConfig(config)
	if initial != nil {
		ctx.MultiStore().GetKVStore(key).Set(types.TXCounterPrefix, initial)
	}
	return ctx, key
}

func TestCountTxSimulationGas(t *testing.T) {
	custom := storetypes.KVGasConfig()
	custom.ReadCostFlat, custom.ReadCostPerByte = 127, 7
	custom.WriteCostFlat, custom.WriteCostPerByte = 409, 11
	for _, config := range []struct {
		name string
		gas  storetypes.GasConfig
	}{
		{"default", storetypes.KVGasConfig()},
		{"custom", custom},
	} {
		for _, tc := range []struct {
			name    string
			initial []byte
			index   uint32
		}{
			{"absent", nil, 0},
			{"current height", counterBytes(100, 5), 5},
			{"previous height", counterBytes(99, 5), 0},
		} {
			t.Run(config.name+"/"+tc.name, func(t *testing.T) {
				ctx, key := counterGasContext(t, config.gas, tc.initial)
				ante := keeper.NewCountTXDecorator(runtime.NewKVStoreService(key))
				var counterGas uint64
				for call := 0; call < 2; call++ {
					before := ctx.GasMeter().GasConsumed()
					result, err := ante.AnteHandle(ctx, nil, true, func(nextCtx sdk.Context, _ sdk.Tx, simulate bool) (sdk.Context, error) {
						require.True(t, simulate)
						require.Equal(t, ctx.MultiStore(), nextCtx.MultiStore())
						require.Same(t, ctx.GasMeter(), nextCtx.GasMeter())
						_, assigned := types.TXCounter(nextCtx)
						require.False(t, assigned)
						require.Equal(t, tc.initial, nextCtx.MultiStore().GetKVStore(key).Get(types.TXCounterPrefix))
						// Downstream gas must accumulate on the same meter.
						nextCtx.GasMeter().ConsumeGas(7, "next decorator")
						return nextCtx, nil
					})
					require.NoError(t, err)
					used := ctx.GasMeter().GasConsumed() - before - 7
					require.Positive(t, used)
					if call == 0 {
						counterGas = used
					} else {
						require.Equal(t, counterGas, used)
					}
					require.Equal(t, ctx.MultiStore(), result.MultiStore())
					require.Equal(t, tc.initial, ctx.MultiStore().GetKVStore(key).Get(types.TXCounterPrefix))
				}
				// Compare to the actual execution path from identical state and config.
				execution, executionKey := counterGasContext(t, config.gas, tc.initial)
				_, err := keeper.NewCountTXDecorator(runtime.NewKVStoreService(executionKey)).AnteHandle(
					execution, nil, false, func(nextCtx sdk.Context, _ sdk.Tx, simulate bool) (sdk.Context, error) {
						require.False(t, simulate)
						index, assigned := types.TXCounter(nextCtx)
						require.True(t, assigned)
						require.Equal(t, tc.index, index)
						require.Equal(t, counterBytes(100, tc.index+1), nextCtx.MultiStore().GetKVStore(executionKey).Get(types.TXCounterPrefix))
						return nextCtx, nil
					})
				require.NoError(t, err)
				require.Equal(t, execution.GasMeter().GasConsumed(), counterGas)
				t.Logf("simulation counter gas=%d execution counter gas=%d", counterGas, execution.GasMeter().GasConsumed())
			})
		}
	}
}

func TestCountTxSimulationPreservesContext(t *testing.T) {
	ctx, key := counterGasContext(t, storetypes.KVGasConfig(), counterBytes(100, 5))
	ctx = types.WithTXCounter(ctx, 23)
	nextErr := errors.New("next decorator failed")
	_, err := keeper.NewCountTXDecorator(runtime.NewKVStoreService(key)).AnteHandle(ctx, nil, true,
		func(nextCtx sdk.Context, _ sdk.Tx, simulate bool) (sdk.Context, error) {
			require.True(t, simulate)
			index, assigned := types.TXCounter(nextCtx)
			require.True(t, assigned)
			require.EqualValues(t, 23, index, "preserve an incoming index without assigning a new one")
			return nextCtx, nextErr
		})
	require.ErrorIs(t, err, nextErr)
	require.Equal(t, counterBytes(100, 5), ctx.MultiStore().GetKVStore(key).Get(types.TXCounterPrefix))
	require.Positive(t, ctx.GasMeter().GasConsumed())
}

func TestCountTxSimulationGasLimit(t *testing.T) {
	for _, initial := range [][]byte{nil, counterBytes(100, 5)} {
		ctx, key := counterGasContext(t, storetypes.KVGasConfig(), initial)
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(2000))
		require.Panics(t, func() {
			_, _ = keeper.NewCountTXDecorator(runtime.NewKVStoreService(key)).AnteHandle(ctx, nil, true,
				func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
					t.Fatal("next decorator must not run after exhausting counter gas")
					return ctx, nil
				})
		})
		require.True(t, ctx.GasMeter().IsOutOfGas())
		require.Equal(t, initial, ctx.MultiStore().GetKVStore(key).Get(types.TXCounterPrefix))
	}
}

type failingCounterStoreService struct {
	corestore.KVStoreService
	readErr  error
	writeErr error
}

func (s failingCounterStoreService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return failingCounterStore{s.KVStoreService.OpenKVStore(ctx), s.readErr, s.writeErr}
}

type failingCounterStore struct {
	corestore.KVStore
	readErr  error
	writeErr error
}

func (s failingCounterStore) Get(key []byte) ([]byte, error) {
	if s.readErr != nil {
		return nil, s.readErr
	}
	return s.KVStore.Get(key)
}

func (s failingCounterStore) Set(key, value []byte) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	return s.KVStore.Set(key, value)
}

func TestCountTxSimulationStoreErrors(t *testing.T) {
	storeErr := errors.New("counter store failed")
	for _, tc := range []struct {
		name string
		read bool
	}{
		{"read tx counter", true},
		{"store tx counter", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, key := counterGasContext(t, storetypes.KVGasConfig(), counterBytes(100, 5))
			service := failingCounterStoreService{KVStoreService: runtime.NewKVStoreService(key)}
			if tc.read {
				service.readErr = storeErr
			} else {
				service.writeErr = storeErr
			}
			result, err := keeper.NewCountTXDecorator(service).AnteHandle(ctx, nil, true,
				func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
					t.Fatal("next decorator must not run after a store error")
					return ctx, nil
				})
			require.ErrorIs(t, err, storeErr)
			require.ErrorContains(t, err, tc.name)
			require.Equal(t, ctx.MultiStore(), result.MultiStore())
			require.Equal(t, counterBytes(100, 5), ctx.MultiStore().GetKVStore(key).Get(types.TXCounterPrefix))
		})
	}
}
