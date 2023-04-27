package e2e

import (
	"bytes"
	"testing"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
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
