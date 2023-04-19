package types

import (
	sdkErrors "github.com/Finschia/finschia-sdk/types/errors"

	wasmtypes "github.com/Finschia/wasmd/x/wasm/types"
)

// Codes for wasm contract errors
var (
	// ErrInactiveContract error if the contract set inactive
	ErrInactiveContract = sdkErrors.Register(wasmtypes.DefaultCodespace, 101, "inactive contract")
)
