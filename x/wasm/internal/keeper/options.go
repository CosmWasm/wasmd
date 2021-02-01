package keeper

import "github.com/CosmWasm/wasmd/x/wasm/internal/types"

type optsFn func(*Keeper)

func (f optsFn) apply(keeper *Keeper) {
	f(keeper)
}

// WithMessageHandler is an optional constructor parameter to replace the default wasm vm engine with the
// given one.
func WithWasmEngine(x types.WasmerEngine) Option {
	return optsFn(func(k *Keeper) {
		k.wasmer = x
	})
}

// WithMessageHandler is an optional constructor parameter to set a custom message handler.
func WithMessageHandler(n messenger) Option {
	return optsFn(func(k *Keeper) {
		k.messenger = n
	})
}
