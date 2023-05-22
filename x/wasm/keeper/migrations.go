package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/CosmWasm/wasmd/x/wasm/types"
	legacytypes "github.com/CosmWasm/wasmd/x/wasm/types/legacy"
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
		creatorAddr := sdk.MustAccAddressFromBech32(codeInfo.Creator)
		err := m.createCodeFromLegacy(ctx, creatorAddr, codeInfo.CodeID, codeInfo.CodeHash)
		if err != nil {
			m.keeper.Logger(ctx).Error("Was not able to store legacy code ID")
		}
		return false
	})

	// TODO
	m.keeper.Logger(ctx).Info("#### Migrating Contract Info ###")
	m.keeper.IterateLegacyContractInfo(ctx, func(contractInfo legacytypes.ContractInfo) bool {

		contractAddress := sdk.MustAccAddressFromBech32(contractInfo.Address)
		creatorAddr := sdk.MustAccAddressFromBech32(contractInfo.Creator)

		newContract := m.migrateAbsoluteTx(ctx, contractInfo)

		// add to contract history
		history := newContract.InitialHistory(contractInfo.InitMsg)
		m.keeper.appendToContractHistory(ctx, contractAddress, history)
		// add to contract creator secondary index
		m.keeper.addToContractCreatorSecondaryIndex(ctx, creatorAddr, newContract.Created, contractAddress)
		// add to contract code secondary index
		m.keeper.addToContractCodeSecondaryIndex(ctx, contractAddress, history)

		return false
	})

	return nil
}

// Migrate AbsoluteTxPosition (Testing needed)
// I am afraid that setting all contracts at one absolute tx position will break query
func (m Migrator) migrateAbsoluteTx(ctx sdk.Context, contractInfo legacytypes.ContractInfo) types.ContractInfo {
	createdAt := types.NewAbsoluteTxPosition(ctx)

	creatorAddr := sdk.MustAccAddressFromBech32(contractInfo.Creator)
	// admin field can be null in legacy contract
	admin := sdk.AccAddress{}
	if contractInfo.Admin != "" {
		admin = sdk.MustAccAddressFromBech32(contractInfo.Admin)
	}
	contractAddr := sdk.MustAccAddressFromBech32(contractInfo.Address)

	newContract := types.NewContractInfo(contractInfo.CodeID, creatorAddr, admin, createdAt)
	m.keeper.storeContractInfo(ctx, contractAddr, &newContract)

	return newContract
}
