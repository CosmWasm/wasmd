package wasm_test

import (
	"testing"

	cosmwasmv1 "github.com/CosmWasm/go-cosmwasm"
	wasmTypes "github.com/CosmWasm/go-cosmwasm/types"
	"github.com/CosmWasm/wasmd/x/wasm/ibc_testing"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/internal/keeper"
	cosmwasmv2 "github.com/CosmWasm/wasmd/x/wasm/internal/keeper/cosmwasm"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
	myContractAddr := chainB.NewRandomContractInstance()
	wasmkeeper.MockContracts[myContractAddr.String()] = &myContractA{t: t, contractAddr: myContractAddr, chain: chainB}

	contractAPortID := chainB.ContractInfo(myContractAddr).IBCPortID

	var (
		sourcePortID      = "transfer"
		counterpartPortID = contractAPortID
	)
	clientA, clientB, connA, connB := coordinator.SetupClientConnections(chainA, chainB, clientexported.Tendermint)

	channelA, channelB := coordinator.CreateChannel(chainA, chainB, connA, connB, sourcePortID, counterpartPortID, channeltypes.UNORDERED)

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

	originalBalance := chainA.App.BankKeeper.GetBalance(chainA.GetContext(), chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

	// with the channels established, let's do a transfer
	coinToSendToB := ibc_testing.TestCoin
	msg := ibctransfertypes.NewMsgTransfer(channelA.PortID, channelA.ID, coinToSendToB, chainA.SenderAccount.GetAddress(), chainB.SenderAccount.GetAddress().String(), 110, 0)
	err := coordinator.SendMsgs(chainA, chainB, clientB, msg)
	require.NoError(t, err)

	fungibleTokenPacket := ibctransfertypes.NewFungibleTokenPacketData(coinToSendToB.Denom, coinToSendToB.Amount.Uint64(), chainA.SenderAccount.GetAddress().String(), chainB.SenderAccount.GetAddress().String())
	packet := channeltypes.NewPacket(fungibleTokenPacket.GetBytes(), 1, channelA.PortID, channelA.ID, channelB.PortID, channelB.ID, 110, 0)
	err = coordinator.RecvPacket(chainA, chainB, clientA, packet) //sent to chainB
	require.NoError(t, err)

	ack := ibctransfertypes.FungibleTokenPacketAcknowledgement{Success: true}.GetBytes()

	err = coordinator.AcknowledgePacket(chainA, chainB, clientB, packet, ack) // sent to chainA
	//err = coordinator.RelayPacket(chainA, chainB, clientA, clientB, packet, ack)
	require.NoError(t, err)
	newBalance := chainA.App.BankKeeper.GetBalance(chainA.GetContext(), chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	assert.Equal(t, originalBalance.Sub(coinToSendToB), newBalance)
	const ibcVoucherTicker = "ibc/310F9D708E5AA2F54CA83BC04C2E56F1EA62DB6FBDA321B337867CF5BEECF531"
	chainBBalance := chainB.App.BankKeeper.GetBalance(chainB.GetContext(), chainB.SenderAccount.GetAddress(), ibcVoucherTicker)
	assert.Equal(t, sdk.Coin{Denom: ibcVoucherTicker, Amount: coinToSendToB.Amount}, chainBBalance, chainB.App.BankKeeper.GetAllBalances(chainB.GetContext(), chainB.SenderAccount.GetAddress()))
}

type myContractA struct {
	t            *testing.T
	contractAddr sdk.AccAddress
	chain        *ibc_testing.TestChain
}

func (c *myContractA) AcceptChannel(hash []byte, params cosmwasmv2.Env, order channeltypes.Order, version string, connectionHops []string, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.AcceptChannelResponse, uint64, error) {
	//if order != channeltypes.ORDERED { // todo: ordered channels fail with `k.GetNextSequenceAck` as there is no value for destPort/ DestChannel stored
	//	return &cosmwasmv2.AcceptChannelResponse{
	//		Result: false,
	//		Reason: "channel type must be ordered",
	//	}, 0, nil
	//}
	return &cosmwasmv2.AcceptChannelResponse{Result: true}, 0, nil
}
func (c *myContractA) OnReceive(ctx sdk.Context, hash []byte, params cosmwasmv2.Env, msg []byte, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.OnReceiveIBCResponse, uint64, error) {
	var src ibctransfertypes.FungibleTokenPacketData
	if err := ibctransfertypes.ModuleCdc.UnmarshalJSON(msg, &src); err != nil {
		return nil, 0, err
	}
	// call original ibctransfer keeper to not copy all code into this
	packet := params.IBC.AsPacket(msg)
	err := c.chain.App.TransferKeeper.OnRecvPacket(ctx, packet, src)
	if err != nil {
		return nil, 0, sdkerrors.Wrap(err, "within our smart contract")
	}

	log := []wasmTypes.LogAttribute{} // note: all events are under `wasm` event type
	myAck := ibctransfertypes.FungibleTokenPacketAcknowledgement{Success: true}.GetBytes()
	return &cosmwasmv2.OnReceiveIBCResponse{Acknowledgement: myAck, Log: log}, 0, nil
}
func (c *myContractA) OnAcknowledgement(ctx sdk.Context, hash []byte, params cosmwasmv2.Env, originalData []byte, acknowledgement []byte, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.OnAcknowledgeIBCResponse, uint64, error) {
	var src ibctransfertypes.FungibleTokenPacketData
	if err := ibctransfertypes.ModuleCdc.UnmarshalJSON(originalData, &src); err != nil {
		return nil, 0, err
	}
	// call original ibctransfer keeper to not copy all code into this
	packet := params.IBC.AsPacket(originalData)

	var ack ibctransfertypes.FungibleTokenPacketAcknowledgement
	if err := ibctransfertypes.ModuleCdc.UnmarshalJSON(acknowledgement, &src); err != nil {
		return nil, 0, err
	}
	// call original ibctransfer keeper to not copy all code into this
	err := c.chain.App.TransferKeeper.OnAcknowledgementPacket(ctx, packet, src, ack)
	if err != nil {
		return nil, 0, sdkerrors.Wrap(err, "within our smart contract")
	}

	return &cosmwasmv2.OnAcknowledgeIBCResponse{}, 0, nil
}

func (c *myContractA) OnTimeout(ctx sdk.Context, hash []byte, params cosmwasmv2.Env, originalData []byte, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.OnTimeoutIBCResponse, uint64, error) {
	var src ibctransfertypes.FungibleTokenPacketData
	if err := ibctransfertypes.ModuleCdc.UnmarshalJSON(originalData, &src); err != nil {
		return nil, 0, err
	}
	// call original ibctransfer keeper to not copy all code into this
	packet := params.IBC.AsPacket(originalData)

	// call original ibctransfer keeper to not copy all code into this
	err := c.chain.App.TransferKeeper.OnTimeoutPacket(ctx, packet, src)
	if err != nil {
		return nil, 0, sdkerrors.Wrap(err, "within our smart contract")
	}

	return &cosmwasmv2.OnTimeoutIBCResponse{}, 0, nil
}
