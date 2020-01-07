package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GenesisState is the struct representation of the export genesis
type GenesisState struct {
	CodeInfos         []CodeInfo       `json:"code_infos"`
	CodesBytes        [][]byte         `json:"code_bytes"`
	ContractAddresses []sdk.AccAddress `json:"contract_addresses"`
	ContractInfos     []Contract       `json:"contract_infos"`
	ContractStates    [][]Model        `json:"contract_states"`
}

// ValidateGenesis performs basic validation of supply genesis data returning an
// error for any failed validation criteria.
func ValidateGenesis(data GenesisState) error {
	if len(data.CodesBytes) != len(data.CodeInfos) {
		return ErrInvalidGenesis("length of Codes != length of Code Infos")
	}
	if len(data.ContractAddresses) != len(data.ContractInfos) {
		return ErrInvalidGenesis("invalid number of Contract Infos")
	}
	if len(data.ContractAddresses) != len(data.ContractStates) {
		return ErrInvalidGenesis("invalid number of Contract States")
	}
	return nil
}
