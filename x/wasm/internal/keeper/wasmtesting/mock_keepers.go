package wasmtesting

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	ibcexported "github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
)

type MockChannelKeeper struct {
	GetChannelFn          func(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool)
	GetNextSequenceSendFn func(ctx sdk.Context, portID, channelID string) (uint64, bool)
	SendPacketFn          func(ctx sdk.Context, channelCap *capabilitytypes.Capability, packet ibcexported.PacketI) error
	ChanCloseInitFn       func(ctx sdk.Context, portID, channelID string, chanCap *capabilitytypes.Capability) error
	GetAllChannelsFn      func(ctx sdk.Context) []channeltypes.IdentifiedChannel
}

func (m *MockChannelKeeper) GetChannel(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool) {
	if m.GetChannelFn == nil {
		panic("not supposed to be called!")
	}
	return m.GetChannelFn(ctx, srcPort, srcChan)
}

func (m *MockChannelKeeper) GetAllChannels(ctx sdk.Context) []channeltypes.IdentifiedChannel {
	if m.GetAllChannelsFn == nil {
		panic("not supposed to be called!")
	}
	return m.GetAllChannelsFn(ctx)
}

// Auto-implemented from GetAllChannels data
func (m *MockChannelKeeper) IterateChannels(ctx sdk.Context, cb func(channeltypes.IdentifiedChannel) bool) {
	channels := m.GetAllChannels(ctx)
	for _, channel := range channels {
		stop := cb(channel)
		if stop {
			break
		}
	}
}

func (m *MockChannelKeeper) GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool) {
	if m.GetNextSequenceSendFn == nil {
		panic("not supposed to be called!")
	}
	return m.GetNextSequenceSendFn(ctx, portID, channelID)
}

func (m *MockChannelKeeper) SendPacket(ctx sdk.Context, channelCap *capabilitytypes.Capability, packet ibcexported.PacketI) error {
	if m.SendPacketFn == nil {
		panic("not supposed to be called!")
	}
	return m.SendPacketFn(ctx, channelCap, packet)
}

func (m *MockChannelKeeper) ChanCloseInit(ctx sdk.Context, portID, channelID string, chanCap *capabilitytypes.Capability) error {
	if m.ChanCloseInitFn == nil {
		panic("not supposed to be called!")
	}
	return m.ChanCloseInitFn(ctx, portID, channelID, chanCap)
}

type MockCapabilityKeeper struct {
	GetCapabilityFn          func(ctx sdk.Context, name string) (*capabilitytypes.Capability, bool)
	ClaimCapabilityFn        func(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error
	AuthenticateCapabilityFn func(ctx sdk.Context, capability *capabilitytypes.Capability, name string) bool
}

func (m MockCapabilityKeeper) GetCapability(ctx sdk.Context, name string) (*capabilitytypes.Capability, bool) {
	if m.GetCapabilityFn == nil {
		panic("not supposed to be called!")
	}
	return m.GetCapabilityFn(ctx, name)
}

func (m MockCapabilityKeeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	if m.ClaimCapabilityFn == nil {
		panic("not supposed to be called!")
	}
	return m.ClaimCapabilityFn(ctx, cap, name)

}

func (m MockCapabilityKeeper) AuthenticateCapability(ctx sdk.Context, capability *capabilitytypes.Capability, name string) bool {
	if m.AuthenticateCapabilityFn == nil {
		panic("not supposed to be called!")
	}
	return m.AuthenticateCapabilityFn(ctx, capability, name)

}
