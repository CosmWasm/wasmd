package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type contextKey int

const (
	// position counter of the TX in the block
	contextKeyTXCount contextKey = iota

	// contextKeyQueryStackSize contextKey = iota
	_

	// contextKeySubMsgAuthzPolicy = iota
	_

	// contextKeyGasRegister = iota
	_

	// contextKeyCallDepth contextKey = iota
	_

	// contextKeyTxContracts contextKey = iota
	_

	// execution mode simulation bool
	contextKeyExecModeSimulation contextKey = iota
)

// WithTXCounter stores a transaction counter value in the context
func WithTXCounter(ctx sdk.Context, counter uint32) sdk.Context {
	return ctx.WithValue(contextKeyTXCount, counter)
}

// TXCounter returns the tx counter value and found bool from the context.
// The result will be (0, false) for external queries or simulations where no counter available.
func TXCounter(ctx sdk.Context) (uint32, bool) {
	val, ok := ctx.Value(contextKeyTXCount).(uint32)
	return val, ok
}

// WithExecModeSimulation stores the simulate bool in the context
func WithExecModeSimulation(ctx sdk.Context, simulate bool) sdk.Context {
	return ctx.WithValue(contextKeyExecModeSimulation, simulate)
}

// TXCounter returns the simulation bool and found bool from the context.
func ExecModeSimulation(ctx sdk.Context) (bool, bool) {
	val, ok := ctx.Value(contextKeyExecModeSimulation).(bool)
	return val, ok
}
