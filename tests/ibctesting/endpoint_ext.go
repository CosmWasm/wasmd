package ibctesting

import (
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
)

// ChanCloseConfirm will construct and execute a NewMsgChannelCloseConfirm on the associated endpoint.
func (endpoint *Endpoint) ChanCloseConfirm() error {
	channelKey := host.ChannelKey(endpoint.Counterparty.ChannelConfig.PortID, endpoint.Counterparty.ChannelID)
	proof, proofHeight := endpoint.Counterparty.QueryProof(channelKey)

	msg := channeltypes.NewMsgChannelCloseConfirm(
		endpoint.ChannelConfig.PortID, endpoint.ChannelID,
		proof, proofHeight,
		endpoint.Chain.SenderAccount.GetAddress().String(),
		0,
	)
	return endpoint.Chain.sendMsgs(msg)
}

// SetChannelClosed sets a channel state to CLOSED.
func (endpoint *Endpoint) SetChannelClosed() error {
	channel := endpoint.GetChannel()

	channel.State = channeltypes.CLOSED
	endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID, channel)

	endpoint.Chain.Coordinator.CommitBlock(endpoint.Chain)

	return endpoint.Counterparty.UpdateClient()
}
