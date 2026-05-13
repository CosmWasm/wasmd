package wasm

import (
	"encoding/json"

	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v11/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

var (
	_ porttypes.IBCModule   = (*IBCDedupMiddleware)(nil)
	_ porttypes.ICS4Wrapper = (*IBCDedupMiddleware)(nil)
	_ porttypes.Middleware  = (*IBCDedupMiddleware)(nil)
)

type IBCDedupMiddleware struct {
	app         porttypes.IBCModule
	ics4Wrapper porttypes.ICS4Wrapper
}

func NewIBCDedupMiddleware(app porttypes.IBCModule, ics4Wrapper porttypes.ICS4Wrapper) *IBCDedupMiddleware {
	return &IBCDedupMiddleware{app: app, ics4Wrapper: ics4Wrapper}
}

func (m *IBCDedupMiddleware) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	if modified := stripCallbacksOnHooksCollision(packet.Data); modified != nil {
		packet.Data = modified
	}
	return m.app.OnRecvPacket(ctx, channelVersion, packet, relayer)
}

func (m *IBCDedupMiddleware) SendPacket(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	if modified := stripCallbacksOnHooksCollision(data); modified != nil {
		data = modified
	}
	return m.ics4Wrapper.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

func (m *IBCDedupMiddleware) WriteAcknowledgement(ctx sdk.Context, packet ibcexported.PacketI, ack ibcexported.Acknowledgement) error {
	return m.ics4Wrapper.WriteAcknowledgement(ctx, packet, ack)
}

func (m *IBCDedupMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return m.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
}

// If the memo has both a hooks key (wasm | ibc_callback) and a callbacks key
// (dest_callback | src_callback), strip both callbacks keys (hooks wins).
// Returns nil when there's nothing to change.
func stripCallbacksOnHooksCollision(data []byte) []byte {
	var packetData transfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(data, &packetData); err != nil {
		return nil
	}
	_, jsonObject := jsonStringHasKey(packetData.Memo, "wasm")
	_, hasWasm := jsonObject["wasm"]
	_, hasIBCCallback := jsonObject["ibc_callback"]
	if !hasWasm && !hasIBCCallback {
		return nil
	}
	_, hasDest := jsonObject["dest_callback"]
	_, hasSrc := jsonObject["src_callback"]
	if !hasDest && !hasSrc {
		return nil
	}

	delete(jsonObject, "dest_callback")
	delete(jsonObject, "src_callback")
	if len(jsonObject) == 0 {
		packetData.Memo = ""
	} else {
		bz, err := json.Marshal(jsonObject)
		if err != nil {
			return nil
		}
		packetData.Memo = string(bz)
	}
	out, err := json.Marshal(packetData)
	if err != nil {
		return nil
	}
	return out
}

func (m *IBCDedupMiddleware) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, counterParty channeltypes.Counterparty, version string) (string, error) {
	return m.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterParty, version)
}

func (m *IBCDedupMiddleware) OnChanOpenTry(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, counterParty channeltypes.Counterparty, counterpartyVersion string) (string, error) {
	return m.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterParty, counterpartyVersion)
}

func (m *IBCDedupMiddleware) OnChanOpenAck(ctx sdk.Context, portID, channelID, counterpartyChannelID, counterpartyVersion string) error {
	return m.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

func (m *IBCDedupMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return m.app.OnChanOpenConfirm(ctx, portID, channelID)
}

func (m *IBCDedupMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return m.app.OnChanCloseInit(ctx, portID, channelID)
}

func (m *IBCDedupMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return m.app.OnChanCloseConfirm(ctx, portID, channelID)
}

func (m *IBCDedupMiddleware) OnAcknowledgementPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	return m.app.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
}

func (m *IBCDedupMiddleware) OnTimeoutPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	return m.app.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
}

func (m *IBCDedupMiddleware) SetUnderlyingApplication(app porttypes.IBCModule) {
	m.app = app
}

func (m *IBCDedupMiddleware) SetICS4Wrapper(wrapper porttypes.ICS4Wrapper) {
	m.ics4Wrapper = wrapper
}

func (m *IBCDedupMiddleware) UnmarshalPacketData(ctx sdk.Context, portID string, channelID string, bz []byte) (any, string, error) {
	if unmarshaler, ok := m.app.(porttypes.PacketDataUnmarshaler); ok {
		return unmarshaler.UnmarshalPacketData(ctx, portID, channelID, bz)
	}
	return nil, "", errorsmod.Wrap(types.ErrInvalid, "underlying app does not implement PacketDataUnmarshaler")
}
