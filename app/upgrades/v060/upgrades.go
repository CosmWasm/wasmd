package v060

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"
	epochstypes "github.com/cosmos/cosmos-sdk/x/epochs/types"
	protocolpooltypes "github.com/cosmos/cosmos-sdk/x/protocolpool/types"

	"github.com/CosmWasm/wasmd/app/upgrades"
)

// UpgradeName defines the on-chain upgrade name
const UpgradeName = "v0.60"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: storetypes.StoreUpgrades{
		Added: []string{
			epochstypes.ModuleName,
			protocolpooltypes.ModuleName,
		},
		Deleted: []string{},
	},
}

func CreateUpgradeHandler(
	mm upgrades.ModuleManager,
	configurator module.Configurator,
	ak *upgrades.AppKeepers,
) upgradetypes.UpgradeHandler {
	// sdk 50 to sdk 53
	return func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
