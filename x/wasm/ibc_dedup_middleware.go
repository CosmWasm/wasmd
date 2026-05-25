package wasm

import (
	"encoding/json"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

var (
	_ porttypes.IBCModule   = (*IBCDedupMiddleware)(nil)
	_ porttypes.ICS4Wrapper = (*IBCDedupMiddleware)(nil)
	_ porttypes.Middleware  = (*IBCDedupMiddleware)(nil)
)

// IBCDedupMiddleware rejects same-side Hooks/Callbacks memo collisions.
type IBCDedupMiddleware struct {
	porttypes.IBCModule
	porttypes.ICS4Wrapper
}

func NewIBCDedupMiddleware(app porttypes.IBCModule, ics4Wrapper porttypes.ICS4Wrapper) *IBCDedupMiddleware {
	return &IBCDedupMiddleware{IBCModule: app, ICS4Wrapper: ics4Wrapper}
}

func (m *IBCDedupMiddleware) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	if hasMemoCollision(packet.Data, "wasm", "dest_callback") {
		return CreateErrorAcknowledgement(errorsmod.Wrap(types.ErrInvalid, "memo must not contain both wasm (Hooks) and dest_callback (Callbacks)"))
	}
	return m.IBCModule.OnRecvPacket(ctx, channelVersion, packet, relayer)
}

func (m *IBCDedupMiddleware) SendPacket(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	if hasMemoCollision(data, "ibc_callback", "src_callback") {
		return 0, errorsmod.Wrap(types.ErrInvalid, "memo must not contain both ibc_callback (Hooks) and src_callback (Callbacks)")
	}
	return m.ICS4Wrapper.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

func (m *IBCDedupMiddleware) SetUnderlyingApplication(app porttypes.IBCModule) {
	m.IBCModule = app
}

func (m *IBCDedupMiddleware) SetICS4Wrapper(wrapper porttypes.ICS4Wrapper) {
	m.ICS4Wrapper = wrapper
}

func (m *IBCDedupMiddleware) UnmarshalPacketData(ctx sdk.Context, portID string, channelID string, bz []byte) (any, string, error) {
	if unmarshaler, ok := m.IBCModule.(porttypes.PacketDataUnmarshaler); ok {
		return unmarshaler.UnmarshalPacketData(ctx, portID, channelID, bz)
	}
	return nil, "", errorsmod.Wrap(types.ErrInvalid, "underlying app does not implement PacketDataUnmarshaler")
}

// hasMemoCollision returns true if the packet memo contains both
// hooksKey and callbacksKey. Returns false otherwise.
func hasMemoCollision(data []byte, hooksKey, callbacksKey string) bool {
	var pd transfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(data, &pd); err != nil {
		return false
	}
	hasHooks, memo := jsonStringHasKey(pd.Memo, hooksKey)
	_, hasCallbacks := memo[callbacksKey]
	return hasHooks && hasCallbacks
}
