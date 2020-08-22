package cosmwasm

import (
	"encoding/json"

	cosmwasmv1 "github.com/CosmWasm/go-cosmwasm/types"
)

// CosmosMsg is an rust enum and only (exactly) one of the fields should be set
// Should we do a cleaner approach in Go? (type/data?)
type CosmosMsg struct {
	Bank    *cosmwasmv1.BankMsg    `json:"bank,omitempty"`
	Custom  json.RawMessage        `json:"custom,omitempty"`
	Staking *cosmwasmv1.StakingMsg `json:"staking,omitempty"`
	Wasm    *cosmwasmv1.WasmMsg    `json:"wasm,omitempty"`
	IBC     *IBCMsg                `json:"wasm_ibc,omitempty"`
}

type IBCMsg struct {
	SendPacket   *IBCSendMsg         `json:"execute,omitempty"`
	CloseChannel *IBCCloseChannelMsg `json:"instantiate,omitempty"`
	Transfer     *IBCTransferMsg     `json:"instantiate,omitempty"`
}

type IBCSendMsg struct {
	// This is our contract-local ID
	ChannelID string
	Data      []byte
	// optional fields (or do we need exactly/at least one of these?)
	TimeoutHeight    uint64
	TimeoutTimestamp uint64
}

type IBCCloseChannelMsg struct {
	ChannelID string
}
type IBCTransferMsg struct {
}

//------- Results / Msgs -------------

// HandleResult is the raw response from the handle call
type HandleResult struct {
	Ok  *HandleResponse      `json:"Ok,omitempty"`
	Err *cosmwasmv1.StdError `json:"Err,omitempty"`
}

// HandleResponse defines the return value on a successful handle
type HandleResponse struct {
	// Messages comes directly from the contract and is it's request for action
	Messages []CosmosMsg `json:"messages"`
	// base64-encoded bytes to return as ABCI.Data field
	Data []byte `json:"data"`
	// log message to return over abci interface
	Log []cosmwasmv1.LogAttribute `json:"log"`
}

//type WasmMsg struct {
//	Execute     *cosmwasmv1.ExecuteMsg     `json:"execute,omitempty"`
//	Instantiate *cosmwasmv1.InstantiateMsg `json:"instantiate,omitempty"`
//}
//
//// ExecuteMsg is used to call another defined contract on this chain.
//// The calling contract requires the callee to be defined beforehand,
//// and the address should have been defined in initialization.
//// And we assume the developer tested the ABIs and coded them together.
////
//// Since a contract is immutable once it is deployed, we don't need to transform this.
//// If it was properly coded and worked once, it will continue to work throughout upgrades.
//type ExecuteMsg struct {
//	// ContractAddr is the sdk.AccAddress of the contract, which uniquely defines
//	// the contract ID and instance ID. The sdk module should maintain a reverse lookup table.
//	ContractAddr string `json:"contract_addr"`
//	// Msg is assumed to be a json-encoded message, which will be passed directly
//	// as `userMsg` when calling `Handle` on the above-defined contract
//	Msg []byte `json:"msg"`
//	// Send is an optional amount of coins this contract sends to the called contract
//	Send cosmwasmv1.Coins `json:"send"`
//}
