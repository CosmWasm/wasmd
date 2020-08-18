package cosmwasm

import (
	wasmTypes "github.com/CosmWasm/go-cosmwasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type OnReceiveIBCResponse struct {
	Messages []sdk.Msg `json:"messages"`

	Acknowledgement []byte
	// log message to return over abci interface
	Log []wasmTypes.LogAttribute `json:"log"`
}

type OnAcknowledgeIBCResponse struct {
	Messages []sdk.Msg `json:"messages"`

	// log message to return over abci interface
	Log []wasmTypes.LogAttribute `json:"log"`
}
type OnTimeoutIBCResponse struct {
	Messages []sdk.Msg `json:"messages"`

	// log message to return over abci interface
	Log []wasmTypes.LogAttribute `json:"log"`
}

// OnConnectIBCResponse response to a channel open event
type OnConnectIBCResponse struct {
	Messages []sdk.Msg `json:"messages"`

	// log message to return over abci interface
	Log []wasmTypes.LogAttribute `json:"log"`
}

// AcceptChannelResponse is a frame for flow control in wasmd.
type AcceptChannelResponse struct {
	Result                       bool   `json:"result"`
	Reason                       string `json:"reason"`
	RestrictCounterpartyVersions string `json:"accepted_counterparty_version"` // todo: return only 1
}
