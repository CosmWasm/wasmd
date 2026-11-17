package v080

import (
	"context"
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/CosmWasm/wasmd/app/upgrades"
)

// UpgradeName defines the on-chain upgrade name
const UpgradeName = "v0.80"

// Names used by x/protocolpool, which was removed in Cosmos SDK v0.55.
const (
	ProtocolPoolModuleName    = "protocolpool"
	ProtocolPoolEscrowAccount = "protocolpool_escrow"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: storetypes.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{ProtocolPoolModuleName},
	},
}

func CreateUpgradeHandler(
	mm upgrades.ModuleManager,
	configurator module.Configurator,
	ak *upgrades.AppKeepers,
) upgradetypes.UpgradeHandler {
	// sdk 54 to sdk 55
	return func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		vm, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return nil, err
		}

		// x/distribution manages the community pool again, so move the funds
		// left in the x/protocolpool module accounts into it.
		for _, name := range []string{ProtocolPoolModuleName, ProtocolPoolEscrowAccount} {
			addr := authtypes.NewModuleAddress(name)
			balance := ak.BankKeeper.GetAllBalances(ctx, addr)
			if balance.IsZero() {
				continue
			}
			if err := ak.DistrKeeper.FundCommunityPool(ctx, balance, addr); err != nil {
				return nil, fmt.Errorf("move %s funds to community pool: %w", name, err)
			}
		}
		return vm, nil
	}
}
