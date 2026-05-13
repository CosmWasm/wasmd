package wasm

import (
	"encoding/json"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

var _ porttypes.ICS4Wrapper = IBCV1CallbacksPlusMiddleware{}

type IBCV1CallbacksPlusMiddleware struct {
	ics4Wrapper porttypes.ICS4Wrapper
}

func NewIBCV1CallbacksPlusMiddleware(ics4Wrapper porttypes.ICS4Wrapper) IBCV1CallbacksPlusMiddleware {
	return IBCV1CallbacksPlusMiddleware{ics4Wrapper: ics4Wrapper}
}

func (m IBCV1CallbacksPlusMiddleware) SendPacket(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	var packetData transfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(data, &packetData); err == nil {
		if err := validateMemo(packetData.Memo); err != nil {
			return 0, err
		}
	}
	return m.ics4Wrapper.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

func (m IBCV1CallbacksPlusMiddleware) WriteAcknowledgement(ctx sdk.Context, packet ibcexported.PacketI, ack ibcexported.Acknowledgement) error {
	return m.ics4Wrapper.WriteAcknowledgement(ctx, packet, ack)
}

func (m IBCV1CallbacksPlusMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return m.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
}

// Rejects:
//   - src_callback.calldata (no source-side execute path).
//   - ibc_callback + src_callback in the same memo (ambiguous source dispatch).
func validateMemo(memo string) error {
	_, jsonObject := jsonStringHasKey(memo, "src_callback")
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
