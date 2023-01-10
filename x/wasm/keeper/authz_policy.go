package keeper

import (
	sdk "github.com/line/lbm-sdk/types"

	"github.com/line/wasmd/x/wasm/types"
)

type AuthorizationPolicy interface {
	CanCreateCode(c types.AccessConfig, creator sdk.AccAddress) bool
	CanInstantiateContract(c types.AccessConfig, actor sdk.AccAddress) bool
	CanModifyContract(admin, actor sdk.AccAddress) bool
}

type DefaultAuthorizationPolicy struct {
}

func (p DefaultAuthorizationPolicy) CanCreateCode(config types.AccessConfig, actor sdk.AccAddress) bool {
	return config.Allowed(actor)
}

func (p DefaultAuthorizationPolicy) CanInstantiateContract(config types.AccessConfig, actor sdk.AccAddress) bool {
	return config.Allowed(actor)
}

func (p DefaultAuthorizationPolicy) CanModifyContract(admin, actor sdk.AccAddress) bool {
	return admin != nil && admin.Equals(actor)
}

// GovAuthorizationPolicy is for the gov handler(proposal_handler.go) authorities
type GovAuthorizationPolicy struct {
}

func (p GovAuthorizationPolicy) CanCreateCode(types.AccessConfig, sdk.AccAddress) bool {
	// The gov handler can create code regardless of the current access config
	return true
}

func (p GovAuthorizationPolicy) CanInstantiateContract(types.AccessConfig, sdk.AccAddress) bool {
	// The gov handler can instantiate contract regardless of the code access config
	return true
}

func (p GovAuthorizationPolicy) CanModifyContract(sdk.AccAddress, sdk.AccAddress) bool {
	// The gov handler can migrate contract regardless of the contract admin
	return true
}
