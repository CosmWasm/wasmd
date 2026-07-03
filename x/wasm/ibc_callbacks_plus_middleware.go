package wasm

import (
	"encoding/json"
	"fmt"

	callbackstypes "github.com/cosmos/ibc-go/v10/modules/apps/callbacks/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcapi "github.com/cosmos/ibc-go/v10/modules/core/api"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

var (
	_ callbackstypes.CallbacksCompatibleModule   = (*IBCV1CallbacksPlusMiddleware)(nil)
	_ callbackstypes.CallbacksCompatibleModuleV2 = (*IBCV2CallbacksPlusMiddleware)(nil)
)

// Verbatim from https://github.com/cosmos/ibc-apps/blob/main/modules/ibc-hooks/types/keys.go
const SenderPrefix = "ibc-wasm-hook-intermediary"

// Verbatim from https://github.com/cosmos/ibc-apps/blob/main/modules/ibc-hooks/keeper/keeper.go
func DeriveIntermediateSender(channel, originalSender, bech32Prefix string) (string, error) {
	senderStr := fmt.Sprintf("%s/%s", channel, originalSender)
	senderHash32 := address.Hash(SenderPrefix, []byte(senderStr))
	sender := sdk.AccAddress(senderHash32)
	return sdk.Bech32ifyAddressBytes(bech32Prefix, sender)
}

func rewriteReceiverForCalldata(data []byte, destChannel string) []byte {
	var pd transfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(data, &pd); err != nil {
		return data
	}
	if calldata, _ := getCallbackCalldataFromKey(pd, callbackstypes.DestinationCallbackKey); len(calldata) == 0 {
		return data
	}
	intermediate, err := DeriveIntermediateSender(destChannel, pd.Sender, sdk.GetConfig().GetBech32AccountAddrPrefix())
	if err != nil {
		return data
	}
	pd.Receiver = intermediate
	out, err := json.Marshal(pd)
	if err != nil {
		return data
	}
	return out
}

func getCallbackCalldataFromKey(packetData any, key string) ([]byte, error) {
	cbData, isCb, err := callbackstypes.GetCallbackData(
		packetData, "", "", 0, DefaultMaxIBCCallbackGas, key,
	)
	if isCb && err != nil {
		return nil, err
	}
	return cbData.Calldata, nil
}

// IBCV1CallbacksPlusMiddleware rewrites the recv packet's Receiver to the
// intermediate sender when memo carries dest_callback.calldata.
type IBCV1CallbacksPlusMiddleware struct {
	callbackstypes.CallbacksCompatibleModule
}

func NewIBCV1CallbacksPlusMiddleware(app porttypes.IBCModule) *IBCV1CallbacksPlusMiddleware {
	compat, ok := app.(callbackstypes.CallbacksCompatibleModule)
	if !ok {
		panic(fmt.Errorf("underlying application does not implement %T", (*callbackstypes.CallbacksCompatibleModule)(nil)))
	}
	return &IBCV1CallbacksPlusMiddleware{CallbacksCompatibleModule: compat}
}

func (m *IBCV1CallbacksPlusMiddleware) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	packet.Data = rewriteReceiverForCalldata(packet.Data, packet.DestinationChannel)
	return m.CallbacksCompatibleModule.OnRecvPacket(ctx, channelVersion, packet, relayer)
}

// IBCV2CallbacksPlusMiddleware rewrites the recv packet's Receiver to the
// intermediate sender when memo carries dest_callback.calldata.
type IBCV2CallbacksPlusMiddleware struct {
	callbackstypes.CallbacksCompatibleModuleV2
}

func NewIBCV2CallbacksPlusMiddleware(app ibcapi.IBCModule) *IBCV2CallbacksPlusMiddleware {
	compat, ok := app.(callbackstypes.CallbacksCompatibleModuleV2)
	if !ok {
		panic(fmt.Errorf("underlying application does not implement %T", (*callbackstypes.CallbacksCompatibleModuleV2)(nil)))
	}
	return &IBCV2CallbacksPlusMiddleware{CallbacksCompatibleModuleV2: compat}
}

func (m *IBCV2CallbacksPlusMiddleware) OnRecvPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) channeltypesv2.RecvPacketResult {
	if payload.SourcePort == transfertypes.PortID && payload.DestinationPort == transfertypes.PortID {
		switch payload.Encoding {
		case "", transfertypes.EncodingJSON:
			payload.Value = rewriteReceiverForCalldata(payload.Value, destinationClient)
		default:
			// calldata is only supported with JSON encoding; reject it on other encodings
			if destCallbackHasCalldata(payload) {
				return channeltypesv2.RecvPacketResult{Status: channeltypesv2.PacketStatus_Failure}
			}
		}
	}
	return m.CallbacksCompatibleModuleV2.OnRecvPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)
}

func destCallbackHasCalldata(payload channeltypesv2.Payload) bool {
	pd, err := transfertypes.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if err != nil {
		return false
	}
	calldata, _ := getCallbackCalldataFromKey(pd, callbackstypes.DestinationCallbackKey)
	return len(calldata) != 0
}
