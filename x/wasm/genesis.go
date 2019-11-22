package wasm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	// authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	// "github.com/cosmwasm/wasmd/x/wasm/internal/types"
)

type GenesisState struct {
	// TODO
}

// InitGenesis sets supply information for genesis.
//
// CONTRACT: all types of accounts must have been already initialized/created
func InitGenesis(ctx sdk.Context, keeper Keeper, data GenesisState) {
	// TODO
}

// ExportGenesis returns a GenesisState for a given context and keeper.
func ExportGenesis(ctx sdk.Context, keeper Keeper) GenesisState {
	return GenesisState{}
}

// ValidateGenesis performs basic validation of supply genesis data returning an
// error for any failed validation criteria.
func ValidateGenesis(data GenesisState) error {
	return nil
}
