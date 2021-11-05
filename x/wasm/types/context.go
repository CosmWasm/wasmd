package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type contextKey int

const (
	// private type creates an interface key for Context that cannot be accessed by any other package
	contextKeyTXCount         contextKey = iota
	contextKeyQueryDepthCount contextKey = iota
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

// WithQueryDepthCounter stores a query depth counter value in the context
// When a contract calls another contract the level is increased
// Used with LimitedDepthWasmQuerier
func WithQueryDepthCounter(ctx sdk.Context, counter uint32) sdk.Context {
	return ctx.WithValue(contextKeyQueryDepthCount, counter)
}

// QueryDepthCounter returns the query counter value and found bool from the context.
// Used with LimitedDepthWasmQuerier
func QueryDepthCounter(ctx sdk.Context) (uint32, bool) {
	val, ok := ctx.Value(contextKeyQueryDepthCount).(uint32)
	return val, ok
}
