package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type contextKey int

const (
	// private type creates an interface key for Context that cannot be accessed by any other package
	contextKeyTXCount contextKey = iota
)

func WithTXCounter(ctx sdk.Context, counter uint64) sdk.Context {
	return ctx.WithValue(contextKeyTXCount, counter)
}

// TXCounter return tx counter value
// Will default to `0` when not set for example in external queries or simulations
func TXCounter(ctx sdk.Context) uint64 {
	val, ok := ctx.Value(contextKeyTXCount).(uint64)
	if !ok {
		return 0
	}
	return val
}
