package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestDefaultAuthzPolicyCanCreateCode(t *testing.T) {
	myActorAddress := RandomAccountAddress(t)
	otherAddress := RandomAccountAddress(t)
	specs := map[string]struct {
		config types.AccessConfig
		actor  sdk.AccAddress
		exp    bool
		panics bool
	}{
		"nobody": {
			config: types.AllowNobody,
			exp:    false,
		},
		"everybody": {
			config: types.AllowEverybody,
			exp:    true,
		},
		"only address - same": {
			config: types.AccessTypeOnlyAddress.With(myActorAddress),
			exp:    true,
		},
		"only address - different": {
			config: types.AccessTypeOnlyAddress.With(otherAddress),
			exp:    false,
		},
		"any address - included": {
			config: types.AccessTypeAnyOfAddresses.With(otherAddress, myActorAddress),
			exp:    true,
		},
		"any address - not included": {
			config: types.AccessTypeAnyOfAddresses.With(otherAddress),
			exp:    false,
		},
		"undefined config - panics": {
			config: types.AccessConfig{},
			panics: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			policy := DefaultAuthorizationPolicy{}
			if !spec.panics {
				got := policy.CanCreateCode(spec.config, myActorAddress)
				assert.Equal(t, spec.exp, got)
				return
			}
			assert.Panics(t, func() {
				policy.CanCreateCode(spec.config, myActorAddress)
			})
		})
	}
}

func TestDefaultAuthzPolicyCanInstantiateContract(t *testing.T) {
	myActorAddress := RandomAccountAddress(t)
	otherAddress := RandomAccountAddress(t)
	specs := map[string]struct {
		config types.AccessConfig
		actor  sdk.AccAddress
		exp    bool
		panics bool
	}{
		"nobody": {
			config: types.AllowNobody,
			exp:    false,
		},
		"everybody": {
			config: types.AllowEverybody,
			exp:    true,
		},
		"only address - same": {
			config: types.AccessTypeOnlyAddress.With(myActorAddress),
			exp:    true,
		},
		"only address - different": {
			config: types.AccessTypeOnlyAddress.With(otherAddress),
			exp:    false,
		},
		"any address - included": {
			config: types.AccessTypeAnyOfAddresses.With(otherAddress, myActorAddress),
			exp:    true,
		},
		"any address - not included": {
			config: types.AccessTypeAnyOfAddresses.With(otherAddress),
			exp:    false,
		},
		"undefined config - panics": {
			config: types.AccessConfig{},
			panics: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			policy := DefaultAuthorizationPolicy{}
			if !spec.panics {
				got := policy.CanInstantiateContract(spec.config, myActorAddress)
				assert.Equal(t, spec.exp, got)
				return
			}
			assert.Panics(t, func() {
				policy.CanInstantiateContract(spec.config, myActorAddress)
			})
		})
	}
}

func TestDefaultAuthzPolicyCanModifyContract(t *testing.T) {
	myActorAddress := RandomAccountAddress(t)
	otherAddress := RandomAccountAddress(t)

	specs := map[string]struct {
		admin sdk.AccAddress
		exp   bool
	}{
		"same as actor": {
			admin: myActorAddress,
			exp:   true,
		},
		"different admin": {
			admin: otherAddress,
			exp:   false,
		},
		"no admin": {
			exp: false,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			policy := DefaultAuthorizationPolicy{}
			got := policy.CanModifyContract(spec.admin, myActorAddress)
			assert.Equal(t, spec.exp, got)
		})
	}
}

func TestDefaultAuthzPolicyCanModifyCodeAccessConfig(t *testing.T) {
	myActorAddress := RandomAccountAddress(t)
	otherAddress := RandomAccountAddress(t)

	specs := map[string]struct {
		admin  sdk.AccAddress
		subset bool
		exp    bool
	}{
		"same as actor - subset": {
			admin:  myActorAddress,
			subset: true,
			exp:    true,
		},
		"same as actor - not subset": {
			admin:  myActorAddress,
			subset: false,
			exp:    false,
		},
		"different admin": {
			admin: otherAddress,
			exp:   false,
		},
		"no admin": {
			exp: false,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			policy := DefaultAuthorizationPolicy{}
			got := policy.CanModifyCodeAccessConfig(spec.admin, myActorAddress, spec.subset)
			assert.Equal(t, spec.exp, got)
		})
	}
}

func TestGovAuthzPolicyCanCreateCode(t *testing.T) {
	myActorAddress := RandomAccountAddress(t)
	otherAddress := RandomAccountAddress(t)
	specs := map[string]struct {
		config types.AccessConfig
		actor  sdk.AccAddress
	}{
		"nobody": {
			config: types.AllowNobody,
		},
		"everybody": {
			config: types.AllowEverybody,
		},
		"only address - same": {
			config: types.AccessTypeOnlyAddress.With(myActorAddress),
		},
		"only address - different": {
			config: types.AccessTypeOnlyAddress.With(otherAddress),
		},
		"any address - included": {
			config: types.AccessTypeAnyOfAddresses.With(otherAddress, myActorAddress),
		},
		"any address - not included": {
			config: types.AccessTypeAnyOfAddresses.With(otherAddress),
		},
		"undefined config - panics": {
			config: types.AccessConfig{},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			policy := GovAuthorizationPolicy{}
			got := policy.CanCreateCode(spec.config, myActorAddress)
			assert.True(t, got)
		})
	}
}

func TestGovAuthzPolicyCanInstantiateContract(t *testing.T) {
	myActorAddress := RandomAccountAddress(t)
	otherAddress := RandomAccountAddress(t)
	specs := map[string]struct {
		config types.AccessConfig
		actor  sdk.AccAddress
	}{
		"nobody": {
			config: types.AllowNobody,
		},
		"everybody": {
			config: types.AllowEverybody,
		},
		"only address - same": {
			config: types.AccessTypeOnlyAddress.With(myActorAddress),
		},
		"only address - different": {
			config: types.AccessTypeOnlyAddress.With(otherAddress),
		},
		"any address - included": {
			config: types.AccessTypeAnyOfAddresses.With(otherAddress, myActorAddress),
		},
		"any address - not included": {
			config: types.AccessTypeAnyOfAddresses.With(otherAddress),
		},
		"undefined config - panics": {
			config: types.AccessConfig{},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			policy := GovAuthorizationPolicy{}
			got := policy.CanInstantiateContract(spec.config, myActorAddress)
			assert.True(t, got)
		})
	}
}

func TestGovAuthzPolicyCanModifyContract(t *testing.T) {
	myActorAddress := RandomAccountAddress(t)
	otherAddress := RandomAccountAddress(t)

	specs := map[string]struct {
		admin sdk.AccAddress
	}{
		"same as actor": {
			admin: myActorAddress,
		},
		"different admin": {
			admin: otherAddress,
		},
		"no admin": {},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			policy := GovAuthorizationPolicy{}
			got := policy.CanModifyContract(spec.admin, myActorAddress)
			assert.True(t, got)
		})
	}
}

func TestGovAuthzPolicyCanModifyCodeAccessConfig(t *testing.T) {
	myActorAddress := RandomAccountAddress(t)
	otherAddress := RandomAccountAddress(t)

	specs := map[string]struct {
		admin  sdk.AccAddress
		subset bool
	}{
		"same as actor - subset": {
			admin:  myActorAddress,
			subset: true,
		},
		"same as actor - not subset": {
			admin:  myActorAddress,
			subset: false,
		},
		"different admin": {
			admin: otherAddress,
		},
		"no admin": {},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			policy := GovAuthorizationPolicy{}
			got := policy.CanModifyCodeAccessConfig(spec.admin, myActorAddress, spec.subset)
			assert.True(t, got)
		})
	}
}
