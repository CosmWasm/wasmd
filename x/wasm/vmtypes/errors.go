package vmtypes

import (
	"github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	errorsmod "cosmossdk.io/errors"
)

// Codes for wasm contract errors
var (
	DefaultCodespace = types.ModuleName

	// ErrNoSuchContractFn error factory for an error when an address does not belong to a contract
	ErrNoSuchContractFn = WasmVMFlavouredErrorFactory(errorsmod.Register(DefaultCodespace, 22, "no such contract"),
		func(addr string) error { return wasmvmtypes.NoSuchContract{Addr: addr} },
	)

	// code 23 -26 were used for json parser

	// ErrExceedMaxQueryStackSize error if max query stack size is exceeded
	ErrExceedMaxQueryStackSize = errorsmod.Register(DefaultCodespace, 27, "max query stack size exceeded")

	// ErrNoSuchCodeFn factory for an error when a code id does not belong to a code info
	ErrNoSuchCodeFn = WasmVMFlavouredErrorFactory(errorsmod.Register(DefaultCodespace, 28, "no such code"),
		func(id uint64) error { return wasmvmtypes.NoSuchCode{CodeID: id} },
	)
)

// WasmVMErrorable mapped error type in wasmvm and are not redacted
type WasmVMErrorable interface {
	// ToWasmVMError convert instance to wasmvm friendly error if possible otherwise root cause. never nil
	ToWasmVMError() error
}

var _ WasmVMErrorable = WasmVMFlavouredError{}

// WasmVMFlavouredError wrapper for sdk error that supports wasmvm error types
type WasmVMFlavouredError struct {
	sdkErr    *errorsmod.Error
	wasmVMErr error
}

// NewWasmVMFlavouredError constructor
func NewWasmVMFlavouredError(sdkErr *errorsmod.Error, wasmVMErr error) WasmVMFlavouredError {
	return WasmVMFlavouredError{sdkErr: sdkErr, wasmVMErr: wasmVMErr}
}

// WasmVMFlavouredErrorFactory is a factory method to build a WasmVMFlavouredError type
func WasmVMFlavouredErrorFactory[T any](sdkErr *errorsmod.Error, wasmVMErrBuilder func(T) error) func(T) WasmVMFlavouredError {
	if wasmVMErrBuilder == nil {
		panic("builder function required")
	}
	return func(d T) WasmVMFlavouredError {
		return WasmVMFlavouredError{sdkErr: sdkErr, wasmVMErr: wasmVMErrBuilder(d)}
	}
}

// ToWasmVMError implements WasmVMError-able
func (e WasmVMFlavouredError) ToWasmVMError() error {
	if e.wasmVMErr != nil {
		return e.wasmVMErr
	}
	return e.sdkErr
}

// implements stdlib error
func (e WasmVMFlavouredError) Error() string {
	return e.sdkErr.Error()
}

// Unwrap implements the built-in errors.Unwrap
func (e WasmVMFlavouredError) Unwrap() error {
	return e.sdkErr
}

// Cause is the same as unwrap but used by errors.abci
func (e WasmVMFlavouredError) Cause() error {
	return e.Unwrap()
}

// Wrap extends this error with additional information.
// It's a handy function to call Wrap with sdk errors.
func (e WasmVMFlavouredError) Wrap(desc string) error { return errorsmod.Wrap(e, desc) }

// Wrapf extends this error with additional information.
// It's a handy function to call Wrapf with sdk errors.
func (e WasmVMFlavouredError) Wrapf(desc string, args ...interface{}) error {
	return errorsmod.Wrapf(e, desc, args...)
}
