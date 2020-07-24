package keeper_test

import (
	"testing"

	cosmwasm "github.com/CosmWasm/go-cosmwasm"
	wasmTypes "github.com/CosmWasm/go-cosmwasm/types"
	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/CosmWasm/wasmd/x/wasm/internal/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	channelexported "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/exported"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	"github.com/stretchr/testify/require"
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

func (c *mockContract) OnReceive(hash []byte, params wasmTypes.Env, msg []byte, store prefix.Store, api cosmwasm.GoAPI, querier keeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*keeper.OnReceiveIBCResponse, uint64, error) {
	panic("mockContract called: implement me")
}

func (c mockContract) DoIBCCall(ctx sdk.Context, channel string, id string) { // todo: move somewhere else
	msg := types.MsgWasmIBCCall{
		SourcePort:       id,
		SourceChannel:    channel,
		Sender:           c.contractAddr,
		DestContractAddr: nil, // remove
		TimeoutHeight:    110,
		TimeoutTimestamp: 0,
		Msg:              []byte("{}"),
	}
	handler := wasm.NewHandler(c.app.WasmKeeper)
	_, err := handler(ctx, &msg)
	require.NoError(c.t, err)
}

const (
	counterpartyPortID       = "otherPortID"
	counterpartyChannelID    = "otherChannelID"
	connectionID             = "myConnectionID"
	channelID                = "myChannelID"
	protocolVersion          = "1.0"
	clientID                 = "myClientID"
	counterpartyClientID     = "otherClientID"
	counterpartyConnectionID = "otherConnectionID"
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
