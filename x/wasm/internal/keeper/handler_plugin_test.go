package keeper

import (
	"github.com/CosmWasm/wasmd/x/wasm/internal/keeper/wasmtesting"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	ibcexported "github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"testing"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeIBCSendPacket(t *testing.T) {
	ibcPort := "contractsIBCPort"
	var ctx sdk.Context
	specs := map[string]struct {
		srcMsg        wasmvmtypes.SendPacketMsg
		expPacketSent channeltypes.Packet
	}{
		"all good": {
			srcMsg: wasmvmtypes.SendPacketMsg{
				ChannelID:    "channel-1",
				Data:         []byte("myData"),
				TimeoutBlock: &wasmvmtypes.IBCTimeoutBlock{Revision: 1, Height: 2},
			},
			expPacketSent: channeltypes.Packet{
				Sequence:           1,
				SourcePort:         ibcPort,
				SourceChannel:      "channel-1",
				DestinationPort:    "other-port",
				DestinationChannel: "other-channel-1",
				Data:               []byte("myData"),
				TimeoutHeight:      clienttypes.Height{RevisionNumber: 1, RevisionHeight: 2},
			},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			var gotPacket ibcexported.PacketI

			var chanKeeper types.ChannelKeeper = &wasmtesting.MockChannelKeeper{
				GetNextSequenceSendFn: func(ctx sdk.Context, portID, channelID string) (uint64, bool) {
					return 1, true
				},
				GetChannelFn: func(ctx sdk.Context, srcPort, srcChan string) (channeltypes.Channel, bool) {
					return channeltypes.Channel{
						Counterparty: channeltypes.NewCounterparty(
							"other-port",
							"other-channel-1",
						)}, true
				},
				SendPacketFn: func(ctx sdk.Context, channelCap *capabilitytypes.Capability, packet ibcexported.PacketI) error {
					gotPacket = packet
					return nil
				},
			}
			var capKeeper types.CapabilityKeeper = &wasmtesting.MockCapabilityKeeper{
				GetCapabilityFn: func(ctx sdk.Context, name string) (*capabilitytypes.Capability, bool) {
					return &capabilitytypes.Capability{}, true
				},
			}
			sender := RandomAccountAddress(t)
			h := NewIBCRawPacketHandler(chanKeeper, capKeeper)
			data, evts, err := h.DispatchMsg(ctx, sender, ibcPort, wasmvmtypes.CosmosMsg{IBC: &wasmvmtypes.IBCMsg{SendPacket: &spec.srcMsg}})
			require.NoError(t, err)
			assert.Nil(t, data)
			assert.Nil(t, evts)
			assert.Equal(t, spec.expPacketSent, gotPacket)
		})
	}
}
