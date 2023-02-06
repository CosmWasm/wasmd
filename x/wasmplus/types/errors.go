package types

import (
	sdkErrors "github.com/line/lbm-sdk/types/errors"

	wasmtypes "github.com/line/wasmd/x/wasm/types"
)

// Codes for wasm contract errors
var (
	// ErrInactiveContract error if the contract set inactive
	ErrInactiveContract = sdkErrors.Register(wasmtypes.DefaultCodespace, 101, "inactive contract")
)
