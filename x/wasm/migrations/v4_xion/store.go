package v4

import (
	"context"

	corestoretypes "cosmossdk.io/core/store"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// StoreContractInfoFn stores contract info
type StoreContractInfoFn func(ctx context.Context, contractAddress sdk.AccAddress, contractInfo *types.ContractInfo)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	storeContractInfoFn StoreContractInfoFn
}

// NewMigrator returns a new Migrator.
func NewMigrator(fn StoreContractInfoFn) Migrator {
	return Migrator{storeContractInfoFn: fn}
}

// Migrate4to5 migrates from consensus version 4 to 5.
// This migration handles the FIELD ORDER SWAP between wasmd v0.61.2 and v0.61.6.
//
// BACKGROUND:
// wasmd v0.61.0-v0.61.4 had INCORRECT field order in ContractInfo:
//   - field 7 = ibc2_port_id (string)
//   - field 8 = extension (Any)
//
// wasmd v0.61.5+ FIXED the field order (commit 6cbaaae4):
//   - field 7 = extension (Any)
//   - field 8 = ibc2_port_id (string)
//
// This field swap causes "proto: illegal wireType 7" errors because:
//   - Reading field 7 expects Any type but finds string data
//   - Reading field 8 expects string but finds Any type data
//
// MIGRATION STRATEGY:
// 1. Read each ContractInfo using LegacyContractInfo (v0.61.2 schema)
// 2. Convert to new ContractInfo (v0.61.6 schema) with fields swapped
// 3. Store with correct field positions
//
// References:
// - https://github.com/CosmWasm/wasmd/issues/2386
// - https://github.com/CosmWasm/wasmd/pull/2123
// - https://github.com/CosmWasm/wasmd/commit/6cbaaae4
func (m Migrator) Migrate4to5(ctx sdk.Context, storeService corestoretypes.KVStoreService, cdc codec.BinaryCodec) error {
	store := storeService.OpenKVStore(ctx)
	prefixStore := prefix.NewStore(runtime.KVStoreAdapter(store), types.ContractKeyPrefix)
	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		// Unmarshal using LEGACY schema (field 7 = ibc2_port_id, field 8 = extension)
		var legacyInfo LegacyContractInfo
		if err := cdc.Unmarshal(iter.Value(), &legacyInfo); err != nil {
			// Skip if unmarshal fails (shouldn't happen in normal operation)
			continue
		}

		// Convert to NEW schema (field 7 = extension, field 8 = ibc2_port_id)
		newInfo := types.ContractInfo{
			CodeID:     legacyInfo.CodeID,
			Creator:    legacyInfo.Creator,
			Admin:      legacyInfo.Admin,
			Label:      legacyInfo.Label,
			IBCPortID:  legacyInfo.IBCPortID,
			IBC2PortID: legacyInfo.IBC2PortID, // Moved from field 7 to field 8
		}

		// Copy Created field
		if legacyInfo.Created != nil {
			newInfo.Created = &types.AbsoluteTxPosition{
				BlockHeight: legacyInfo.Created.BlockHeight,
				TxIndex:     legacyInfo.Created.TxIndex,
			}
		}

		// Copy Extension field - moved from field 8 to field 7
		if legacyInfo.Extension != nil {
			newInfo.Extension = legacyInfo.Extension
		}

		// Store with NEW schema
		contractAddress := sdk.AccAddress(iter.Key())
		m.storeContractInfoFn(ctx, contractAddress, &newInfo)
	}

	return nil
}
