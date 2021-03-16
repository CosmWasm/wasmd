package keeper

import (
	"fmt"
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
)

type optsFn func(*Keeper)

func (f optsFn) apply(keeper *Keeper) {
	f(keeper)
}

// WithMessageHandler is an optional constructor parameter to replace the default wasmVM engine with the
// given one.
func WithWasmEngine(x types.WasmerEngine) Option {
	return optsFn(func(k *Keeper) {
		k.wasmVM = x
	})
}

// WithMessageHandler is an optional constructor parameter to set a custom handler for wasmVM messages.
func WithMessageHandler(x messenger) Option {
	return optsFn(func(k *Keeper) {
		k.messenger = x
	})
}

// WithQueryHandler is an optional constructor parameter to set custom query handler for wasmVM requests.
func WithQueryHandler(x wasmVMQueryHandler) Option {
	return optsFn(func(k *Keeper) {
		k.wasmVMQueryHandler = x
	})
}

// WithQueryPlugins is an optional constructor parameter to pass custom query plugins for wasmVM requests.
func WithQueryPlugins(x *QueryPlugins) Option {
	return optsFn(func(k *Keeper) {
		q, ok := k.wasmVMQueryHandler.(interface {
			Merge(o *QueryPlugins) QueryPlugins
		})
		if !ok {
			panic(fmt.Sprintf("Unsupported query handler type: %T", k.wasmVMQueryHandler))
		}
		k.wasmVMQueryHandler = q.Merge(x)
	})
}

// WithMessageEncoders is an optional constructor parameter to pass custom message encoder to the default wasm message handler.
func WithMessageEncoders(x *MessageEncoders) Option {
	return optsFn(func(k *Keeper) {
		q, ok := k.messenger.(*MessageHandlerChain)
		if !ok {
			panic(fmt.Sprintf("Unsupported message handler type: %T", k.messenger))
		}
		s, ok := q.handlers[0].(SDKMessageHandler)
		if !ok {
			panic(fmt.Sprintf("Unexpected message handler type: %T", q.handlers[0]))
		}
		e, ok := s.encoders.(interface {
			Merge(o *MessageEncoders) MessageEncoders
		})
		if !ok {
			panic(fmt.Sprintf("Unsupported encoder type: %T", s.encoders))
		}
		s.encoders = e.Merge(x)
		q.handlers[0] = s
	})
}

// WithCoinTransferrer is an optional constructor parameter to set a custom coin transferrer
func WithCoinTransferrer(x coinTransferrer) Option {
	return optsFn(func(k *Keeper) {
		k.bank = x
	})
}
