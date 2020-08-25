package cosmwasm

import (
	cosmwasmv1 "github.com/CosmWasm/go-cosmwasm/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
)

type IBCEndpoint struct {
	Channel string `json:"channel"`
	Port    string `json:"port"`
}

type IBCChannel struct {
	Endpoint             IBCEndpoint
	CounterpartyEndpoint IBCEndpoint
	Order                channeltypes.Order
	Version              string
	// CounterpartyVersion can be nil when not known this context, yet
	CounterpartyVersion *string `json:"counterparty_version,omitempty"`
}

type IBCPacket struct {
	Data []byte
	// identifies the channel and port on the sending chain.
	Source IBCEndpoint
	// identifies the channel and port on the receiving chain.
	Destination IBCEndpoint
	Sequence    uint64
	// block height after which the packet times out
	TimeoutHeight uint64
	// block timestamp (in nanoseconds) after which the packet times out
	TimeoutTimestamp uint64
}

type IBCAcknowledgement struct {
	Acknowledgement []byte    `json:"acknowledgement"`
	OriginalPacket  IBCPacket `json:"original_packet"`
}

type IBCPacketReceiveResponse struct {
	// Acknowledgement contains the data to acknowledge the ibc packet execution
	Acknowledgement []byte `json:"acknowledgement"`
	// Messages comes directly from the contract and is it's request for action
	Messages []CosmosMsg `json:"messages,omitempty"`
	// Log contains event attributes to expose over abci interface
	Log []cosmwasmv1.LogAttribute `json:"log,omitempty"`
}

type IBCPacketAcknowledgementResponse struct {
	Messages []CosmosMsg               `json:"messages"`
	Log      []cosmwasmv1.LogAttribute `json:"log"`
}

type IBCPacketTimeoutResponse struct {
	Messages []CosmosMsg               `json:"messages"`
	Log      []cosmwasmv1.LogAttribute `json:"log"`
}

type IBCChannelOpenResponse struct {
	// Success contains a boolean if the channel would be accepted
	Success bool `json:"result"`
	// Reason optional description why it was not accepted
	Reason string `json:"reason"`
}

type IBCChannelConnectResponse struct {
	Messages []CosmosMsg               `json:"messages"`
	Log      []cosmwasmv1.LogAttribute `json:"log"`
}

type IBCChannelCloseResponse struct {
	Messages []CosmosMsg               `json:"messages"`
	Log      []cosmwasmv1.LogAttribute `json:"log"`
}
