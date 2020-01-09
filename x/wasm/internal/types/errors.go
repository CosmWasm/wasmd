package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Codes for wasm contract errors
var (
	DefaultCodespace = ModuleName

	// ErrCreateFailed error for wasm code that has already been uploaded or failed
	ErrCreateFailed = sdkerrors.Register(DefaultCodespace, 1, "create wasm contract failed")

	// ErrAccountExists error for a contract account that already exists
	ErrAccountExists = sdkerrors.Register(DefaultCodespace, 2, "contract account already exists")

	// ErrInstantiateFailed error for rust instantiate contract failure
	ErrInstantiateFailed = sdkerrors.Register(DefaultCodespace, 3, "instantiate wasm contract failed")

	// ErrExecuteFailed error for rust execution contract failure
	ErrExecuteFailed = sdkerrors.Register(DefaultCodespace, 4, "execute wasm contract failed")

	// ErrGasLimit error for out of gas
	ErrGasLimit = sdkerrors.Register(DefaultCodespace, 5, "insufficient gas")

	// ErrInvalidGenesis error for invalid genesis file syntax
	ErrInvalidGenesis = sdkerrors.Register(DefaultCodespace, 6, "invalid genesis")

	// ErrNotFound error for an entry not found in the store
	ErrNotFound = sdkerrors.Register(DefaultCodespace, 7, "not found")

	// ErrQueryFailed error for rust smart query contract failure
	ErrQueryFailed = sdkerrors.Register(DefaultCodespace, 8, "query wasm contract failed")
)
