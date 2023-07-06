package xadr8

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var _ porttypes.Middleware = &IBCMiddleware{}

// PacketDecoder IBC packet decoder definition
type PacketDecoder interface {
	// DecodeSender returns sender address of the given packet, true
	// false on any issue of when not found
	DecodeSender(packet channeltypes.Packet) (string, bool)
}

var _ PacketDecoder = PacketDecoderFn(nil)

type PacketDecoderFn func(packet channeltypes.Packet) (string, bool)

func (p PacketDecoderFn) DecodeSender(packet channeltypes.Packet) (string, bool) {
	return p(packet)
}

// CallbackExecutor is an abstract callback executor. To be implemented by contract engines
type CallbackExecutor interface {
	// does it make sense to have 2 methods or have a single method with additinal flag for ack/timeout??

	OnIBCPacketAcked(
		ctx sdk.Context,
		packet channeltypes.Packet,
		acknowledgement []byte,
		relayer sdk.AccAddress,
		contractAddr sdk.AccAddress,
		meta CustomDataI,
	) error

	OnIBCPacketTimedOut(
		ctx sdk.Context,
		packet channeltypes.Packet,
		relayer sdk.AccAddress,
		contractAddr sdk.AccAddress,
		meta CustomDataI,
	) error
}

// IBCMiddleware adr-8 middleware
type IBCMiddleware struct {
	porttypes.Middleware
	packetDecoder PacketDecoder
	k             *Keeper
	exec          CallbackExecutor
}

// NewIBCMiddleware constructor with generic decoder
func NewIBCMiddleware[T SenderAuthorized](
	cdc codec.Codec,
	next porttypes.Middleware,
	k *Keeper,
	exec CallbackExecutor,
) *IBCMiddleware {
	return NewIBCMiddleware2(next, k, NewGenericPacketDecoder[T](cdc), exec)
}

// NewIBCMiddleware2 constructor that allows a custom decoder
func NewIBCMiddleware2(
	next porttypes.Middleware,
	k *Keeper,
	decoder PacketDecoder,
	exec CallbackExecutor,
) *IBCMiddleware {
	return &IBCMiddleware{
		Middleware:    next,
		k:             k,
		exec:          exec,
		packetDecoder: decoder,
	}
}

func (m IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	err := m.Middleware.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
	if err != nil {
		return err
	}
	// todo: should panics be handled or ignored
	sender, ok := m.packetDecoder.DecodeSender(packet)
	if !ok {
		return nil
	}
	return m.k.ExecutePacketCallback(ctx, channeltypes.NewPacketID(packet.SourcePort, packet.SourceChannel, packet.Sequence),
		sender,
		ExecutorFn(func(ctx sdk.Context, contractAddr sdk.AccAddress, meta CustomDataI) error {
			return m.exec.OnIBCPacketAcked(ctx, packet, acknowledgement, relayer, contractAddr, meta)
		}))
}

func (m IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	err := m.Middleware.OnTimeoutPacket(ctx, packet, relayer)
	if err != nil {
		return err
	}
	sender, ok := m.packetDecoder.DecodeSender(packet)
	if !ok {
		return nil
	}
	return m.k.ExecutePacketCallback(ctx, channeltypes.NewPacketID(packet.SourcePort, packet.SourceChannel, packet.Sequence),
		sender,
		ExecutorFn(func(ctx sdk.Context, contractAddr sdk.AccAddress, meta CustomDataI) error {
			return m.exec.OnIBCPacketTimedOut(ctx, packet, relayer, contractAddr, meta)
		}))
}

var _ porttypes.Middleware = &IBCMiddleware{}

// IBCMiddlewareAdapter is an adapter to build a Middleware for IBC modules
type IBCMiddlewareAdapter struct {
	porttypes.IBCModule
	w porttypes.ICS4Wrapper
}

// NewIBCMiddlewareAdapter constructor
func NewIBCMiddlewareAdapter(app porttypes.IBCModule, w porttypes.ICS4Wrapper) porttypes.Middleware {
	if w, ok := app.(porttypes.Middleware); ok {
		return w
	}
	return &IBCMiddlewareAdapter{IBCModule: app, w: w}
}

func (m IBCMiddlewareAdapter) SendPacket(ctx sdk.Context, chanCap *capabilitytypes.Capability, sourcePort string, sourceChannel string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte) (sequence uint64, err error) {
	return m.w.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

func (m IBCMiddlewareAdapter) WriteAcknowledgement(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI, ack ibcexported.Acknowledgement) error {
	return m.w.WriteAcknowledgement(ctx, chanCap, packet, ack)
}

func (m IBCMiddlewareAdapter) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return m.w.GetAppVersion(ctx, portID, channelID)
}
