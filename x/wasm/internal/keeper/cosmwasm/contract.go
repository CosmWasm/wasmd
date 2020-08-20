package cosmwasm

import (
	wasmTypes "github.com/CosmWasm/go-cosmwasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type IBCPacketReceiveResponse struct {
	// Acknowledgement contains the data to acknowledge the ibc packet execution
	Acknowledgement []byte `json:"acknowledgement"`
	// Messages comes directly from the contract and is it's request for action
	Messages []sdk.Msg `json:"messages,omitempty"`
	// Log contains event attributes to expose over abci interface
	Log []wasmTypes.LogAttribute `json:"log,omitempty"`
}

type IBCPacketAcknowledgementResponse struct {
	Messages []sdk.Msg                `json:"messages"`
	Log      []wasmTypes.LogAttribute `json:"log"`
}

type IBCPacketTimeoutResponse struct {
	Messages []sdk.Msg                `json:"messages"`
	Log      []wasmTypes.LogAttribute `json:"log"`
}

type IBCChannelOpenResponse struct {
	// Result contains a boolean if the channel would be accepted
	Result bool `json:"result"`
	// Reason optional description why it was not accepted
	Reason string `json:"reason"`
}

type IBCChannelConnectResponse struct {
	Messages []sdk.Msg                `json:"messages"`
	Log      []wasmTypes.LogAttribute `json:"log"`
}

type IBCChannelCloseResponse struct {
	Messages []sdk.Msg                `json:"messages"`
	Log      []wasmTypes.LogAttribute `json:"log"`
}
