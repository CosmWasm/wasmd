package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper *Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper *Keeper) Migrator {
	return Migrator{keeper: keeper}
}

var keyLastInstanceID = append(types.SequenceKeyPrefix, []byte("lastContractId")...)

// Migrate1to2 migrates from version 1 to 2.
// Remove the unused sequence for address generation
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	store := ctx.KVStore(m.keeper.storeKey)
	store.Delete(keyLastInstanceID)
	return nil
}
