package types

import (
	wasmTypes "github.com/confio/go-cosmwasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth/exported"
)

// Model is a struct that holds a KV pair
type Model struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// CodeData struct encompasses CodeInfo and CodeBytes
type Code struct {
	CodeInfo   CodeInfo `json:"code_info"`
	CodesBytes []byte   `json:"code_bytes"`
}

// CodeInfo is data for the uploaded contract WASM code
type CodeInfo struct {
	CodeHash []byte         `json:"code_hash"`
	Creator  sdk.AccAddress `json:"creator"`
}

// NewCodeInfo fills a new Contract struct
func NewCodeInfo(codeHash []byte, creator sdk.AccAddress) CodeInfo {
	return CodeInfo{
		CodeHash: codeHash,
		Creator:  creator,
	}
}

// ContractData struct encompasses ContractAddress, ContractInfo, and ContractState
type Contract struct {
	ContractAddress sdk.AccAddress `json:"contract_address"`
	ContractInfo    ContractInfo   `json:"contract_info"`
	ContractState   []Model        `json:"contract_state"`
}

// Contract stores a WASM contract instance
type ContractInfo struct {
	CodeID  uint64         `json:"code_id"`
	Creator sdk.AccAddress `json:"creator"`
	InitMsg string         `json:"init_msg"`
}

// NewParams initializes params for a contract instance
func NewParams(ctx sdk.Context, creator sdk.AccAddress, deposit sdk.Coins, contractAcct auth.Account) wasmTypes.Params {
	return wasmTypes.Params{
		Block: wasmTypes.BlockInfo{
			Height:  ctx.BlockHeight(),
			Time:    ctx.BlockTime().Unix(),
			ChainID: ctx.ChainID(),
		},
		Message: wasmTypes.MessageInfo{
			Signer:    creator.String(),
			SentFunds: NewWasmCoins(deposit),
		},
		Contract: wasmTypes.ContractInfo{
			Address: contractAcct.GetAddress().String(),
			Balance: NewWasmCoins(contractAcct.GetCoins()),
		},
	}
}

// NewWasmCoins translates between Cosmos SDK coins and Wasm coins
func NewWasmCoins(cosmosCoins sdk.Coins) (wasmCoins []wasmTypes.Coin) {
	for _, coin := range cosmosCoins {
		wasmCoin := wasmTypes.Coin{
			Denom:  coin.Denom,
			Amount: coin.Amount.String(),
		}
		wasmCoins = append(wasmCoins, wasmCoin)
	}
	return wasmCoins
}

// NewContractInfo creates a new instance of a given WASM contract info
func NewContractInfo(codeID uint64, creator sdk.AccAddress, initMsg string) ContractInfo {
	return ContractInfo{
		CodeID:  codeID,
		Creator: creator,
		InitMsg: initMsg,
	}
}

// CosmosResult converts from a Wasm Result type
func CosmosResult(wasmResult wasmTypes.Result) sdk.Result {
	return sdk.Result{
		Data:    []byte(wasmResult.Data),
		Log:     wasmResult.Log,
		GasUsed: wasmResult.GasUsed,
	}
}
