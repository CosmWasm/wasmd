package keeper

import (
	"errors"
	"fmt"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// msgEncoder is an extension point to customize encodings
type msgEncoder interface {
	// Encode converts wasmvm message to n cosmos message types
	Encode(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) ([]sdk.Msg, error)
}

// MessageRouter ADR 031 request type routing
type MessageRouter interface {
	Handler(msg sdk.Msg) baseapp.MsgServiceHandler
}

// SDKMessageHandler can handles messages that can be encoded into sdk.Message types and routed.
type SDKMessageHandler struct {
	router   MessageRouter
	encoders msgEncoder
	cdc      codec.Codec
}

// NewDefaultMessageHandler constructor
func NewDefaultMessageHandler(
	keeper *Keeper,
	router MessageRouter,
	ics4Wrapper types.ICS4Wrapper,
	channelKeeperV2 types.ChannelKeeperV2,
	bankKeeper types.Burner,
	cdc codec.Codec,
	portSource types.ICS20TransferPortSource,
	customEncoders ...*MessageEncoders,
) Messenger {
	encoders := DefaultEncoders(cdc, portSource)
	for _, e := range customEncoders {
		encoders = encoders.Merge(e)
	}
	return NewMessageHandlerChain(
		NewSDKMessageHandler(cdc, router, encoders),
		NewIBCRawPacketHandler(ics4Wrapper, keeper),
		NewIBC2RawPacketHandler(channelKeeperV2),
		NewBurnCoinMessageHandler(bankKeeper),
	)
}

func NewSDKMessageHandler(cdc codec.Codec, router MessageRouter, encoders msgEncoder) SDKMessageHandler {
	return SDKMessageHandler{
		cdc:      cdc,
		router:   router,
		encoders: encoders,
	}
}

func (h SDKMessageHandler) DispatchMsg(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, msgResponses [][]*codectypes.Any, err error) {
	sdkMsgs, err := h.encoders.Encode(ctx, contractAddr, contractIBCPortID, msg)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, sdkMsg := range sdkMsgs {
		res, err := h.handleSdkMessage(ctx, contractAddr, sdkMsg)
		if err != nil {
			return nil, nil, nil, err
		}
		// append data and msgResponses
		data = append(data, res.Data)
		msgResponses = append(msgResponses, res.MsgResponses)
		// append events
		sdkEvents := make([]sdk.Event, len(res.Events))
		for i := range res.Events {
			sdkEvents[i] = sdk.Event(res.Events[i])
		}
		events = append(events, sdkEvents...)
	}
	return
}

func (h SDKMessageHandler) handleSdkMessage(ctx sdk.Context, contractAddr sdk.Address, msg sdk.Msg) (*sdk.Result, error) {
	// Perform message validation and authorization checks
	if m, ok := msg.(sdk.HasValidateBasic); ok {
		if err := m.ValidateBasic(); err != nil {
			return nil, errorsmod.Wrapf(err, "failed basic validation for message type %T", msg)
		}
	}

	// Verify contract address has proper authorization
	signers, _, err := h.cdc.GetMsgV1Signers(msg)
	if err != nil {
		return nil, err
	}
	for _, acct := range signers {
		if !contractAddr.Equals(sdk.AccAddress(acct)) {
			return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "contract doesn't have permission")
		}
	}
	// --- end block

	// find the handler and execute it
	if handler := h.router.Handler(msg); handler != nil {
		// ADR 031 request type routing
		msgResult, err := handler(ctx, msg)
		return msgResult, err
	}
	// legacy sdk.Msg routing
	// Assuming that the app developer has migrated all their Msgs to
	// proto messages and has registered all `Msg services`, then this
	// path should never be called, because all those Msgs should be
	// registered within the `msgServiceRouter` already.
	return nil, errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "can't route message %+v", msg)
}

type callDepthMessageHandler struct {
	Messenger
	MaxCallDepth uint32
}

func (h callDepthMessageHandler) DispatchMsg(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, msgResponses [][]*codectypes.Any, err error) {
	ctx, err = checkAndIncreaseCallDepth(ctx, h.MaxCallDepth)
	if err != nil {
		return nil, nil, nil, errorsmod.Wrap(err, "dispatch")
	}

	return h.Messenger.DispatchMsg(ctx, contractAddr, contractIBCPortID, msg)
}

// MessageHandlerChain defines a chain of handlers that are called one by one until it can be handled.
type MessageHandlerChain struct {
	handlers []Messenger
}

func NewMessageHandlerChain(first Messenger, others ...Messenger) *MessageHandlerChain {
	r := &MessageHandlerChain{handlers: append([]Messenger{first}, others...)}
	for i := range r.handlers {
		if r.handlers[i] == nil {
			panic(fmt.Sprintf("handler must not be nil at position : %d", i))
		}
	}
	return r
}

// DispatchMsg dispatch message and calls chained handlers one after another in
// order to find the right one to process given message. If a handler cannot
// process given message (returns ErrUnknownMsg), its result is ignored and the
// next handler is executed.
func (m MessageHandlerChain) DispatchMsg(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) ([]sdk.Event, [][]byte, [][]*codectypes.Any, error) {
	for _, h := range m.handlers {
		events, data, msgResponses, err := h.DispatchMsg(ctx, contractAddr, contractIBCPortID, msg)
		switch {
		case err == nil:
			return events, data, msgResponses, nil
		case errors.Is(err, types.ErrUnknownMsg):
			continue
		default:
			return events, data, msgResponses, err
		}
	}
	return nil, nil, nil, errorsmod.Wrap(types.ErrUnknownMsg, "no handler found")
}

// IBCRawPacketHandler handles IBC.SendPacket messages which are published to an IBC channel.
type IBCRawPacketHandler struct {
	ics4Wrapper types.ICS4Wrapper
	wasmKeeper  types.IBCContractKeeper
}

// NewIBCRawPacketHandler constructor
func NewIBCRawPacketHandler(ics4Wrapper types.ICS4Wrapper, wasmKeeper types.IBCContractKeeper) IBCRawPacketHandler {
	return IBCRawPacketHandler{
		ics4Wrapper: ics4Wrapper,
		wasmKeeper:  wasmKeeper,
	}
}

// DispatchMsg publishes a raw IBC packet onto the channel.
func (h IBCRawPacketHandler) DispatchMsg(ctx sdk.Context, _ sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) ([]sdk.Event, [][]byte, [][]*codectypes.Any, error) {
	if msg.IBC == nil {
		return nil, nil, nil, types.ErrUnknownMsg
	}
	switch {
	case msg.IBC.SendPacket != nil:
		if contractIBCPortID == "" {
			return nil, nil, nil, errorsmod.Wrapf(types.ErrUnsupportedForContract, "ibc not supported")
		}
		contractIBCChannelID := msg.IBC.SendPacket.ChannelID
		if contractIBCChannelID == "" {
			return nil, nil, nil, errorsmod.Wrapf(types.ErrEmpty, "ibc channel")
		}

		seq, err := h.ics4Wrapper.SendPacket(ctx, contractIBCPortID, contractIBCChannelID, ConvertWasmIBCTimeoutHeightToCosmosHeight(msg.IBC.SendPacket.Timeout.Block), msg.IBC.SendPacket.Timeout.Timestamp, msg.IBC.SendPacket.Data)
		if err != nil {
			return nil, nil, nil, errorsmod.Wrap(err, "channel")
		}
		moduleLogger(ctx).Debug("ibc packet set", "seq", seq)

		resp := &types.MsgIBCSendResponse{Sequence: seq}
		val, err := resp.Marshal()
		if err != nil {
			return nil, nil, nil, errorsmod.Wrap(err, "failed to marshal IBC send response")
		}
		any, err := codectypes.NewAnyWithValue(resp)
		if err != nil {
			return nil, nil, nil, errorsmod.Wrap(err, "failed to convert IBC send response to Any")
		}
		msgResponses := [][]*codectypes.Any{{any}}

		return nil, [][]byte{val}, msgResponses, nil
	case msg.IBC.WriteAcknowledgement != nil:
		if contractIBCPortID == "" {
			return nil, nil, nil, errorsmod.Wrapf(types.ErrUnsupportedForContract, "ibc not supported")
		}
		contractIBCChannelID := msg.IBC.WriteAcknowledgement.ChannelID
		if contractIBCChannelID == "" {
			return nil, nil, nil, errorsmod.Wrapf(types.ErrEmpty, "ibc channel")
		}

		packet, err := h.wasmKeeper.LoadAsyncAckPacket(ctx, contractIBCPortID, contractIBCChannelID, msg.IBC.WriteAcknowledgement.PacketSequence)
		if err != nil {
			return nil, nil, nil, errorsmod.Wrap(types.ErrInvalid, "packet")
		}

		err = h.ics4Wrapper.WriteAcknowledgement(ctx, packet, ContractConfirmStateAck(msg.IBC.WriteAcknowledgement.Ack.Data))
		if err != nil {
			return nil, nil, nil, errorsmod.Wrap(err, "acknowledgement")
		}

		// Delete the packet from the store after acknowledgement.
		// This ensures WriteAcknowledgement can only be used once per packet
		// such that overriding the acknowledgement later on is not possible.
		h.wasmKeeper.DeleteAsyncAckPacket(ctx, contractIBCPortID, contractIBCChannelID, msg.IBC.WriteAcknowledgement.PacketSequence)

		resp := &types.MsgIBCWriteAcknowledgementResponse{}
		val, err := resp.Marshal()
		if err != nil {
			return nil, nil, nil, errorsmod.Wrap(err, "failed to marshal IBC send response")
		}

		any, err := codectypes.NewAnyWithValue(resp)
		if err != nil {
			return nil, nil, nil, errorsmod.Wrap(err, "failed to convert IBC send response to Any")
		}
		msgResponses := [][]*codectypes.Any{{any}}

		return nil, [][]byte{val}, msgResponses, nil
	default:
		return nil, nil, nil, types.ErrUnknownMsg
	}
}

// IBC2RawPacketHandler handles IBC2Msg received from CosmWasm.
type IBC2RawPacketHandler struct {
	channelKeeperV2 types.ChannelKeeperV2
}

// NewIBC2RawPacketHandler constructor
func NewIBC2RawPacketHandler(channelKeeperV2 types.ChannelKeeperV2) IBC2RawPacketHandler {
	return IBC2RawPacketHandler{
		channelKeeperV2: channelKeeperV2,
	}
}

// DispatchMsg publishes a raw IBC2 packet onto the channel.
func (h IBC2RawPacketHandler) DispatchMsg(ctx sdk.Context,
	contractAddr sdk.AccAddress, contractIBC2PortID string, msg wasmvmtypes.CosmosMsg,
) ([]sdk.Event, [][]byte, [][]*codectypes.Any, error) {
	if msg.IBC2 == nil {
		return nil, nil, nil, types.ErrUnknownMsg
	}
	switch {
	case msg.IBC2.WriteAcknowledgement != nil:
		packet := msg.IBC2.WriteAcknowledgement
		if contractIBC2PortID == "" {
			return nil, nil, nil, errorsmod.Wrapf(types.ErrUnsupportedForContract, "ibc2 not supported")
		}
		sourceClient := msg.IBC2.WriteAcknowledgement.SourceClient
		if sourceClient == "" {
			return nil, nil, nil, errorsmod.Wrapf(types.ErrEmpty, "ibc2 channel")
		}

		err := h.channelKeeperV2.WriteAcknowledgement(
			ctx,
			packet.SourceClient,
			packet.PacketSequence,
			channeltypesv2.Acknowledgement{AppAcknowledgements: [][]byte{msg.IBC2.WriteAcknowledgement.Ack.Data}},
		)
		if err != nil {
			return nil, nil, nil, errorsmod.Wrap(err, "ibc2 Write Acknowledgement")
		}

		resp := &types.MsgIBCWriteAcknowledgementResponse{}
		val, err := resp.Marshal()
		if err != nil {
			return nil, nil, nil, errorsmod.Wrap(err, "failed to marshal IBC2 Write Acknowledgement")
		}

		any, err := codectypes.NewAnyWithValue(resp)
		if err != nil {
			return nil, nil, nil, errorsmod.Wrap(err, "failed to convert IBC2 Write Acknowledgement to Any")
		}
		msgResponses := [][]*codectypes.Any{{any}}

		return nil, [][]byte{val}, msgResponses, nil
	default:
		return nil, nil, nil, types.ErrUnknownMsg
	}
}

var _ Messenger = MessageHandlerFunc(nil)

// MessageHandlerFunc is a helper to construct a function based message handler.
type MessageHandlerFunc func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, msgResponses [][]*codectypes.Any, err error)

// DispatchMsg delegates dispatching of provided message into the MessageHandlerFunc.
func (m MessageHandlerFunc) DispatchMsg(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, msgResponses [][]*codectypes.Any, err error) {
	return m(ctx, contractAddr, contractIBCPortID, msg)
}

// NewBurnCoinMessageHandler handles wasmvm.BurnMsg messages
func NewBurnCoinMessageHandler(burner types.Burner) MessageHandlerFunc {
	return func(ctx sdk.Context, contractAddr sdk.AccAddress, _ string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, msgResponses [][]*codectypes.Any, err error) {
		if msg.Bank != nil && msg.Bank.Burn != nil {
			coins, err := ConvertWasmCoinsToSdkCoins(msg.Bank.Burn.Amount)
			if err != nil {
				return nil, nil, nil, err
			}
			if coins.IsZero() {
				return nil, nil, nil, types.ErrEmpty.Wrap("amount")
			}
			if err := burner.SendCoinsFromAccountToModule(ctx, contractAddr, types.ModuleName, coins); err != nil {
				return nil, nil, nil, errorsmod.Wrap(err, "transfer to module")
			}
			if err := burner.BurnCoins(ctx, types.ModuleName, coins); err != nil {
				return nil, nil, nil, errorsmod.Wrap(err, "burn coins")
			}
			moduleLogger(ctx).Info("Burned", "amount", coins)
			return nil, nil, nil, nil
		}
		return nil, nil, nil, types.ErrUnknownMsg
	}
}
