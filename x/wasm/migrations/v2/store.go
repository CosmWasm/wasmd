package v2

import (
	"errors"

	corestoretypes "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MigrateStore rejects migrations that require the removed legacy params module.
func MigrateStore(ctx sdk.Context, _ corestoretypes.KVStoreService, _ codec.BinaryCodec) error {
	return errors.New("x/wasm v2->v3 migration needs x/params; upgrade through wasmd v0.70.4 first")
}
