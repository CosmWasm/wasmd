package wasmtesting

import (
	"context"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type MockMsgDispatcher struct {
	DispatchSubmessagesFn func(ctx context.Context, contractAddr sdk.AccAddress, ibcPort string, msgs []wasmvmtypes.SubMsg) ([]byte, error)
}

func (m MockMsgDispatcher) DispatchSubmessages(ctx context.Context, contractAddr sdk.AccAddress, ibcPort string, msgs []wasmvmtypes.SubMsg) ([]byte, error) {
	if m.DispatchSubmessagesFn == nil {
		panic("not expected to be called")
	}
	return m.DispatchSubmessagesFn(ctx, contractAddr, ibcPort, msgs)
}
