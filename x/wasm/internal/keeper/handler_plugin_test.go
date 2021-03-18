package keeper

import (
	"encoding/json"
	"github.com/CosmWasm/wasmd/x/wasm/internal/keeper/wasmtesting"
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	ibcexported "github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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
		handlers  []messenger
		expErr    *sdkerrors.Error
		expEvents []sdk.Event
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
		"return events when handle": {
			handlers: []messenger{&wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
					_, data, _ = capturingHandler.DispatchMsg(ctx, contractAddr, contractIBCPortID, msg)
					return []sdk.Event{sdk.NewEvent("myEvent", sdk.NewAttribute("foo", "bar"))}, data, nil
				}},
			},
			expEvents: []sdk.Event{sdk.NewEvent("myEvent", sdk.NewAttribute("foo", "bar"))},
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
			assert.Equal(t, [][]byte{{1}}, gotData) // {1} is default in capturing handler
			assert.Equal(t, spec.expEvents, gotEvents)
		})
	}
}

func TestSDKMessageHandlerDispatch(t *testing.T) {
	myEvent := sdk.NewEvent("myEvent", sdk.NewAttribute("foo", "bar"))
	const myData = "myData"
	myRouterResult := sdk.Result{
		Data:   []byte(myData),
		Events: sdk.Events{myEvent}.ToABCIEvents(),
	}

	var gotMsg []sdk.Msg
	capturingRouteFn := func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		gotMsg = append(gotMsg, msg)
		return &myRouterResult, nil
	}

	myContractAddr := RandomAccountAddress(t)
	myContractMessage := wasmvmtypes.CosmosMsg{Custom: []byte("{}")}

	specs := map[string]struct {
		srcRoute         sdk.Route
		srcEncoder       CustomEncoder
		expErr           *sdkerrors.Error
		expMsgDispatched int
	}{
		"all good": {
			srcRoute: sdk.NewRoute(types.RouterKey, capturingRouteFn),
			srcEncoder: func(sender sdk.AccAddress, msg json.RawMessage) ([]sdk.Msg, error) {
				myMsg := types.MsgExecuteContract{
					Sender:   myContractAddr.String(),
					Contract: RandomBech32AccountAddress(t),
					Msg:      []byte("{}"),
				}
				return []sdk.Msg{&myMsg}, nil
			},
			expMsgDispatched: 1,
		},
		"multiple output msgs": {
			srcRoute: sdk.NewRoute(types.RouterKey, capturingRouteFn),
			srcEncoder: func(sender sdk.AccAddress, msg json.RawMessage) ([]sdk.Msg, error) {
				first := &types.MsgExecuteContract{
					Sender:   myContractAddr.String(),
					Contract: RandomBech32AccountAddress(t),
					Msg:      []byte("{}"),
				}
				second := &types.MsgExecuteContract{
					Sender:   myContractAddr.String(),
					Contract: RandomBech32AccountAddress(t),
					Msg:      []byte("{}"),
				}
				return []sdk.Msg{first, second}, nil
			},
			expMsgDispatched: 2,
		},
		"invalid sdk message rejected": {
			srcRoute: sdk.NewRoute(types.RouterKey, capturingRouteFn),
			srcEncoder: func(sender sdk.AccAddress, msg json.RawMessage) ([]sdk.Msg, error) {
				invalidMsg := types.MsgExecuteContract{
					Sender:   myContractAddr.String(),
					Contract: RandomBech32AccountAddress(t),
					Msg:      []byte("INVALID_JSON"),
				}
				return []sdk.Msg{&invalidMsg}, nil
			},
			expErr: types.ErrInvalid,
		},
		"invalid sender rejected": {
			srcRoute: sdk.NewRoute(types.RouterKey, capturingRouteFn),
			srcEncoder: func(sender sdk.AccAddress, msg json.RawMessage) ([]sdk.Msg, error) {
				invalidMsg := types.MsgExecuteContract{
					Sender:   RandomBech32AccountAddress(t),
					Contract: RandomBech32AccountAddress(t),
					Msg:      []byte("{}"),
				}
				return []sdk.Msg{&invalidMsg}, nil
			},
			expErr: sdkerrors.ErrUnauthorized,
		},
		"unroutable message rejected": {
			srcRoute: sdk.NewRoute("nothing", capturingRouteFn),
			srcEncoder: func(sender sdk.AccAddress, msg json.RawMessage) ([]sdk.Msg, error) {
				myMsg := types.MsgExecuteContract{
					Sender:   myContractAddr.String(),
					Contract: RandomBech32AccountAddress(t),
					Msg:      []byte("{}"),
				}
				return []sdk.Msg{&myMsg}, nil
			},
			expErr: sdkerrors.ErrUnknownRequest,
		},
		"encoding error passed": {
			srcRoute: sdk.NewRoute("nothing", capturingRouteFn),
			srcEncoder: func(sender sdk.AccAddress, msg json.RawMessage) ([]sdk.Msg, error) {
				myErr := types.ErrUnpinContractFailed
				return nil, myErr
			},
			expErr: types.ErrUnpinContractFailed,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotMsg = make([]sdk.Msg, 0)
			router := baseapp.NewRouter()
			router.AddRoute(spec.srcRoute)

			// when
			ctx := sdk.Context{}
			h := NewSDKMessageHandler(router, MessageEncoders{Custom: spec.srcEncoder})
			gotEvents, gotData, gotErr := h.DispatchMsg(ctx, myContractAddr, "myPort", myContractMessage)

			// then
			require.True(t, spec.expErr.Is(gotErr), "exp %v but got %#+v", spec.expErr, gotErr)
			if spec.expErr != nil {
				require.Len(t, gotMsg, 0)
				return
			}
			assert.Len(t, gotMsg, spec.expMsgDispatched)
			for i := 0; i < spec.expMsgDispatched; i++ {
				assert.Equal(t, myEvent, gotEvents[i])
				assert.Equal(t, []byte(myData), gotData[i])
			}
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
