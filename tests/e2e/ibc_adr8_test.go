package e2e

import (
	"bytes"
	"testing"
	"time"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	hosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	ibcfee "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/xwasmvm"
	"github.com/CosmWasm/wasmd/x/xwasmvm/custom"

	"github.com/CosmWasm/wasmd/app"
	wasmibctesting "github.com/CosmWasm/wasmd/x/wasm/ibctesting"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestIBCAdr8WithTransfer(t *testing.T) {
	// scenario:
	// given 2 chains
	//   with an ics-20 channel established
	//   and a callback for a contract on chain A registered for ics-20 packet ack/ timeout
	// when the ack is relayed
	// then the callback is executed
	var contractAckCallbackCalled bool
	mockOpt := wasmkeeper.WithWasmEngineDecorator(func(old wasmtypes.WasmerEngine) wasmtypes.WasmerEngine {
		// hack to mock the new vm methods
		xvm, ok := old.(*xwasmvm.XVM)
		require.True(t, ok)
		xvm.OnIBCPacketAckedFn = func(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg xwasmvm.IBCPacketAckedMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
			contractAckCallbackCalled = true
			return &wasmvmtypes.IBCBasicResponse{}, 0, nil
		}
		return xvm
	})

	marshaler := app.MakeEncodingConfig().Marshaler
	coord := wasmibctesting.NewCoordinator(t, 2, []wasmkeeper.Option{mockOpt}) // add mock to chain A
	chainA := coord.GetChain(wasmibctesting.GetChainID(1))
	chainB := coord.GetChain(wasmibctesting.GetChainID(2))

	contractAddr := InstantiateReflectContract(t, chainA) // test with a real contract

	// actorChainA := contractAddr
	// actorChainB := sdk.AccAddress(chainB.SenderPrivKey.PubKey().Address())
	receiver := sdk.AccAddress(bytes.Repeat([]byte{1}, address.Len))
	// payee := sdk.AccAddress(bytes.Repeat([]byte{2}, address.Len))
	// oneToken := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1)))

	path := wasmibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
		PortID:  ibctransfertypes.PortID,
		Version: string(marshaler.MustMarshalJSON(&ibcfee.Metadata{FeeVersion: ibcfee.Version, AppVersion: ibctransfertypes.Version})),
		Order:   channeltypes.UNORDERED,
	}
	path.EndpointB.ChannelConfig = &ibctesting.ChannelConfig{
		PortID:  ibctransfertypes.PortID,
		Version: string(marshaler.MustMarshalJSON(&ibcfee.Metadata{FeeVersion: ibcfee.Version, AppVersion: ibctransfertypes.Version})),
		Order:   channeltypes.UNORDERED,
	}
	// with an ics-20 transfer channel setup between both chains
	coord.Setup(path)
	require.True(t, chainA.App.IBCFeeKeeper.IsFeeEnabled(chainA.GetContext(), ibctransfertypes.PortID, path.EndpointA.ChannelID))

	// when an ics20 transfer package is sent
	transferCoin := sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1))
	myIBCTransfer := wasmvmtypes.CosmosMsg{IBC: &wasmvmtypes.IBCMsg{Transfer: &wasmvmtypes.TransferMsg{
		ChannelID: path.EndpointA.ChannelID,
		ToAddress: receiver.String(),
		Amount:    wasmkeeper.ConvertSdkCoinToWasmCoin(transferCoin),
		Timeout:   wasmvmtypes.IBCTimeout{Block: &wasmvmtypes.IBCTimeoutBlock{Revision: 1, Height: 1_000}},
	}}}
	regMsg := custom.CustomADR8Msg{
		RegisterPacketProcessedCallback: &custom.RegisterPacketProcessedCallback{
			Sequence:            1, // hard code seq here as reflect contract does not read response from transfer msg
			ChannelID:           path.EndpointA.ChannelID,
			PortID:              ibctransfertypes.PortID,
			MaxCallbackGasLimit: 200_000,
		},
	}
	myCallbackRegistr := MustEncodeAsReflectCustomMessage(t, regMsg)
	MustExecViaReflectContract(t, chainA, contractAddr, myIBCTransfer, myCallbackRegistr)
	coord.CommitBlock(chainA, chainB)

	// and packages relayed
	require.NoError(t, coord.RelayAndAckPendingPackets(path))

	// then
	expBalance := ibctransfertypes.GetTransferCoin(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, transferCoin.Denom, transferCoin.Amount)
	gotBalance := chainB.Balance(receiver, expBalance.Denom)
	assert.Equal(t, expBalance.String(), gotBalance.String())

	// then
	assert.True(t, contractAckCallbackCalled, "contract callback was not executed")
}

func TestAdr8WithICA(t *testing.T) {
	// scenario:
	// given a host and controller chain
	//		 with a channel established to the host chain
	//       and an ICA account owned by a contract
	// when the contract sends an ICA tx and registers for a callback
	// then the contract should receive the callback on ack

	var contractAckCallbackCalled bool
	mockOpt := wasmkeeper.WithWasmEngineDecorator(func(old wasmtypes.WasmerEngine) wasmtypes.WasmerEngine {
		// hack to mock the new vm methods
		xvm, ok := old.(*xwasmvm.XVM)
		require.True(t, ok)
		xvm.OnIBCPacketAckedFn = func(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg xwasmvm.IBCPacketAckedMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
			contractAckCallbackCalled = true
			return &wasmvmtypes.IBCBasicResponse{}, 0, nil
		}
		return xvm
	})

	coord := wasmibctesting.NewCoordinator(t, 2, []wasmkeeper.Option{}, []wasmkeeper.Option{mockOpt}) // add mock to chain 2
	hostChain := coord.GetChain(ibctesting.GetChainID(1))
	hostParams := hosttypes.NewParams(true, []string{sdk.MsgTypeURL(&banktypes.MsgSend{})})
	hostChain.App.ICAHostKeeper.SetParams(hostChain.GetContext(), hostParams)

	controllerChain := coord.GetChain(ibctesting.GetChainID(2))
	contractAddr := InstantiateReflectContract(t, controllerChain) // test with a real contract

	path := wasmibctesting.NewPath(controllerChain, hostChain)
	coord.SetupConnections(path)

	// register contract for ICA address
	msg := icacontrollertypes.NewMsgRegisterInterchainAccount(path.EndpointA.ConnectionID, contractAddr.String(), "")
	res := MustExecViaStargateReflectContract(t, controllerChain, contractAddr, msg)
	chanID, portID, version := parseIBCChannelEvents(t, res)

	// next open channels on both sides
	path.EndpointA.ChannelID = chanID
	path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
		PortID:  portID,
		Version: version,
		Order:   channeltypes.ORDERED,
	}
	path.EndpointB.ChannelConfig = &ibctesting.ChannelConfig{
		PortID:  icatypes.HostPortID,
		Version: icatypes.Version,
		Order:   channeltypes.ORDERED,
	}
	coord.CreateChannels(path)

	// assert ICA exists on controller
	icaRsp, err := controllerChain.App.ICAControllerKeeper.InterchainAccount(sdk.WrapSDKContext(controllerChain.GetContext()), &icacontrollertypes.QueryInterchainAccountRequest{
		Owner:        contractAddr.String(),
		ConnectionId: path.EndpointA.ConnectionID,
	})
	require.NoError(t, err)
	icaAddr := sdk.MustAccAddressFromBech32(icaRsp.GetAddress())
	hostChain.Fund(icaAddr, sdk.NewInt(1_000))

	// submit a tx to controller to be relayed
	targetAddr := sdk.AccAddress(bytes.Repeat([]byte{1}, address.Len))
	sendCoin := sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))
	payloadMsg := banktypes.NewMsgSend(icaAddr, targetAddr, sdk.NewCoins(sendCoin))
	rawPayloadData, err := icatypes.SerializeCosmosTx(controllerChain.Codec, []proto.Message{payloadMsg})
	require.NoError(t, err)
	payloadPacket := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: rawPayloadData,
		Memo: "testing",
	}
	relativeTimeout := uint64(time.Minute.Nanoseconds()) // note this is in nanoseconds
	msgSendTx := icacontrollertypes.NewMsgSendTx(contractAddr.String(), path.EndpointA.ConnectionID, relativeTimeout, payloadPacket)
	MustExecViaStargateReflectContract(t, controllerChain, contractAddr, msgSendTx)
	regMsg := custom.CustomADR8Msg{
		RegisterPacketProcessedCallback: &custom.RegisterPacketProcessedCallback{
			Sequence:            1, // hard code seq here as reflect contract does not read response from transfer msg
			ChannelID:           path.EndpointA.ChannelID,
			PortID:              path.EndpointA.ChannelConfig.PortID,
			MaxCallbackGasLimit: 200_000,
		},
	}
	myCallbackRegistrMsg := MustEncodeAsReflectCustomMessage(t, regMsg)
	MustExecViaReflectContract(t, controllerChain, contractAddr, myCallbackRegistrMsg)

	assert.Equal(t, 1, len(controllerChain.PendingSendPackets))
	require.NoError(t, coord.RelayAndAckPendingPackets(path))

	gotBalance := hostChain.Balance(targetAddr, sdk.DefaultBondDenom)
	assert.Equal(t, sendCoin.String(), gotBalance.String())

	// and verify contract callback executed
	assert.True(t, contractAckCallbackCalled)
}
