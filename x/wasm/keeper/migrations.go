package keeper

import (
	"github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate2to3 migrates from version 2 to 3.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	m.keeper.IterateContractInfo(ctx, func(contractAddr sdk.AccAddress, contractInfo types.ContractInfo) bool {
		creator := sdk.MustAccAddressFromBech32(contractInfo.Creator)
		m.keeper.addToContractCreatorSecondaryIndex(ctx, creator, contractInfo.Created, contractAddr)
		return false
	})
	return nil
}
