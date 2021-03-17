package keeper

import (
	"github.com/CosmWasm/wasmd/x/wasm/internal/keeper/wasmtesting"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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

func TestMessageHandlerChainDispatch(t *testing.T) {
	capturingHandler, gotMsgs := wasmtesting.NewCapturingMessageHandler()

	alwaysUnknownMsgHandler := &wasmtesting.MockMessageHandler{
		DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
			return nil, nil, types.ErrUnknownMsg
		}}

	assertNotCalledHandler := &wasmtesting.MockMessageHandler{
		DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
			t.Fatal("not expected to be called")
			return
		}}

	myMsg := wasmvmtypes.CosmosMsg{Custom: []byte(`{}`)}
	specs := map[string]struct {
		handlers []messenger
		expErr   *sdkerrors.Error
	}{
		"single handler": {
			handlers: []messenger{capturingHandler},
		},
		"passed to next handler": {
			handlers: []messenger{alwaysUnknownMsgHandler, capturingHandler},
		},
		"stops iteration when handled": {
			handlers: []messenger{capturingHandler, assertNotCalledHandler},
		},
		"stops iteration on handler error": {
			handlers: []messenger{&wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
					return nil, nil, types.ErrInvalidMsg
				}}, assertNotCalledHandler},
			expErr: types.ErrInvalidMsg,
		},
		"return error when none can handle": {
			handlers: []messenger{alwaysUnknownMsgHandler},
			expErr:   types.ErrUnknownMsg,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			*gotMsgs = make([]wasmvmtypes.CosmosMsg, 0)

			// when
			h := MessageHandlerChain{spec.handlers}
			gotEvents, gotData, gotErr := h.DispatchMsg(sdk.Context{}, RandomAccountAddress(t), "anyPort", myMsg)

			// then
			require.True(t, spec.expErr.Is(gotErr), "exp %v but got %#+v", spec.expErr, gotErr)
			if spec.expErr != nil {
				return
			}
			assert.Equal(t, []wasmvmtypes.CosmosMsg{myMsg}, *gotMsgs)
			assert.Equal(t, [][]byte{{1}}, gotData)
			assert.Nil(t, gotEvents)
		})
	}
}

func TestIBCRawPacketHandler(t *testing.T) {
	ibcPort := "contractsIBCPort"
	var ctx sdk.Context

	var capturedPacket ibcexported.PacketI

	chanKeeper := &wasmtesting.MockChannelKeeper{
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
			capturedPacket = packet
			return nil
		},
	}
	capKeeper := &wasmtesting.MockCapabilityKeeper{
		GetCapabilityFn: func(ctx sdk.Context, name string) (*capabilitytypes.Capability, bool) {
			return &capabilitytypes.Capability{}, true
		},
	}

	specs := map[string]struct {
		srcMsg        wasmvmtypes.SendPacketMsg
		chanKeeper    types.ChannelKeeper
		capKeeper     types.CapabilityKeeper
		expPacketSent channeltypes.Packet
		expErr        *sdkerrors.Error
	}{
		"all good": {
			srcMsg: wasmvmtypes.SendPacketMsg{
				ChannelID:    "channel-1",
				Data:         []byte("myData"),
				TimeoutBlock: &wasmvmtypes.IBCTimeoutBlock{Revision: 1, Height: 2},
			},
			chanKeeper: chanKeeper,
			capKeeper:  capKeeper,
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
		"sequence not found returns error": {
			srcMsg: wasmvmtypes.SendPacketMsg{
				ChannelID:    "channel-1",
				Data:         []byte("myData"),
				TimeoutBlock: &wasmvmtypes.IBCTimeoutBlock{Revision: 1, Height: 2},
			},
			chanKeeper: &wasmtesting.MockChannelKeeper{
				GetNextSequenceSendFn: func(ctx sdk.Context, portID, channelID string) (uint64, bool) {
					return 0, false
				}},
			expErr: channeltypes.ErrSequenceSendNotFound,
		},
		"capability not found returns error": {
			srcMsg: wasmvmtypes.SendPacketMsg{
				ChannelID:    "channel-1",
				Data:         []byte("myData"),
				TimeoutBlock: &wasmvmtypes.IBCTimeoutBlock{Revision: 1, Height: 2},
			},
			chanKeeper: chanKeeper,
			capKeeper: wasmtesting.MockCapabilityKeeper{
				GetCapabilityFn: func(ctx sdk.Context, name string) (*capabilitytypes.Capability, bool) {
					return nil, false
				}},
			expErr: channeltypes.ErrChannelCapabilityNotFound,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			capturedPacket = nil
			// when
			h := NewIBCRawPacketHandler(spec.chanKeeper, spec.capKeeper)
			data, evts, gotErr := h.DispatchMsg(ctx, RandomAccountAddress(t), ibcPort, wasmvmtypes.CosmosMsg{IBC: &wasmvmtypes.IBCMsg{SendPacket: &spec.srcMsg}})
			// then
			require.True(t, spec.expErr.Is(gotErr), "exp %v but got %#+v", spec.expErr, gotErr)
			if spec.expErr != nil {
				return
			}
			assert.Nil(t, data)
			assert.Nil(t, evts)
			assert.Equal(t, spec.expPacketSent, capturedPacket)
		})
	}
}
