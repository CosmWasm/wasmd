package wasmtesting

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type MockCoinTransferrer struct {
	TransferCoinsFn func(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
}

func (m *MockCoinTransferrer) TransferCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	if m.TransferCoinsFn == nil {
		panic("not expected to be called")
	}
	return m.TransferCoinsFn(ctx, fromAddr, toAddr, amt)
}

type AccountPrunerMock struct {
	CleanupExistingAccountFn func(ctx context.Context, existingAccount sdk.AccountI) (handled bool, err error)
}

func (m AccountPrunerMock) CleanupExistingAccount(ctx context.Context, existingAccount sdk.AccountI) (handled bool, err error) {
	if m.CleanupExistingAccountFn == nil {
		panic("not expected to be called")
	}
	return m.CleanupExistingAccountFn(ctx, existingAccount)
}
