package wasm_test

import (
	"testing"

	"github.com/CosmWasm/go-cosmwasm"
	cosmwasmv1 "github.com/CosmWasm/go-cosmwasm/types"
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
	// scenario: a contract can handle the receiving side of a ibc transfer
	var (
		coordinator = ibc_testing.NewCoordinator(t, 2)
		chainA      = coordinator.GetChain(ibc_testing.GetChainID(0))
		chainB      = coordinator.GetChain(ibc_testing.GetChainID(1))
	)
	myContractAddr := chainB.NewRandomContractInstance()
	wasmkeeper.MockContracts[myContractAddr.String()] = &receiverContract{t: t, contractAddr: myContractAddr, chain: chainB}

	contractAPortID := chainB.ContractInfo(myContractAddr).IBCPortID

	var (
		sourcePortID      = "transfer"
		counterpartPortID = contractAPortID
	)
	clientA, clientB, connA, connB := coordinator.SetupClientConnections(chainA, chainB, clientexported.Tendermint)
	channelA, channelB := coordinator.CreateChannel(chainA, chainB, connA, connB, sourcePortID, counterpartPortID, channeltypes.UNORDERED)

	originalBalance := chainA.App.BankKeeper.GetBalance(chainA.GetContext(), chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)

	// with the channels established, let's do a transfer
	coinToSendToB := sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1))
	msg := ibctransfertypes.NewMsgTransfer(channelA.PortID, channelA.ID, coinToSendToB, chainA.SenderAccount.GetAddress(), chainB.SenderAccount.GetAddress().String(), 110, 0)
	err := coordinator.SendMsgs(chainA, chainB, clientB, msg)
	require.NoError(t, err)

	fungibleTokenPacket := ibctransfertypes.NewFungibleTokenPacketData(coinToSendToB.Denom, coinToSendToB.Amount.Uint64(), chainA.SenderAccount.GetAddress().String(), chainB.SenderAccount.GetAddress().String())
	packet := channeltypes.NewPacket(fungibleTokenPacket.GetBytes(), 1, channelA.PortID, channelA.ID, channelB.PortID, channelB.ID, 110, 0)
	err = coordinator.RecvPacket(chainA, chainB, clientA, packet) //sent to chainB
	require.NoError(t, err)

	ack := ibctransfertypes.FungibleTokenPacketAcknowledgement{Success: true}.GetBytes()

	err = coordinator.AcknowledgePacket(chainA, chainB, clientB, packet, ack) // sent to chainA
	require.NoError(t, err)
	newBalance := chainA.App.BankKeeper.GetBalance(chainA.GetContext(), chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	assert.Equal(t, originalBalance.Sub(coinToSendToB), newBalance)
	const ibcVoucherTicker = "ibc/1AAD10C9C252ACF464C7167E328C866BBDA0BDED3D89EFAB7B7C30BF01DE4657"
	chainBBalance := chainB.App.BankKeeper.GetBalance(chainB.GetContext(), chainB.SenderAccount.GetAddress(), ibcVoucherTicker)
	// note: the contract is called during check and deliverTX but the context used in the contract does not rollback
	// so that we got twice the amount
	assert.Equal(t, sdk.Coin{Denom: ibcVoucherTicker, Amount: coinToSendToB.Amount.Mul(sdk.NewInt(2))}.String(), chainBBalance.String(), chainB.App.BankKeeper.GetAllBalances(chainB.GetContext(), chainB.SenderAccount.GetAddress()))
}

func TestContractCanInitiateIBCTransfer(t *testing.T) {
	// scenario: a contract can start an ibc transfer via ibctransfertypes.NewMsgTransfer
	// on an existing connection
	var (
		coordinator   = ibc_testing.NewCoordinator(t, 2)
		chainA        = coordinator.GetChain(ibc_testing.GetChainID(0))
		chainB        = coordinator.GetChain(ibc_testing.GetChainID(1))
		coinToSendToB = ibc_testing.TestCoin
	)
	myContractAddr := chainA.NewRandomContractInstance()
	myContract := &senderContract{t: t, contractAddr: myContractAddr, chain: chainA, receiverAddr: chainB.SenderAccount.GetAddress(), coinsToSend: coinToSendToB}
	wasmkeeper.MockContracts[myContractAddr.String()] = myContract

	contractAPortID := chainA.ContractInfo(myContractAddr).IBCPortID

	var (
		sourcePortID      = contractAPortID
		counterpartPortID = "transfer"
		ibcVoucherTicker  = "ibc/8D5B148875A26426899137B476C646A94652D73BAEEE3CD30B9C261EB7BC0E1B"
	)
	clientA, clientB, connA, connB := coordinator.SetupClientConnections(chainA, chainB, clientexported.Tendermint)
	channelA, channelB := coordinator.CreateChannel(chainA, chainB, connA, connB, sourcePortID, counterpartPortID, channeltypes.UNORDERED)

	// send to chain B
	err := coordinator.UpdateClient(chainB, chainA, clientB, clientexported.Tendermint)
	require.NoError(t, err)

	packet := channeltypes.NewPacket(myContract.packetSent.GetBytes(), 1, channelA.PortID, channelA.ID, channelB.PortID, channelB.ID, 110, 0)
	err = coordinator.RecvPacket(chainA, chainB, clientA, packet) //sent to chainB
	require.NoError(t, err)

	// send Ack to chain A
	ack := ibctransfertypes.FungibleTokenPacketAcknowledgement{Success: true}.GetBytes()
	err = coordinator.AcknowledgePacket(chainA, chainB, clientB, packet, ack) // sent to chainA
	require.NoError(t, err)

	newBalance := chainB.App.BankKeeper.GetBalance(chainB.GetContext(), chainB.SenderAccount.GetAddress(), ibcVoucherTicker)
	assert.Equal(t, sdk.NewCoin(ibcVoucherTicker, coinToSendToB.Amount).String(), newBalance.String(), chainB.App.BankKeeper.GetAllBalances(chainB.GetContext(), chainB.SenderAccount.GetAddress()))
}

type senderContract struct {
	t            *testing.T
	contractAddr sdk.AccAddress
	chain        *ibc_testing.TestChain
	receiverAddr sdk.AccAddress
	coinsToSend  sdk.Coin
	packetSent   *ibctransfertypes.FungibleTokenPacketData
}

func (s *senderContract) OnIBCChannelOpen(hash []byte, params cosmwasmv2.Env, channel cosmwasmv2.IBCChannel, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelOpenResponse, uint64, error) {
	return &cosmwasmv2.IBCChannelOpenResponse{Success: true}, 0, nil
}
func (s *senderContract) OnIBCChannelConnect(hash []byte, params cosmwasmv2.Env, channel cosmwasmv2.IBCChannel, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelConnectResponse, uint64, error) {
	// abusing onConnect event to send the message. can be any execute event which is not mocked though

	escrowAddress := ibctransfertypes.GetEscrowAddress(channel.Endpoint.Port, channel.Endpoint.Channel)
	sendToEscrowMsg := &cosmwasmv1.BankMsg{
		Send: &cosmwasmv1.SendMsg{
			FromAddress: s.contractAddr.String(),
			ToAddress:   escrowAddress.String(),
			Amount:      cosmwasmv1.Coins{cosmwasmv1.NewCoin(s.coinsToSend.Amount.Uint64(), s.coinsToSend.Denom)},
		}}

	dataPacket := ibctransfertypes.NewFungibleTokenPacketData(
		s.coinsToSend.Denom, s.coinsToSend.Amount.Uint64(), s.contractAddr.String(), s.receiverAddr.String(),
	)
	s.packetSent = &dataPacket
	ibcPacket := &cosmwasmv2.IBCMsg{
		SendPacket: &cosmwasmv2.IBCSendMsg{
			ChannelID:        channel.Endpoint.Channel,
			Data:             dataPacket.GetBytes(),
			TimeoutHeight:    110,
			TimeoutTimestamp: 0,
		},
	}
	return &cosmwasmv2.IBCChannelConnectResponse{Messages: []cosmwasmv2.CosmosMsg{{Bank: sendToEscrowMsg}, {IBC: ibcPacket}}}, 0, nil
}

func (s *senderContract) OnIBCChannelClose(ctx sdk.Context, hash []byte, params cosmwasmv2.Env, channel cosmwasmv2.IBCChannel, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelCloseResponse, uint64, error) {
	panic("implement me")
}

func (s *senderContract) OnIBCPacketReceive(hash []byte, params cosmwasmv2.Env, packet cosmwasmv2.IBCPacket, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketReceiveResponse, uint64, error) {
	panic("implement me")
}

func (s *senderContract) OnIBCPacketAcknowledgement(hash []byte, params cosmwasmv2.Env, packetAck cosmwasmv2.IBCAcknowledgement, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketAcknowledgementResponse, uint64, error) {
	return &cosmwasmv2.IBCPacketAcknowledgementResponse{}, 0, nil
}

func (s *senderContract) OnIBCPacketTimeout(hash []byte, params cosmwasmv2.Env, packet cosmwasmv2.IBCPacket, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketTimeoutResponse, uint64, error) {
	// return from escrow
	panic("implement me")
}

func (s *senderContract) Execute(hash []byte, params cosmwasmv1.Env, msg []byte, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.HandleResponse, uint64, error) {
	panic("implement me")
}

type receiverContract struct {
	t            *testing.T
	contractAddr sdk.AccAddress
	chain        *ibc_testing.TestChain
}

func (c *receiverContract) OnIBCChannelOpen(hash []byte, params cosmwasmv2.Env, channel cosmwasmv2.IBCChannel, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelOpenResponse, uint64, error) {
	//if order != channeltypes.ORDERED { // todo: ordered channels fail with `k.GetNextSequenceAck` as there is no value for destPort/ DestChannel stored
	//	return &cosmwasmv2.IBCChannelOpenResponse{
	//		Result: false,
	//		Reason: "channel type must be ordered",
	//	}, 0, nil
	//}
	return &cosmwasmv2.IBCChannelOpenResponse{Success: true}, 0, nil
}

func (c *receiverContract) OnIBCChannelConnect(hash []byte, params cosmwasmv2.Env, channel cosmwasmv2.IBCChannel, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelConnectResponse, uint64, error) {
	return &cosmwasmv2.IBCChannelConnectResponse{}, 0, nil
}

func (c *receiverContract) OnIBCChannelClose(ctx sdk.Context, hash []byte, params cosmwasmv2.Env, channel cosmwasmv2.IBCChannel, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelCloseResponse, uint64, error) {
	return &cosmwasmv2.IBCChannelCloseResponse{}, 0, nil
}

func (c *receiverContract) OnIBCPacketReceive(hash []byte, params cosmwasmv2.Env, packet cosmwasmv2.IBCPacket, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketReceiveResponse, uint64, error) {
	var src ibctransfertypes.FungibleTokenPacketData
	if err := ibctransfertypes.ModuleCdc.UnmarshalJSON(packet.Data, &src); err != nil {
		return nil, 0, err
	}
	// call original ibctransfer keeper to not copy all code into this
	ibcPacket := toIBCPacket(packet)
	ctx := c.chain.GetContext() // HACK: please note that this is not reverted after checkTX
	err := c.chain.App.TransferKeeper.OnRecvPacket(ctx, ibcPacket, src)
	if err != nil {
		return nil, 0, sdkerrors.Wrap(err, "within Our smart contract")
	}

	log := []cosmwasmv1.LogAttribute{} // note: all events are under `wasm` event type
	myAck := ibctransfertypes.FungibleTokenPacketAcknowledgement{Success: true}.GetBytes()
	return &cosmwasmv2.IBCPacketReceiveResponse{Acknowledgement: myAck, Log: log}, 0, nil
}

func (c *receiverContract) OnIBCPacketAcknowledgement(hash []byte, params cosmwasmv2.Env, packetAck cosmwasmv2.IBCAcknowledgement, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketAcknowledgementResponse, uint64, error) {
	var src ibctransfertypes.FungibleTokenPacketData
	if err := ibctransfertypes.ModuleCdc.UnmarshalJSON(packetAck.OriginalPacket.Data, &src); err != nil {
		return nil, 0, err
	}
	// call original ibctransfer keeper to not copy all code into this

	var ack ibctransfertypes.FungibleTokenPacketAcknowledgement
	if err := ibctransfertypes.ModuleCdc.UnmarshalJSON(packetAck.Acknowledgement, &ack); err != nil {
		return nil, 0, err
	}

	// call original ibctransfer keeper to not copy all code into this
	ctx := c.chain.GetContext() // HACK: please note that this is not reverted after checkTX
	ibcPacket := toIBCPacket(packetAck.OriginalPacket)
	err := c.chain.App.TransferKeeper.OnAcknowledgementPacket(ctx, ibcPacket, src, ack)
	if err != nil {
		return nil, 0, sdkerrors.Wrap(err, "within Our smart contract")
	}

	return &cosmwasmv2.IBCPacketAcknowledgementResponse{}, 0, nil
}

func (c *receiverContract) OnIBCPacketTimeout(hash []byte, params cosmwasmv2.Env, packet cosmwasmv2.IBCPacket, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketTimeoutResponse, uint64, error) {
	var src ibctransfertypes.FungibleTokenPacketData
	if err := ibctransfertypes.ModuleCdc.UnmarshalJSON(packet.Data, &src); err != nil {
		return nil, 0, err
	}
	// call original ibctransfer keeper to not copy all code into this
	ibcPacket := toIBCPacket(packet)

	// call original ibctransfer keeper to not copy all code into this
	ctx := c.chain.GetContext() // HACK: please note that this is not reverted after checkTX
	err := c.chain.App.TransferKeeper.OnTimeoutPacket(ctx, ibcPacket, src)
	if err != nil {
		return nil, 0, sdkerrors.Wrap(err, "within Our smart contract")
	}

	return &cosmwasmv2.IBCPacketTimeoutResponse{}, 0, nil
}

func (c *receiverContract) Execute(hash []byte, params cosmwasmv1.Env, msg []byte, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.HandleResponse, uint64, error) {
	panic("implement me")
}

func toIBCPacket(p cosmwasmv2.IBCPacket) channeltypes.Packet {
	return channeltypes.Packet{
		Sequence:           p.Sequence,
		SourcePort:         p.Source.Port,
		SourceChannel:      p.Source.Channel,
		DestinationPort:    p.Destination.Port,
		DestinationChannel: p.Destination.Channel,
		Data:               p.Data,
		TimeoutHeight:      p.TimeoutHeight,
		TimeoutTimestamp:   p.TimeoutTimestamp,
	}
}
