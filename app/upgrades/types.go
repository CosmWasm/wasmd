package upgrades

import (
	"context"

	"cosmossdk.io/core/appmodule"
	corestore "cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"
	consensusparamkeeper "cosmossdk.io/x/consensus/keeper"
	paramskeeper "cosmossdk.io/x/params/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	ibckeeper "github.com/cosmos/ibc-go/v9/modules/core/keeper"
)

type AppKeepers struct {
	AccountKeeper         *authkeeper.AccountKeeper
	ParamsKeeper          *paramskeeper.Keeper
	ConsensusParamsKeeper *consensusparamkeeper.Keeper
	Codec                 codec.Codec
	GetStoreKey           func(storeKey string) *storetypes.KVStoreKey
	IBCKeeper             *ibckeeper.Keeper
	AuthKeeper            authkeeper.AccountKeeper
}
type ModuleManager interface {
	RunMigrations(ctx context.Context, cfg module.Configurator, fromVM appmodule.VersionMap) (appmodule.VersionMap, error)
	GetVersionMap() appmodule.VersionMap
}

// Upgrade defines a struct containing necessary fields that a SoftwareUpgradeProposal
// must have written, in order for the state migration to go smoothly.
// An upgrade must implement this struct, and then set it in the app.go.
// The app.go will then define the handler.
type Upgrade struct {
	// Upgrade version name, for the upgrade handler, e.g. `v7`
	UpgradeName string

	// CreateUpgradeHandler defines the function that creates an upgrade handler
	CreateUpgradeHandler func(ModuleManager, module.Configurator, *AppKeepers) upgradetypes.UpgradeHandler
	StoreUpgrades        corestore.StoreUpgrades
}
