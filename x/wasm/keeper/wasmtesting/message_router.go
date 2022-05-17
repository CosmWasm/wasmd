package wasmtesting

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/middleware"
)

// MockMessageRouter mock for testing
type MockMessageRouter struct {
	HandlerFn func(msg sdk.Msg) middleware.MsgServiceHandler
}

// Handler is the entry point
func (m MockMessageRouter) Handler(msg sdk.Msg) middleware.MsgServiceHandler {
	if m.HandlerFn == nil {
		panic("not expected to be called")
	}
	return m.HandlerFn(msg)
}

// MessageRouterFunc convenient type to match the keeper.MessageRouter interface
type MessageRouterFunc func(msg sdk.Msg) middleware.MsgServiceHandler

// Handler is the entry point
func (m MessageRouterFunc) Handler(msg sdk.Msg) middleware.MsgServiceHandler {
	return m(msg)
}
