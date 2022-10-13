package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

type AuthorizationPolicy interface {
	CanCreateCode(c1 types.AccessConfig, creator sdk.AccAddress, c2 types.AccessConfig) bool
	CanInstantiateContract(c types.AccessConfig, actor sdk.AccAddress) bool
	CanModifyContract(admin, actor sdk.AccAddress) bool
	CanModifyCodeAccessConfig(creator, actor sdk.AccAddress, isSubset bool) bool
}

type DefaultAuthorizationPolicy struct{}

func (p DefaultAuthorizationPolicy) CanCreateCode(config types.AccessConfig, actor sdk.AccAddress, defaultConfig types.AccessConfig) bool {
	isSubset := true
	if !config.IsSubset(defaultConfig) {
		isSubset = false
	}
	return config.Allowed(actor) && isSubset
}

func (p DefaultAuthorizationPolicy) CanInstantiateContract(config types.AccessConfig, actor sdk.AccAddress) bool {
	return config.Allowed(actor)
}

func (p DefaultAuthorizationPolicy) CanModifyContract(admin, actor sdk.AccAddress) bool {
	return admin != nil && admin.Equals(actor)
}

func (p DefaultAuthorizationPolicy) CanModifyCodeAccessConfig(creator, actor sdk.AccAddress, isSubset bool) bool {
	return creator != nil && creator.Equals(actor) && isSubset
}

type GovAuthorizationPolicy struct{}

func (p GovAuthorizationPolicy) CanCreateCode(types.AccessConfig, sdk.AccAddress, types.AccessConfig) bool {
	return true
}

func (p GovAuthorizationPolicy) CanInstantiateContract(types.AccessConfig, sdk.AccAddress) bool {
	return true
}

func (p GovAuthorizationPolicy) CanModifyContract(sdk.AccAddress, sdk.AccAddress) bool {
	return true
}

func (p GovAuthorizationPolicy) CanModifyCodeAccessConfig(sdk.AccAddress, sdk.AccAddress, bool) bool {
	return true
}
