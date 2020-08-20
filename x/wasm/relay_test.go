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
	//err = coordinator.RelayPacket(chainA, chainB, clientA, clientB, packet, ack)
	require.NoError(t, err)
	newBalance := chainA.App.BankKeeper.GetBalance(chainA.GetContext(), chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
	assert.Equal(t, originalBalance.Sub(coinToSendToB), newBalance)
	const ibcVoucherTicker = "ibc/310F9D708E5AA2F54CA83BC04C2E56F1EA62DB6FBDA321B337867CF5BEECF531"
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
	)
	clientA, clientB, connA, connB := coordinator.SetupClientConnections(chainA, chainB, clientexported.Tendermint)
	// a channel for transfer to transfer
	transChanA, transChanB := coordinator.CreateTransferChannels(chainA, chainB, connA, connB, channeltypes.UNORDERED)
	myContract.transferChannelID = transChanA.ID

	originalBalance := chainA.App.BankKeeper.GetBalance(chainA.GetContext(), myContractAddr, sdk.DefaultBondDenom)
	require.Equal(t, ibc_testing.TestCoin, originalBalance, "exp %q but got %q", ibc_testing.TestCoin, originalBalance)

	// a channel for contranct to transfer
	channelA, channelB := coordinator.CreateChannel(chainA, chainB, connA, connB, sourcePortID, counterpartPortID, channeltypes.UNORDERED)

	_ = transChanB
	_, _, _ = clientA, channelB, originalBalance
	_ = clientB
	_ = channelA
	newBalance := chainA.App.BankKeeper.GetBalance(chainA.GetContext(), myContractAddr, sdk.DefaultBondDenom)
	assert.Equal(t, originalBalance.Sub(coinToSendToB).String(), newBalance.String())
}

type senderContract struct {
	t                 *testing.T
	contractAddr      sdk.AccAddress
	chain             *ibc_testing.TestChain
	receiverAddr      sdk.AccAddress
	coinsToSend       sdk.Coin
	transferChannelID string
}

func (s *senderContract) OnIBCChannelOpen(hash []byte, params cosmwasmv2.Env, order channeltypes.Order, version string, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelOpenResponse, uint64, error) {
	return &cosmwasmv2.IBCChannelOpenResponse{Result: true}, 0, nil
}
func (s *senderContract) OnIBCChannelConnect(hash []byte, params cosmwasmv2.Env, counterpartyPortID string, counterpartyChannelID string, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelConnectResponse, uint64, error) {
	// abusing onConnect event to send the message. can be any execute event which is not mocked though
	//todo: better demo would be to use querier to find the transfer port

	msg := ibctransfertypes.NewMsgTransfer("transfer", s.transferChannelID, s.coinsToSend, s.contractAddr, s.receiverAddr.String(), 110, 0)
	return &cosmwasmv2.IBCChannelConnectResponse{Messages: []sdk.Msg{msg}}, 0, nil
}

func (s *senderContract) OnIBCPacketReceive(hash []byte, params cosmwasmv2.Env, msg []byte, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketReceiveResponse, uint64, error) {
	panic("implement me")
}

func (s *senderContract) OnIBCPacketAcknowledgement(hash []byte, params cosmwasmv2.Env, originalData []byte, acknowledgement []byte, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketAcknowledgementResponse, uint64, error) {
	panic("implement me")
}

func (s *senderContract) OnIBCPacketTimeout(hash []byte, params cosmwasmv2.Env, msg []byte, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketTimeoutResponse, uint64, error) {
	panic("implement me")
}

func (s *senderContract) OnIBCChannelClose(ctx sdk.Context, hash []byte, params cosmwasmv2.Env, counterpartyPortID, counterpartyChannelID string, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelCloseResponse, uint64, error) {
	panic("implement me")
}

type receiverContract struct {
	t            *testing.T
	contractAddr sdk.AccAddress
	chain        *ibc_testing.TestChain
}

func (c *receiverContract) OnIBCChannelOpen(hash []byte, params cosmwasmv2.Env, order channeltypes.Order, version string, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelOpenResponse, uint64, error) {
	//if order != channeltypes.ORDERED { // todo: ordered channels fail with `k.GetNextSequenceAck` as there is no value for destPort/ DestChannel stored
	//	return &cosmwasmv2.IBCChannelOpenResponse{
	//		Result: false,
	//		Reason: "channel type must be ordered",
	//	}, 0, nil
	//}
	return &cosmwasmv2.IBCChannelOpenResponse{Result: true}, 0, nil
}
func (c *receiverContract) OnIBCPacketReceive(hash []byte, params cosmwasmv2.Env, msg []byte, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketReceiveResponse, uint64, error) {
	var src ibctransfertypes.FungibleTokenPacketData
	if err := ibctransfertypes.ModuleCdc.UnmarshalJSON(msg, &src); err != nil {
		return nil, 0, err
	}
	// call original ibctransfer keeper to not copy all code into this
	packet := params.IBC.AsPacket(msg)
	ctx := c.chain.GetContext() // HACK: please note that this is not reverted after checkTX
	err := c.chain.App.TransferKeeper.OnRecvPacket(ctx, packet, src)
	if err != nil {
		return nil, 0, sdkerrors.Wrap(err, "within our smart contract")
	}

	log := []wasmTypes.LogAttribute{} // note: all events are under `wasm` event type
	myAck := ibctransfertypes.FungibleTokenPacketAcknowledgement{Success: true}.GetBytes()
	return &cosmwasmv2.IBCPacketReceiveResponse{Acknowledgement: myAck, Log: log}, 0, nil
}
func (c *receiverContract) OnIBCPacketAcknowledgement(hash []byte, params cosmwasmv2.Env, originalData []byte, acknowledgement []byte, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketAcknowledgementResponse, uint64, error) {
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
	ctx := c.chain.GetContext() // HACK: please note that this is not reverted after checkTX
	err := c.chain.App.TransferKeeper.OnAcknowledgementPacket(ctx, packet, src, ack)
	if err != nil {
		return nil, 0, sdkerrors.Wrap(err, "within our smart contract")
	}

	return &cosmwasmv2.IBCPacketAcknowledgementResponse{}, 0, nil
}

func (c *receiverContract) OnIBCPacketTimeout(hash []byte, params cosmwasmv2.Env, originalData []byte, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketTimeoutResponse, uint64, error) {
	var src ibctransfertypes.FungibleTokenPacketData
	if err := ibctransfertypes.ModuleCdc.UnmarshalJSON(originalData, &src); err != nil {
		return nil, 0, err
	}
	// call original ibctransfer keeper to not copy all code into this
	packet := params.IBC.AsPacket(originalData)

	// call original ibctransfer keeper to not copy all code into this
	ctx := c.chain.GetContext() // HACK: please note that this is not reverted after checkTX
	err := c.chain.App.TransferKeeper.OnTimeoutPacket(ctx, packet, src)
	if err != nil {
		return nil, 0, sdkerrors.Wrap(err, "within our smart contract")
	}

	return &cosmwasmv2.IBCPacketTimeoutResponse{}, 0, nil
}
func (s *receiverContract) OnIBCChannelConnect(hash []byte, params cosmwasmv2.Env, counterpartyPortID string, counterpartyChannelID string, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelConnectResponse, uint64, error) {
	return &cosmwasmv2.IBCChannelConnectResponse{}, 0, nil
}
func (s *receiverContract) OnIBCChannelClose(ctx sdk.Context, hash []byte, params cosmwasmv2.Env, counterpartyPortID, counterpartyChannelID string, store prefix.Store, api cosmwasmv1.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelCloseResponse, uint64, error) {
	panic("implement me")
}
