package app

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/xwasmvm/custom"
)

// AllCapabilities returns all capabilities available with the current wasmvm
// See https://github.com/CosmWasm/cosmwasm/blob/main/docs/CAPABILITIES-BUILT-IN.md
// This functionality is going to be moved upstream: https://github.com/CosmWasm/wasmvm/issues/425
func AllCapabilities() []string {
	return []string{
		"iterator",
		"staking",
		"stargate",
		"cosmwasm_1_1",
		"cosmwasm_1_2",
	}
}

// CustomMsgDecorator is hack to add a custom message that is not part of wasmvm, yet.
// used for rapid prototyping only
func CustomMsgDecorator(k custom.Adr8Keeper) func(nested wasmkeeper.Messenger) wasmkeeper.Messenger {
	return func(nested wasmkeeper.Messenger) wasmkeeper.Messenger {
		return wasmkeeper.NewMessageHandlerChain(
			custom.XMessageHandler(k),
			nested,
		)
	}
}
