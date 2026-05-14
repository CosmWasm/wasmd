package wasm

import (
	"encoding/json"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcapi "github.com/cosmos/ibc-go/v10/modules/core/api"
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

var _ ibcapi.IBCModule = IBCV2CallbacksPlusMiddleware{}

type IBCV2CallbacksPlusMiddleware struct {
	app ibcapi.IBCModule
}

func NewIBCV2CallbacksPlusMiddleware(app ibcapi.IBCModule) IBCV2CallbacksPlusMiddleware {
	return IBCV2CallbacksPlusMiddleware{app: app}
}

func (m IBCV2CallbacksPlusMiddleware) OnSendPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	signer sdk.AccAddress,
) error {
	if payload.SourcePort == transfertypes.PortID {
		if data, err := transfertypes.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding); err == nil {
			if err := validateMemo(data.Memo); err != nil {
				return err
			}
		}
	}
	return m.app.OnSendPacket(ctx, sourceClient, destinationClient, sequence, payload, signer)
}

func (m IBCV2CallbacksPlusMiddleware) OnRecvPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) channeltypesv2.RecvPacketResult {
	return m.app.OnRecvPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)
}

func (m IBCV2CallbacksPlusMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	acknowledgement []byte,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	return m.app.OnAcknowledgementPacket(ctx, sourceClient, destinationClient, sequence, acknowledgement, payload, relayer)
}

func (m IBCV2CallbacksPlusMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	return m.app.OnTimeoutPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)
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
