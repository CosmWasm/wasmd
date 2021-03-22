package keeper

import (
	"fmt"
	"github.com/CosmWasm/wasmd/x/wasm/types"
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
// This option should not be combined with Option `WithMessageEncoders`.
func WithMessageHandler(x Messenger) Option {
	return optsFn(func(k *Keeper) {
		k.messenger = x
	})
}

// WithQueryHandler is an optional constructor parameter to set custom query handler for wasmVM requests.
// This option should not be combined with Option `WithQueryPlugins`.
func WithQueryHandler(x WASMVMQueryHandler) Option {
	return optsFn(func(k *Keeper) {
		k.wasmVMQueryHandler = x
	})
}

// WithQueryPlugins is an optional constructor parameter to pass custom query plugins for wasmVM requests.
// This option expects the default `QueryHandler` set an should not be combined with Option `WithQueryHandler`.
func WithQueryPlugins(x *QueryPlugins) Option {
	return optsFn(func(k *Keeper) {
		q, ok := k.wasmVMQueryHandler.(QueryPlugins)
		if !ok {
			panic(fmt.Sprintf("Unsupported query handler type: %T", k.wasmVMQueryHandler))
		}
		k.wasmVMQueryHandler = q.Merge(x)
	})
}

// WithMessageEncoders is an optional constructor parameter to pass custom message encoder to the default wasm message handler.
// This option expects the `DefaultMessageHandler` set an should not be combined with Option `WithMessageHandler`.
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
		e, ok := s.encoders.(MessageEncoders)
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
