package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/exported"
	"github.com/CosmWasm/wasmd/x/wasm/types"
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
	m.keeper.IterateContractInfo(ctx, func(contractAddr sdk.AccAddress, contractInfo types.ContractInfo) bool {
		creator := sdk.MustAccAddressFromBech32(contractInfo.Creator)
		m.keeper.addToContractCreatorSecondaryIndex(ctx, creator, contractInfo.Created, contractAddr)
		return false
	})
	return nil
}

// Migrate2to3 migrates the x/wasm module state from the consensus
// version 2 to version 3. Specifically, it takes the parameters that are currently stored
// and managed by the x/params module and stores them directly into the x/wasm
// module state.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	k := m.keeper
	store := ctx.KVStore(k.storeKey)
	var currParams types.Params
	m.legacySubspace.GetParamSet(ctx, &currParams)

	if err := currParams.ValidateBasic(); err != nil {
		return err
	}

	bz, err := k.cdc.Marshal(&currParams)
	if err != nil {
		return err
	}

	store.Set(types.ParamsKey, bz)

	return nil
}
