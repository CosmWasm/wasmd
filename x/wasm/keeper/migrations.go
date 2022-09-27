package keeper

import (
	"github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate2to3 migrates from version 2 to 3.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	var allContract []string
	m.keeper.IterateContractInfo(ctx, func(addr sdk.AccAddress, _ types.ContractInfo) bool {
		allContract = append(allContract, addr.String())
		return false
	})
	for _, contract := range allContract {
		contractAddress, err := sdk.AccAddressFromBech32(contract)
		if err != nil {
			return err
		}
		contractInfo := m.keeper.GetContractInfo(ctx, contractAddress)
		creator, err := sdk.AccAddressFromBech32(contractInfo.Creator)
		if err != nil {
			return err
		}
		m.keeper.addToContractCreatorSecondaryIndex(ctx, creator, contractAddress)
	}
	return nil
}
