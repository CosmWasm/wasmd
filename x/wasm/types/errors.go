package types

import (
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdkErrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Codes for wasm contract errors
var (
	DefaultCodespace = ModuleName

	// Note: never use code 1 for any errors - that is reserved for ErrInternal in the core cosmos sdk

	// ErrCreateFailed error for wasm code that has already been uploaded or failed
	ErrCreateFailed = sdkErrors.Register(DefaultCodespace, 2, "create wasm contract failed")

	// ErrAccountExists error for a contract account that already exists
	ErrAccountExists = sdkErrors.Register(DefaultCodespace, 3, "contract account already exists")

	// ErrInstantiateFailed error for rust instantiate contract failure
	ErrInstantiateFailed = sdkErrors.Register(DefaultCodespace, 4, "instantiate wasm contract failed")

	// ErrExecuteFailed error for rust execution contract failure
	ErrExecuteFailed = sdkErrors.Register(DefaultCodespace, 5, "execute wasm contract failed")

	// ErrGasLimit error for out of gas
	ErrGasLimit = sdkErrors.Register(DefaultCodespace, 6, "insufficient gas")

	// ErrInvalidGenesis error for invalid genesis file syntax
	ErrInvalidGenesis = sdkErrors.Register(DefaultCodespace, 7, "invalid genesis")

	// ErrNotFound error for an entry not found in the store
	ErrNotFound = sdkErrors.Register(DefaultCodespace, 8, "not found")

	// ErrQueryFailed error for rust smart query contract failure
	ErrQueryFailed = sdkErrors.Register(DefaultCodespace, 9, "query wasm contract failed")

	// ErrInvalidMsg error when we cannot process the error returned from the contract
	ErrInvalidMsg = sdkErrors.Register(DefaultCodespace, 10, "invalid CosmosMsg from the contract")

	// ErrMigrationFailed error for rust execution contract failure
	ErrMigrationFailed = sdkErrors.Register(DefaultCodespace, 11, "migrate wasm contract failed")

	// ErrEmpty error for empty content
	ErrEmpty = sdkErrors.Register(DefaultCodespace, 12, "empty")

	// ErrLimit error for content that exceeds a limit
	ErrLimit = sdkErrors.Register(DefaultCodespace, 13, "exceeds limit")

	// ErrInvalid error for content that is invalid in this context
	ErrInvalid = sdkErrors.Register(DefaultCodespace, 14, "invalid")

	// ErrDuplicate error for content that exists
	ErrDuplicate = sdkErrors.Register(DefaultCodespace, 15, "duplicate")

	// ErrMaxIBCChannels error for maximum number of ibc channels reached
	ErrMaxIBCChannels = sdkErrors.Register(DefaultCodespace, 16, "max transfer channels")

	// ErrUnsupportedForContract error when a capability is used that is not supported for/ by this contract
	ErrUnsupportedForContract = sdkErrors.Register(DefaultCodespace, 17, "unsupported for this contract")

	// ErrPinContractFailed error for pinning contract failures
	ErrPinContractFailed = sdkErrors.Register(DefaultCodespace, 18, "pinning contract failed")

	// ErrUnpinContractFailed error for unpinning contract failures
	ErrUnpinContractFailed = sdkErrors.Register(DefaultCodespace, 19, "unpinning contract failed")

	// ErrUnknownMsg error by a message handler to show that it is not responsible for this message type
	ErrUnknownMsg = sdkErrors.Register(DefaultCodespace, 20, "unknown message from the contract")

	// ErrInvalidEvent error if an attribute/event from the contract is invalid
	ErrInvalidEvent = sdkErrors.Register(DefaultCodespace, 21, "invalid event")

	// ErrNoSuchContractFn error factory for an error when an address does not belong to a contract
	ErrNoSuchContractFn = WasmVMFlavouredErrorFactory(sdkErrors.Register(DefaultCodespace, 22, "no such contract"),
		func(addr string) error { return wasmvmtypes.NoSuchContract{Addr: addr} },
	)

	// code 23 -26 were used for json parser

	// ErrExceedMaxQueryStackSize error if max query stack size is exceeded
	ErrExceedMaxQueryStackSize = sdkErrors.Register(DefaultCodespace, 27, "max query stack size exceeded")

	// ErrNoSuchCodeFn factory for an error when a code id does not belong to a code info
	ErrNoSuchCodeFn = WasmVMFlavouredErrorFactory(sdkErrors.Register(DefaultCodespace, 28, "no such code"),
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
	sdkErr    *sdkErrors.Error
	wasmVMErr error
}

// NewWasmVMFlavouredError constructor
func NewWasmVMFlavouredError(sdkErr *sdkErrors.Error, wasmVMErr error) WasmVMFlavouredError {
	return WasmVMFlavouredError{sdkErr: sdkErr, wasmVMErr: wasmVMErr}
}

// WasmVMFlavouredErrorFactory is a factory method to build a WasmVMFlavouredError type
func WasmVMFlavouredErrorFactory[T any](sdkErr *sdkErrors.Error, wasmVMErrBuilder func(T) error) func(T) WasmVMFlavouredError {
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
func (e WasmVMFlavouredError) Wrap(desc string) error { return sdkErrors.Wrap(e, desc) }

// Wrapf extends this error with additional information.
// It's a handy function to call Wrapf with sdk errors.
func (e WasmVMFlavouredError) Wrapf(desc string, args ...interface{}) error {
	return sdkErrors.Wrapf(e, desc, args...)
}
