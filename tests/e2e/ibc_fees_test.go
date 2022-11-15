package e2e

import (
	"bytes"
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/app"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	ibctransfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v4/testing"

	ibcfee "github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"

	wasmibctesting "github.com/CosmWasm/wasmd/x/wasm/ibctesting"
)

func TestIBCFees(t *testing.T) {
	// scenario:
	// given 2 chains
	//   with an ics-20 channel established
	// when an ics-29 fee is attached to an ibc package
	// then the relayer's payee is receiving the fee(s) on success
	marshaler := app.MakeEncodingConfig().Marshaler
	coord := wasmibctesting.NewCoordinator(t, 2)
	chainA := coord.GetChain(ibctesting.GetChainID(0))
	chainB := coord.GetChain(ibctesting.GetChainID(1))

	sender := sdk.AccAddress(chainA.SenderPrivKey.PubKey().Address())
	receiver := sdk.AccAddress(bytes.Repeat([]byte{1}, address.Len))
	payee := sdk.AccAddress(bytes.Repeat([]byte{2}, address.Len))
	oneToken := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1)))

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
	// and with a payee registered on both chains
	_, err := chainA.SendMsgs(ibcfee.NewMsgRegisterPayee(ibctransfertypes.PortID, path.EndpointA.ChannelID, sender.String(), payee.String()))
	require.NoError(t, err)
	_, err = chainB.SendMsgs(ibcfee.NewMsgRegisterCounterpartyPayee(ibctransfertypes.PortID, path.EndpointB.ChannelID, chainB.SenderAccount.GetAddress().String(), payee.String()))
	require.NoError(t, err)

	// when a transfer package is sent
	transferCoin := sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1))
	ibcPayloadMsg := ibctransfertypes.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, transferCoin, sender.String(), receiver.String(), clienttypes.Height{}, uint64(time.Now().Add(time.Minute).UnixNano()))
	ibcPackageFee := ibcfee.NewFee(oneToken, oneToken, sdk.Coins{})
	feeMsg := ibcfee.NewMsgPayPacketFee(ibcPackageFee, ibctransfertypes.PortID, path.EndpointA.ChannelID, sender.String(), nil)
	_, err = chainA.SendMsgs(feeMsg, ibcPayloadMsg)
	require.NoError(t, err)
	pendingIncentivisedPackages := chainA.App.IBCFeeKeeper.GetIdentifiedPacketFeesForChannel(chainA.GetContext(), ibctransfertypes.PortID, path.EndpointA.ChannelID)
	assert.Len(t, pendingIncentivisedPackages, 1)

	// and packages relayed
	require.NoError(t, coord.RelayAndAckPendingPackets(path))

	// then
	expBalance := ibctransfertypes.GetTransferCoin(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, transferCoin.Denom, transferCoin.Amount)
	gotBalance := chainB.Balance(receiver, expBalance.Denom)
	assert.Equal(t, expBalance.String(), gotBalance.String())
	payeeBalance := chainA.AllBalances(payee)
	assert.Equal(t, oneToken.Add(oneToken...).String(), payeeBalance.String())
}
