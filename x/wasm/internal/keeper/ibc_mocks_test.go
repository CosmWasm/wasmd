package keeper_test

import (
	"bytes"
	"testing"

	cosmwasmv1 "github.com/CosmWasm/go-cosmwasm"
	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/CosmWasm/wasmd/x/wasm/internal/keeper"
	cosmwasmv2 "github.com/CosmWasm/wasmd/x/wasm/internal/keeper/cosmwasm"
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	channelexported "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/exported"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/rand"
)

type mockContract struct {
	app          *app.WasmApp
	t            *testing.T
	contractAddr sdk.AccAddress
}

func MockContract(t *testing.T, contractAddr sdk.AccAddress, app *app.WasmApp) *mockContract {
	c := &mockContract{t: t,
		contractAddr: contractAddr,
		app:          app,
	}
	keeper.MockContracts[contractAddr.String()] = c
	return c
}

func (c *mockContract) AcceptChannel(hash []byte, params cosmwasmv2.Env, order channeltypes.Order, version string, connectionHops []string, store prefix.Store, api cosmwasmv1.GoAPI, querier keeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.AcceptChannelResponse, uint64, error) {
	if order != channeltypes.ORDERED {
		return &cosmwasmv2.AcceptChannelResponse{
			Result: false,
			Reason: "channel type must be ordered",
		}, 0, nil
	}
	return &cosmwasmv2.AcceptChannelResponse{Result: true}, 0, nil
}
func (c *mockContract) OnReceive(hash []byte, params cosmwasmv2.Env, msg []byte, store prefix.Store, api cosmwasmv1.GoAPI, querier keeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.OnReceiveIBCResponse, uint64, error) {
	// real contract would do something with incoming msg
	// create some random ackknowledgement
	myAck := rand.Bytes(25)
	store.Set(hash, append(msg, myAck...))
	return &cosmwasmv2.OnReceiveIBCResponse{Acknowledgement: myAck}, 0, nil
}
func (c *mockContract) OnAcknowledgement(hash []byte, params cosmwasmv2.Env, originalData []byte, acknowledgement []byte, store prefix.Store, api cosmwasmv1.GoAPI, querier keeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.OnAcknowledgeIBCResponse, uint64, error) {
	state := store.Get(hash)
	require.NotNil(c.t, state)
	assert.Equal(c.t, state, append(originalData, acknowledgement...))
	return &cosmwasmv2.OnAcknowledgeIBCResponse{}, 0, nil
}

func (c *mockContract) OnTimeout(hash []byte, params cosmwasmv2.Env, originalData []byte, store prefix.Store, api cosmwasmv1.GoAPI, querier keeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.OnTimeoutIBCResponse, uint64, error) {
	state := store.Get(hash)
	require.NotNil(c.t, state)
	assert.True(c.t, bytes.HasPrefix(state, originalData))
	return &cosmwasmv2.OnTimeoutIBCResponse{}, 0, nil
}

func (c mockContract) DoIBCCall(ctx sdk.Context, sourceChannelID string, sourcePortID string) { // todo: move somewhere else
	// how does contract know where to send package too? channel/portID
	// can environment have a list of available channel/portIDs or do we check later?
	// alternative: query for ibc channels/ portids?

	msg := types.MsgWasmIBCCall{
		SourcePort:       sourcePortID,
		SourceChannel:    sourceChannelID,
		Sender:           c.contractAddr,
		TimeoutHeight:    110,
		TimeoutTimestamp: 0,
		Msg:              []byte("{}"),
	}
	handler := wasm.NewHandler(c.app.WasmKeeper)
	_, err := handler(ctx, &msg)
	require.NoError(c.t, err)
}

const (
	protocolVersion          = "1.0"
	counterpartyPortID       = "otherPortID"
	counterpartyChannelID    = "otherChannelID"
	counterpartyConnectionID = "otherConnectionID"
	counterpartyClientID     = "otherClientID"
	channelID                = "myChannelID"
	connectionID             = "myConnectionID"
	clientID                 = "myClientID"
)

type receivedPackets struct {
	channelCap *capabilitytypes.Capability
	packet     channelexported.PacketI
}

type mockChannelKeeper struct {
	seq      uint64
	k        types.ChannelKeeper
	received []receivedPackets
}

func NewMockChannelKeeper(k types.ChannelKeeper) *mockChannelKeeper {
	return &mockChannelKeeper{k: k}
}

func (m mockChannelKeeper) GetChannel(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool) {
	counterpartyChannel := channeltypes.NewCounterparty(counterpartyPortID, counterpartyChannelID)
	return channeltypes.NewChannel(channeltypes.OPEN, channeltypes.ORDERED, counterpartyChannel,
		[]string{connectionID}, protocolVersion,
	), true
}

func (m *mockChannelKeeper) GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool) {
	m.seq++
	return m.seq, true
}

func (m *mockChannelKeeper) SendPacket(ctx sdk.Context, channelCap *capabilitytypes.Capability, packet channelexported.PacketI) error {
	m.received = append(m.received, receivedPackets{
		channelCap: channelCap,
		packet:     packet,
	})
	return nil
}

func (m mockChannelKeeper) PacketExecuted(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet channelexported.PacketI, acknowledgement []byte) error {
	panic("implement me")
}

func (m mockChannelKeeper) ChanCloseInit(ctx sdk.Context, portID, channelID string, chanCap *capabilitytypes.Capability) error {
	panic("implement me")
}

func (m *mockChannelKeeper) Reset() {
	m.seq = 0
	m.received = nil
}
