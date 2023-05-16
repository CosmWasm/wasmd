package keeper

import (
	"fmt"

	"github.com/CosmWasm/wasmd/x/wasm/types"
	legacytypes "github.com/CosmWasm/wasmd/x/wasm/types/legacy"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	highestCodeID uint64
	keeper        Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{
		highestCodeID: 0,
		keeper:        keeper,
	}
}

// setWasmParams sets the wasm parameters to the default in wasmd
// in terra classic we don't have these params - so we set them
// to default
func (m Migrator) setWasmDefaultParams(ctx sdk.Context) {

	params := types.DefaultParams()
	m.keeper.SetParams(ctx, params)

}

// setLastCodeID sets the LastCodeId store to a value
func (m Migrator) setLastCodeID(ctx sdk.Context, id uint64) {

	store := ctx.KVStore(m.keeper.storeKey)
	bz := sdk.Uint64ToBigEndian(id)
	store.Set(types.KeyLastCodeID, bz)

}

// createCodeFromLegacy - this function migrates the CodeInfo store
func (m Migrator) createCodeFromLegacy(ctx sdk.Context, creator sdk.AccAddress, codeID uint64, hash []byte) error {
	if creator == nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "creator cannot be nil")
	}

	// on terra wasm there was no access config
	// this returns default AccessConfig
	defaultAccessConfig := m.keeper.getInstantiateAccessConfig(ctx).With(creator)

	// unsure whether we need this?
	_, err := m.keeper.wasmVM.AnalyzeCode(hash)
	if err != nil {
		return sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	}

	// can we expect the code IDs to come in order from the
	// iterator? Dunno - that's why we need this mechanism to
	// identify the last ID
	if codeID > m.highestCodeID {
		m.highestCodeID = codeID
		m.setLastCodeID(ctx, m.highestCodeID)
	}

	// Create wasmd compatible CodeInfo and store it
	m.keeper.Logger(ctx).Info(fmt.Sprintf("codeID = %d", codeID))
	codeInfo := types.NewCodeInfo(hash, creator, defaultAccessConfig)
	m.keeper.storeCodeInfo(ctx, codeID, codeInfo)

	return nil
}

// Migrate1to2 migrates from version 1 to 2. Note that this is not the
// canonical migration path, because the terra classic wasm store is
func (m Migrator) Migrate1to2(ctx sdk.Context) error {

	ctx.Logger().Info("### Setting Default wasmd parameters ###")
	m.setWasmDefaultParams(ctx)

	ctx.Logger().Info("### Migrating Code Info ###")
	m.keeper.IterateLegacyCodeInfo(ctx, func(codeInfo legacytypes.CodeInfo) bool {
		creatorAddr, _ := sdk.AccAddressFromBech32(codeInfo.Creator)
		err := m.createCodeFromLegacy(ctx, creatorAddr, codeInfo.CodeID, codeInfo.CodeHash)
		if err != nil {
			m.keeper.Logger(ctx).Error("Was not able to store legacy code ID")
		}
		return false
	})

	// TODO
	m.keeper.Logger(ctx).Info("#### Migrating Contract Info ###")
	m.keeper.IterateLegacyContractInfo(ctx, func(contractInfo legacytypes.ContractInfo) bool {

		// Migrate AbsoluteTxPosition (Testing needed)
		// I am afraid that setting all contracts at one absolute tx position will break query
		createdAt := types.NewAbsoluteTxPosition(ctx)

		creatorAddr, _ := sdk.AccAddressFromBech32(contractInfo.Creator)
		admin, _ := sdk.AccAddressFromBech32(contractInfo.Admin)
		contractAddr, _ := sdk.AccAddressFromBech32(contractInfo.Address)

		newContract := types.NewContractInfo(contractInfo.CodeID, creatorAddr, admin, "", createdAt)
		m.keeper.storeContractInfo(ctx, contractAddr, &newContract)

		return false
	})

	/*m.keeper.IterateContractInfo(ctx, func(contractAddr sdk.AccAddress, contractInfo types.ContractInfo) bool {
		creator := sdk.MustAccAddressFromBech32(contractInfo.Creator)
		m.keeper.addToContractCreatorSecondaryIndex(ctx, creator, contractInfo.Created, contractAddr)
		return false
	})*/

	return nil
}
