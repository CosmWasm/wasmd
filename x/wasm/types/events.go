package types

import (
	"fmt"

	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
	EventTypeUpdateContractLabel    = "update_contract_label"
	EventTypeUpdateCodeAccessConfig = "update_code_access_config"
	EventTypePacketRecv             = "ibc_packet_received"
	// add new types to IsAcceptedEventOnRecvPacketErrorAck
)

// EmitAcknowledgementEvent emits an event signaling a successful or failed acknowledgement and including the error
// details if any.
func EmitAcknowledgementEvent(ctx sdk.Context, contractAddr sdk.AccAddress, ack exported.Acknowledgement, err error) {
	success := err == nil && (ack == nil || ack.Success())
	emitEvent(ctx, contractAddr, success, err)
}

func EmitAcknowledgementIBC2Event(ctx sdk.Context, contractAddr sdk.AccAddress, ack channeltypesv2.RecvPacketResult, err error) {
	success := err == nil && (ack.Acknowledgement == nil || ack.Status == channeltypesv2.PacketStatus_Success)
	emitEvent(ctx, contractAddr, success, err)
}

func emitEvent(ctx sdk.Context, contractAddr sdk.AccAddress, success bool, err error) {
	attributes := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyModule, ModuleName),
		sdk.NewAttribute(AttributeKeyContractAddr, contractAddr.String()),
		sdk.NewAttribute(AttributeKeyAckSuccess, fmt.Sprintf("%t", success)),
	}

	if err != nil {
		attributes = append(attributes, sdk.NewAttribute(AttributeKeyAckError, err.Error()))
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			EventTypePacketRecv,
			attributes...,
		),
	)
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
	AttributeKeyNewLabel            = "new_label"
	AttributeKeyCodePermission      = "code_permission"
	AttributeKeyAuthorizedAddresses = "authorized_addresses"
	AttributeKeyAckSuccess          = "success"
	AttributeKeyAckError            = "error"
)
