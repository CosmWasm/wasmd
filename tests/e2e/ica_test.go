package e2e

import (
	"bytes"
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/app"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	hosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wasmibctesting "github.com/CosmWasm/wasmd/x/wasm/ibctesting"
)

func TestICA(t *testing.T) {
	// scenario:
	// given a host and controller chain
	// when an ica is registered on the controller chain
	// and the channel is established to the host chain
	// then the ICA owner can submit a message via IBC
	//      to control their account on the host chain
	coord := wasmibctesting.NewCoordinator(t, 2)
	hostChain := coord.GetChain(ibctesting.GetChainID(1))
	hostParams := hosttypes.NewParams(true, []string{sdk.MsgTypeURL(&banktypes.MsgSend{})})
	hostApp := hostChain.App.(*app.WasmApp)
	hostApp.ICAHostKeeper.SetParams(hostChain.GetContext(), hostParams)

	controllerChain := coord.GetChain(ibctesting.GetChainID(2))

	path := wasmibctesting.NewPath(controllerChain, hostChain)
	coord.SetupConnections(path)

	ownerAddr := sdk.AccAddress(controllerChain.SenderPrivKey.PubKey().Address())
	msg := icacontrollertypes.NewMsgRegisterInterchainAccount(path.EndpointA.ConnectionID, ownerAddr.String(), "")
	res, err := controllerChain.SendMsgs(msg)
	require.NoError(t, err)
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
	contApp := controllerChain.App.(*app.WasmApp)
	icaRsp, err := contApp.ICAControllerKeeper.InterchainAccount(sdk.WrapSDKContext(controllerChain.GetContext()), &icacontrollertypes.QueryInterchainAccountRequest{
		Owner:        ownerAddr.String(),
		ConnectionId: path.EndpointA.ConnectionID,
	})
	require.NoError(t, err)
	icaAddr := sdk.MustAccAddressFromBech32(icaRsp.GetAddress())
	hostChain.Fund(icaAddr, sdk.NewInt(1_000))

	// submit a tx
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
	msgSendTx := icacontrollertypes.NewMsgSendTx(ownerAddr.String(), path.EndpointA.ConnectionID, relativeTimeout, payloadPacket)
	_, err = controllerChain.SendMsgs(msgSendTx)
	require.NoError(t, err)

	assert.Equal(t, 1, len(controllerChain.PendingSendPackets))
	require.NoError(t, coord.RelayAndAckPendingPackets(path))

	gotBalance := hostChain.Balance(targetAddr, sdk.DefaultBondDenom)
	assert.Equal(t, sendCoin.String(), gotBalance.String())
}

func parseIBCChannelEvents(t *testing.T, res *sdk.Result) (string, string, string) {
	t.Helper()
	chanID, err := ibctesting.ParseChannelIDFromEvents(res.GetEvents())
	require.NoError(t, err)
	portID, err := wasmibctesting.ParsePortIDFromEvents(res.GetEvents())
	require.NoError(t, err)
	version, err := wasmibctesting.ParseChannelVersionFromEvents(res.GetEvents())
	require.NoError(t, err)
	return chanID, portID, version
}
