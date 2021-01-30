package wasmtesting

import (
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type MockMessageHandler struct {
	DispatchFn func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msgs ...wasmvmtypes.CosmosMsg) error
}

func (m *MockMessageHandler) Dispatch(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msgs ...wasmvmtypes.CosmosMsg) error {
	return m.DispatchFn(ctx, contractAddr, contractIBCPortID, msgs...)
}

func NewCapturingMessageHandler() (*MockMessageHandler, *[]wasmvmtypes.CosmosMsg) {
	var messages []wasmvmtypes.CosmosMsg
	return &MockMessageHandler{
		DispatchFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msgs ...wasmvmtypes.CosmosMsg) error {
			messages = append(messages, msgs...)
			return nil
		},
	}, &messages
}
