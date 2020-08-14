package wasm_test

import (
	"bytes"
	"testing"

	cosmwasmv1 "github.com/CosmWasm/go-cosmwasm"
	"github.com/CosmWasm/wasmd/x/wasm/ibc_testing"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/internal/keeper"
	cosmwasmv2 "github.com/CosmWasm/wasmd/x/wasm/internal/keeper/cosmwasm"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctransfertypes "github.com/cosmos/cosmos-sdk/x/ibc-transfer/types"
	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromIBCTransferToContract(t *testing.T) {
	var (
		coordinator = ibc_testing.NewCoordinator(t, 2)
		chainA      = coordinator.GetChain(ibc_testing.GetChainID(0))
		chainB      = coordinator.GetChain(ibc_testing.GetChainID(1))
	)
	myContractAddr := chainA.NewRandomContractInstance()
	wasmkeeper.MockContracts[myContractAddr.String()] = &myContractA{t: t, contractAddr: myContractAddr}

	contractAPortID := chainA.ContractInfo(myContractAddr).IBCPortID

	var (
		counterpartPortID = "transfer"
		sourcePortID      = contractAPortID
	)
	clientA, clientB, connA, connB := coordinator.SetupClientConnections(chainA, chainB, clientexported.Tendermint)

	channelA, channelB := coordinator.CreateChannel(chainA, chainB, connA, connB, sourcePortID, counterpartPortID, channeltypes.ORDERED)

	//_, _, err := coordinator.ChanOpenInit(chainA, chainB, connA, connB, sourcePortID, counterpartPortID, channeltypes.UNORDERED)
	//require.Error(t, err) //  can not test this as it fails in SignCheckDeliver with the test assertions there

	// open channel with our countract taking part in the handshake
	//channelA, channelB, err := coordinator.ChanOpenInit(chainA, chainB, connA, connB, sourcePortID, counterpartPortID, channeltypes.ORDERED)
	//require.NoError(t, err)
	//err = coordinator.UpdateClient(chainA, chainB,connA.ClientID, clientexported.Tendermint)
	//require.NoError(t, err)
	//
	//err = coordinator.ChanOpenAck(chainA, chainB, channelA, channelB)
	//require.NoError(t, err)
	//err = coordinator.ChanOpenConfirm(chainB, chainA, channelB, channelA)
	//require.NoError(t, err)

	// with the channels established, let's do a transfer
	coinToSendToA := ibc_testing.TestCoin
	msg := ibctransfertypes.NewMsgTransfer(channelB.PortID, channelB.ID, coinToSendToA, chainB.SenderAccount.GetAddress(), chainA.SenderAccount.GetAddress().String(), 110, 0)
	err := coordinator.SendMsgs(chainB, chainA, clientA, msg)
	require.NoError(t, err)


	t.Skip("debug failure in relay call")
	fungibleTokenPacket := ibctransfertypes.NewFungibleTokenPacketData(coinToSendToA.Denom, coinToSendToA.Amount.Uint64(), chainB.SenderAccount.GetAddress().String(), chainA.SenderAccount.GetAddress().String())
	packet := channeltypes.NewPacket(fungibleTokenPacket.GetBytes(), 1, channelB.PortID, channelB.ID, channelA.PortID, channelA.ID, 110, 0)
	ack := ibctransfertypes.FungibleTokenPacketAcknowledgement{Success: true}
	err = coordinator.RelayPacket(chainB, chainA, clientB, clientA, packet, ack.GetBytes())
	require.NoError(t, err)
}

type myContractA struct {
	t            *testing.T
	contractAddr sdk.AccAddress
}

func (c *myContractA) AcceptChannel(hash []byte, params cosmwasmv2.Env, order channeltypes.Order, version string, connectionHops []string, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.AcceptChannelResponse, uint64, error) {
	if order != channeltypes.ORDERED {
		return &cosmwasmv2.AcceptChannelResponse{
			Result: false,
			Reason: "channel type must be ordered",
		}, 0, nil
	}
	return &cosmwasmv2.AcceptChannelResponse{Result: true}, 0, nil
}
func (c *myContractA) OnReceive(hash []byte, params cosmwasmv2.Env, msg []byte, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.OnReceiveIBCResponse, uint64, error) {
	myAck := ibctransfertypes.FungibleTokenPacketAcknowledgement{Success: true}.GetBytes()
	return &cosmwasmv2.OnReceiveIBCResponse{Acknowledgement: myAck}, 0, nil
}
func (c *myContractA) OnAcknowledgement(hash []byte, params cosmwasmv2.Env, originalData []byte, acknowledgement []byte, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.OnAcknowledgeIBCResponse, uint64, error) {
	//state := store.Get(hash)
	//require.NotNil(c.t, state)
	//assert.Equal(c.t, state, append(originalData, acknowledgement...))
	return &cosmwasmv2.OnAcknowledgeIBCResponse{}, 0, nil
}

func (c *myContractA) OnTimeout(hash []byte, params cosmwasmv2.Env, originalData []byte, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.OnTimeoutIBCResponse, uint64, error) {
	state := store.Get(hash)
	require.NotNil(c.t, state)
	assert.True(c.t, bytes.HasPrefix(state, originalData))
	return &cosmwasmv2.OnTimeoutIBCResponse{}, 0, nil
}
