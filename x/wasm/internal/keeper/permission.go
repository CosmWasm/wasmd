package keeper

import (
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AuthorizationSchema interface {
	CanModifyContractAdmin(c types.ContractInfo, actor sdk.AccAddress) bool
}

type DefaultAuthorizationSchema struct {
}

func (p DefaultAuthorizationSchema) CanModifyContractAdmin(c types.ContractInfo, actor sdk.AccAddress) bool {
	return c.Admin != nil && c.Admin.Equals(actor)
}

type GovAuthorizationSchema struct {
}

func (p GovAuthorizationSchema) CanModifyContractAdmin(c types.ContractInfo, actor sdk.AccAddress) bool {
	return true
}
