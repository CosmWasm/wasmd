package types

const (
	// WasmModuleEventType is stored with any contract TX
	WasmModuleEventType = "wasm"
	// CustomContractEventPrefix contracts can create custom events. To not mix them with other system events they got the `wasm-` prefix.
	CustomContractEventPrefix = "wasm-"
	EventTypePinCode          = "pin_code"
	EventTypeUnpinCode        = "unpin_code"
)

// event attributes returned from contract execution
const (
	AttributeReservedPrefix  = "_"
	AttributeKeyContractAddr = "_contract_address"
)

// event attributes returned under "message" type - no prefix needed there
const (
	AttributeKeyCodeID     = "code_id"
	AttributeKeySigner     = "signer"
	AttributeResultDataHex = "result"
)
