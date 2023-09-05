package v4

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Keeper abstract keeper
type wasmKeeper interface {
	PruneWasmCodes(ctx sdk.Context, maxCodeID uint64) error
}

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper wasmKeeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(k wasmKeeper) Migrator {
	return Migrator{keeper: k}
}

// Migrate4to5 migrates from version 4 to 5.
func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	return m.keeper.PruneWasmCodes(ctx, math.MaxUint64)
}
