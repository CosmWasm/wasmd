package keeper

import (
	"context"
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/assert"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestSelectAuthorizationPolicy(t *testing.T) {
	myGovAuthority := RandomAccountAddress(t)
	m := msgServer{keeper: &Keeper{
		propagateGovAuthorization: map[types.AuthorizationPolicyAction]struct{}{
			types.AuthZActionMigrateContract: {},
			types.AuthZActionInstantiate:     {},
		},
		authority: myGovAuthority.String(),
	}}

	ms := store.NewCommitMultiStore(dbm.NewMemDB(), log.NewTestLogger(t), storemetrics.NewNoOpMetrics())
	ctx := sdk.NewContext(ms, tmproto.Header{}, false, log.NewNopLogger())

	specs := map[string]struct {
		ctx   sdk.Context
		actor sdk.AccAddress
		exp   types.AuthorizationPolicy
	}{
		"always gov policy for gov authority sender": {
			ctx:   types.WithSubMsgAuthzPolicy(ctx, NewPartialGovAuthorizationPolicy(nil, types.AuthZActionMigrateContract)),
			actor: myGovAuthority,
			exp:   NewGovAuthorizationPolicy(types.AuthZActionMigrateContract, types.AuthZActionInstantiate),
		},
		"pick from context when set": {
			ctx:   types.WithSubMsgAuthzPolicy(ctx, NewPartialGovAuthorizationPolicy(nil, types.AuthZActionMigrateContract)),
			actor: RandomAccountAddress(t),
			exp:   NewPartialGovAuthorizationPolicy(nil, types.AuthZActionMigrateContract),
		},
		"fallback to default policy": {
			ctx:   ctx,
			actor: RandomAccountAddress(t),
			exp:   DefaultAuthorizationPolicy{},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			got := m.selectAuthorizationPolicy(spec.ctx, spec.actor.String())
			assert.Equal(t, spec.exp, got)
		})
	}
}

var _ types.AuthorizationPolicy = TestCustomAuthorizationPolicy{}

type TestCustomAuthorizationPolicy struct{}

func (p TestCustomAuthorizationPolicy) CanCreateCode(checksum []byte, chainConfigs types.ChainAccessConfigs, actor sdk.AccAddress, contractConfig types.AccessConfig) bool {
	return true
}

func (p TestCustomAuthorizationPolicy) CanInstantiateContract(code *types.CodeInfo, actor sdk.AccAddress) bool {
	return true
}

func (p TestCustomAuthorizationPolicy) CanModifyContract(contract *types.ContractInfo, actor sdk.AccAddress) bool {
	return true
}

func (p TestCustomAuthorizationPolicy) CanModifyCodeAccessConfig(code *types.CodeInfo, actor sdk.AccAddress, isSubset bool) bool {
	return true
}

func (p TestCustomAuthorizationPolicy) SubMessageAuthorizationPolicy(_ types.AuthorizationPolicyAction) types.AuthorizationPolicy {
	return p
}

func CustomAuthPolicy(ctx context.Context, actor string) (types.AuthorizationPolicy, bool) {
	return TestCustomAuthorizationPolicy{}, true
}

func TestSelectCustomAuthorizationPolicy(t *testing.T) {
	myGovAuthority := RandomAccountAddress(t)
	m := msgServer{keeper: &Keeper{
		propagateGovAuthorization: map[types.AuthorizationPolicyAction]struct{}{
			types.AuthZActionMigrateContract: {},
			types.AuthZActionInstantiate:     {},
		},
		authority:        myGovAuthority.String(),
		customAuthPolicy: CustomAuthPolicy,
	}}

	ms := store.NewCommitMultiStore(dbm.NewMemDB(), log.NewTestLogger(t), storemetrics.NewNoOpMetrics())
	ctx := sdk.NewContext(ms, tmproto.Header{}, false, log.NewNopLogger())

	t.Run("TestSelectCustomAuthorizationPolicy", func(t *testing.T) {
		got := m.selectAuthorizationPolicy(ctx, RandomAccountAddress(t).String())
		assert.Equal(t, TestCustomAuthorizationPolicy{}, got)
	})
}
