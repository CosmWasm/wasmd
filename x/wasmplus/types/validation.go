package types

import (
	sdkerrors "github.com/line/lbm-sdk/types/errors"

	wasmtypes "github.com/line/wasmd/x/wasm/types"
)

func validateWasmCode(s []byte) error {
	if len(s) == 0 {
		return sdkerrors.Wrap(wasmtypes.ErrEmpty, "is required")
	}
	if len(s) > wasmtypes.MaxWasmSize {
		return sdkerrors.Wrapf(wasmtypes.ErrLimit, "cannot be longer than %d bytes", wasmtypes.MaxWasmSize)
	}
	return nil
}

func validateLabel(label string) error {
	if label == "" {
		return sdkerrors.Wrap(wasmtypes.ErrEmpty, "is required")
	}
	if len(label) > wasmtypes.MaxLabelSize {
		return sdkerrors.Wrapf(wasmtypes.ErrLimit, "cannot be longer than %d characters", wasmtypes.MaxWasmSize)
	}
	return nil
}
