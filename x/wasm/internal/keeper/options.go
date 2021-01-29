package keeper

import "github.com/CosmWasm/wasmd/x/wasm/internal/types"

type optsFn func(*Keeper)

func (f optsFn) apply(keeper *Keeper) {
	f(keeper)
}

func WithWasmEngine(x types.WasmerEngine) Option {
	return optsFn(func(k *Keeper) {
		k.wasmer = x
	})
}
