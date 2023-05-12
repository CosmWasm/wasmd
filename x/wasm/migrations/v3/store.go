package v3

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// StoreCodeInfoFn stores code info
type StoreCodeInfoFn func(ctx sdk.Context, codeID uint64, codeInfo types.CodeInfo)

// Keeper abstract keeper
type wasmKeeper interface {
	IterateCodeInfos(ctx sdk.Context, cb func(uint64, types.CodeInfo) bool)
	GetParams(ctx sdk.Context) types.Params
	SetParams(ctx sdk.Context, ps types.Params) error
}

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper          wasmKeeper
	storeCodeInfoFn StoreCodeInfoFn
}

// NewMigrator returns a new Migrator.
func NewMigrator(k wasmKeeper, fn StoreCodeInfoFn) Migrator {
	return Migrator{keeper: k, storeCodeInfoFn: fn}
}

// Migrate3to4 migrates from version 3 to 4.
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	params := m.keeper.GetParams(ctx)
	if params.CodeUploadAccess.Permission == types.AccessTypeOnlyAddress {
		params.CodeUploadAccess.Permission = types.AccessTypeAnyOfAddresses
		params.CodeUploadAccess.Addresses = []string{params.CodeUploadAccess.Address}
		params.CodeUploadAccess.Address = ""
	}

	if params.InstantiateDefaultPermission == types.AccessTypeOnlyAddress {
		params.InstantiateDefaultPermission = types.AccessTypeAnyOfAddresses
	}

	err := m.keeper.SetParams(ctx, params)
	if err != nil {
		return err
	}

	m.keeper.IterateCodeInfos(ctx, func(codeID uint64, info types.CodeInfo) bool {
		if info.InstantiateConfig.Permission == types.AccessTypeOnlyAddress {
			info.InstantiateConfig.Permission = types.AccessTypeAnyOfAddresses
			info.InstantiateConfig.Addresses = []string{info.InstantiateConfig.Address}
			info.InstantiateConfig.Address = ""

			m.storeCodeInfoFn(ctx, codeID, info)
		}
		return false
	})
	return nil
}
