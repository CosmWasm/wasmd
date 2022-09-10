package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

func TestMigrateV1ToV2(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)
	store := ctx.KVStore(keepers.WasmKeeper.storeKey)
	store.Set(keyLastInstanceID, sdk.Uint64ToBigEndian(100))

	// when
	NewMigrator(keepers.WasmKeeper).Migrate1to2(ctx)

	// then
	assert.False(t, store.Has(keyLastInstanceID))
}
