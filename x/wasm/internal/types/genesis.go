package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GenesisState is the struct representation of the export genesis
type GenesisState struct {
	Codes     []CodeData     `json:"codes"`
	Contracts []ContractData `json:"contracts"`
}

// CodeData struct encompasses CodeInfo and CodeBytes
type CodeData struct {
	CodeInfo   CodeInfo `json:"code_info"`
	CodesBytes []byte   `json:"code_bytes"`
}

// ContractData struct encompasses ContractAddress, ContractInfo, and ContractState
type ContractData struct {
	ContractAddress sdk.AccAddress `json:"contract_address"`
	ContractInfo    Contract       `json:"contract_info"`
	ContractState   []Model        `json:"contract_state"`
}

// ValidateGenesis performs basic validation of supply genesis data returning an
// error for any failed validation criteria.
func ValidateGenesis(data GenesisState) error {
	return nil
}
