package wasm

import (
	"encoding/json"

	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v11/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

var _ porttypes.ICS4Wrapper = IBCCallbacksICS4Middleware{}

type IBCCallbacksICS4Middleware struct {
	ics4Wrapper porttypes.ICS4Wrapper
}

func NewIBCCallbacksICS4Middleware(ics4Wrapper porttypes.ICS4Wrapper) IBCCallbacksICS4Middleware {
	return IBCCallbacksICS4Middleware{ics4Wrapper: ics4Wrapper}
}

func (m IBCCallbacksICS4Middleware) SendPacket(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	if err := validateSendMemo(data); err != nil {
		return 0, err
	}
	return m.ics4Wrapper.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

func (m IBCCallbacksICS4Middleware) WriteAcknowledgement(ctx sdk.Context, packet ibcexported.PacketI, ack ibcexported.Acknowledgement) error {
	return m.ics4Wrapper.WriteAcknowledgement(ctx, packet, ack)
}

func (m IBCCallbacksICS4Middleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return m.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
}

// Rejects:
//   - src_callback.calldata (no source-side execute path).
//   - ibc_callback + src_callback in the same memo (ambiguous source dispatch).
func validateSendMemo(data []byte) error {
	var packetData transfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(data, &packetData); err != nil {
		return nil
	}

	_, jsonObject := jsonStringHasKey(packetData.Memo, "src_callback")
	if srcObj, ok := jsonObject["src_callback"].(map[string]interface{}); ok {
		if _, hasCalldata := srcObj["calldata"]; hasCalldata {
			return errorsmod.Wrap(types.ErrInvalid, "src_callback must not contain a calldata field")
		}
	}

	_, hasHooksSrc := jsonObject["ibc_callback"]
	_, hasCallbacksSrc := jsonObject["src_callback"]
	if hasHooksSrc && hasCallbacksSrc {
		return errorsmod.Wrap(types.ErrInvalid, "ibc_callback and src_callback must not both be present in the memo")
	}

	return nil
}
