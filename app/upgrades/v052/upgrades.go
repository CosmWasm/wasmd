package v050

import (
	"context"

	"cosmossdk.io/core/appmodule"
	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/x/accounts"
	epochstypes "cosmossdk.io/x/epochs/types"
	protocolpooltypes "cosmossdk.io/x/protocolpool/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	"github.com/CosmWasm/wasmd/app/upgrades"
)

// UpgradeName defines the on-chain upgrade name
const UpgradeName = "v0.52"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: corestore.StoreUpgrades{
		Added: []string{
			accounts.StoreKey,
			protocolpooltypes.StoreKey,
			epochstypes.StoreKey,
		},
		Deleted: []string{"crisis"}, // The SDK discontinued the crisis module in v0.52.0
	},
}

func CreateUpgradeHandler(
	mm upgrades.ModuleManager,
	configurator module.Configurator,
	ak *upgrades.AppKeepers,
) upgradetypes.UpgradeHandler {
	// sdk 50 to sdk 52
	return func(ctx context.Context, plan upgradetypes.Plan, fromVM appmodule.VersionMap) (appmodule.VersionMap, error) {
		err := authkeeper.MigrateAccountNumberUnsafe(ctx, &ak.AuthKeeper)
		if err != nil {
			return nil, err
		}

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
