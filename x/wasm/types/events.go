package types

const (
	// WasmModuleEventType is stored with any contract TX that returns non empty EventAttributes
	WasmModuleEventType = "wasm"
	// CustomContractEventPrefix contracts can create custom events. To not mix them with other system events they got the `wasm-` prefix.
	CustomContractEventPrefix = "wasm-"

	EventTypeStoreCode              = "store_code"
	EventTypeInstantiate            = "instantiate"
	EventTypeExecute                = "execute"
	EventTypeMigrate                = "migrate"
	EventTypePinCode                = "pin_code"
	EventTypeUnpinCode              = "unpin_code"
	EventTypeSudo                   = "sudo"
	EventTypeReply                  = "reply"
	EventTypeGovContractResult      = "gov_contract_result"
	EventTypeUpdateContractAdmin    = "update_contract_admin"
	EventTypeUpdateCodeAccessConfig = "update_code_access_config"
	EventTypePacketRecv             = "ibc_packet_received"
	// add new types to IsAcceptedEventOnRecvPacketErrorAck
)

// IsAcceptedEventOnRecvPacketErrorAck returns true for all wasm event types that do not contain custom attributes
func IsAcceptedEventOnRecvPacketErrorAck(s string) bool {
	for _, v := range []string{
		EventTypeStoreCode,
		EventTypeInstantiate,
		EventTypeExecute,
		// EventTypeMigrate, not true as we rolled back
		// EventTypePinCode, not relevant
		// EventTypeUnpinCode, not relevant
		EventTypeSudo,
		EventTypeReply,
		EventTypeGovContractResult,
		// EventTypeUpdateContractAdmin, not true
		// EventTypeUpdateCodeAccessConfig, not true
		// EventTypePacketRecv, can not happen
	} {
		if s == v {
			return true
		}
	}
	return false
}

// event attributes returned from contract execution
const (
	AttributeReservedPrefix = "_"

	AttributeKeyContractAddr        = "_contract_address"
	AttributeKeyCodeID              = "code_id"
	AttributeKeyChecksum            = "code_checksum"
	AttributeKeyResultDataHex       = "result"
	AttributeKeyRequiredCapability  = "required_capability"
	AttributeKeyNewAdmin            = "new_admin_address"
	AttributeKeyCodePermission      = "code_permission"
	AttributeKeyAuthorizedAddresses = "authorized_addresses"
	AttributeKeyAckSuccess          = "success"
	AttributeKeyAckError            = "error"
)
