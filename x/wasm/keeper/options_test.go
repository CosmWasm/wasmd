package keeper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authkeeper "github.com/line/lbm-sdk/x/auth/keeper"
	bankpluskeeper "github.com/line/lbm-sdk/x/bankplus/keeper"
	distributionkeeper "github.com/line/lbm-sdk/x/distribution/keeper"
	paramtypes "github.com/line/lbm-sdk/x/params/types"
	stakingkeeper "github.com/line/lbm-sdk/x/staking/keeper"

	"github.com/line/wasmd/x/wasm/keeper/wasmtesting"
	"github.com/line/wasmd/x/wasm/types"
)

func TestConstructorOptions(t *testing.T) {
	specs := map[string]struct {
		srcOpt Option
		verify func(*testing.T, Keeper)
	}{
		"wasm engine": {
			srcOpt: WithWasmEngine(&wasmtesting.MockWasmer{}),
			verify: func(t *testing.T, k Keeper) {
				assert.IsType(t, &wasmtesting.MockWasmer{}, k.wasmVM)
			},
		},
		"message handler": {
			srcOpt: WithMessageHandler(&wasmtesting.MockMessageHandler{}),
			verify: func(t *testing.T, k Keeper) {
				assert.IsType(t, &wasmtesting.MockMessageHandler{}, k.messenger)
			},
		},
		"query plugins": {
			srcOpt: WithQueryHandler(&wasmtesting.MockQueryHandler{}),
			verify: func(t *testing.T, k Keeper) {
				assert.IsType(t, &wasmtesting.MockQueryHandler{}, k.wasmVMQueryHandler)
			},
		},
		"message handler decorator": {
			srcOpt: WithMessageHandlerDecorator(func(old Messenger) Messenger {
				require.IsType(t, &MessageHandlerChain{}, old)
				return &wasmtesting.MockMessageHandler{}
			}),
			verify: func(t *testing.T, k Keeper) {
				assert.IsType(t, &wasmtesting.MockMessageHandler{}, k.messenger)
			},
		},
		"query plugins decorator": {
			srcOpt: WithQueryHandlerDecorator(func(old WasmVMQueryHandler) WasmVMQueryHandler {
				require.IsType(t, QueryPlugins{}, old)
				return &wasmtesting.MockQueryHandler{}
			}),
			verify: func(t *testing.T, k Keeper) {
				assert.IsType(t, &wasmtesting.MockQueryHandler{}, k.wasmVMQueryHandler)
			},
		},
		"coin transferrer": {
			srcOpt: WithCoinTransferrer(&wasmtesting.MockCoinTransferrer{}),
			verify: func(t *testing.T, k Keeper) {
				assert.IsType(t, &wasmtesting.MockCoinTransferrer{}, k.bank)
			},
		},
		"max recursion query limit": {
			srcOpt: WithMaxQueryStackSize(1),
			verify: func(t *testing.T, k Keeper) {
				assert.IsType(t, uint32(1), k.maxQueryStackSize)
			},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			k := NewKeeper(nil, nil, paramtypes.NewSubspace(nil, nil, nil, nil, ""), authkeeper.AccountKeeper{}, bankpluskeeper.BaseKeeper{}, stakingkeeper.Keeper{}, distributionkeeper.Keeper{}, nil, nil, nil, nil, nil, nil, "tempDir", types.DefaultWasmConfig(), SupportedFeatures, nil, nil, spec.srcOpt)
			spec.verify(t, k)
		})
	}

}
