package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/exported"
	v1 "github.com/CosmWasm/wasmd/x/wasm/migrations/v1"
	v2 "github.com/CosmWasm/wasmd/x/wasm/migrations/v2"
	v3 "github.com/CosmWasm/wasmd/x/wasm/migrations/v3"
	v4_xion "github.com/CosmWasm/wasmd/x/wasm/migrations/v4_xion"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper         Keeper
	legacySubspace exported.Subspace
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper, legacySubspace exported.Subspace) Migrator {
	return Migrator{keeper: keeper, legacySubspace: legacySubspace}
}

// Migrate1to2 migrates from version 1 to 2.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v1.NewMigrator(m.keeper, m.keeper.addToContractCreatorSecondaryIndex).Migrate1to2(ctx)
}

// Migrate2to3 migrates the x/wasm module state from the consensus
// version 2 to version 3.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	return v2.MigrateStore(ctx, m.keeper.storeService, m.legacySubspace, m.keeper.cdc)
}

// Migrate3to4 migrates the x/wasm module state from the consensus
// version 3 to version 4.
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	return v3.NewMigrator(m.keeper, m.keeper.mustStoreCodeInfo).Migrate3to4(ctx, m.keeper.storeService, m.keeper.cdc)
}

// Migrate4to5 migrates the x/wasm module state from the consensus
// version 4 to version 5. This migration fixes the ContractInfo field order
// swap between v0.61.2 (incorrect) and v0.61.6 (correct).
func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	return v4_xion.NewMigrator(m.keeper.mustStoreContractInfo).Migrate4to5(ctx, m.keeper.storeService, m.keeper.cdc)
}
