package cosmwasm

import (
	"encoding/json"

	wasmTypes "github.com/CosmWasm/go-cosmwasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type OnReceiveIBCResponse struct {
	Messages []sdk.Msg `json:"messages"`

	Acknowledgement json.RawMessage
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

type AcceptChannelResponse struct {
	Result                       bool     `json:"result"`
	Reason                       string   `json:"reason"`
	RestrictCounterpartyVersions []string `json:"accepted_counterpary_versions"`
}
