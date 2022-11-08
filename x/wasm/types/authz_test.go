package types

import (
	"math"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContractAuthzFilterValidate(t *testing.T) {
	specs := map[string]struct {
		src    ContractAuthzFilterX
		expErr bool
	}{
		"allow all": {
			src: &AllowAllMessagesFilter{},
		},
		"allow keys - single": {
			src: NewAcceptedMessageKeysFilter("foo"),
		},
		"allow keys - multi": {
			src: NewAcceptedMessageKeysFilter("foo", "bar"),
		},
		"allow keys - empty": {
			src:    NewAcceptedMessageKeysFilter(),
			expErr: true,
		},
		"allow keys - duplicates": {
			src:    NewAcceptedMessageKeysFilter("foo", "foo"),
			expErr: true,
		},
		"allow keys - whitespaces": {
			src:    NewAcceptedMessageKeysFilter(" foo"),
			expErr: true,
		},
		"allow keys - empty key": {
			src:    NewAcceptedMessageKeysFilter("", "bar"),
			expErr: true,
		},
		"allow keys - whitespace key": {
			src:    NewAcceptedMessageKeysFilter(" ", "bar"),
			expErr: true,
		},
		"allow message - single": {
			src: NewAcceptedMessagesFilter([]byte(`{}`)),
		},
		"allow message - multiple": {
			src: NewAcceptedMessagesFilter([]byte(`{}`), []byte(`{"foo":"bar"}`)),
		},
		"allow message - multiple with empty": {
			src:    NewAcceptedMessagesFilter([]byte(`{}`), nil),
			expErr: true,
		},
		"allow message - duplicate": {
			src:    NewAcceptedMessagesFilter([]byte(`{}`), []byte(`{}`)),
			expErr: true,
		},
		"allow message - non json": {
			src:    NewAcceptedMessagesFilter([]byte("non-json")),
			expErr: true,
		},
		"allow message - empty": {
			src:    NewAcceptedMessagesFilter(),
			expErr: true,
		},
		"allow all message - always valid": {
			src: NewAllowAllMessagesFilter(),
		},
		"undefined - always invalid": {
			src:    &UndefinedFilter{},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotErr := spec.src.ValidateBasic()
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
		})
	}
}

func TestContractAuthzFilterAccept(t *testing.T) {
	specs := map[string]struct {
		filter         ContractAuthzFilterX
		src            RawContractMessage
		exp            bool
		expGasConsumed sdk.Gas
		expErr         bool
	}{
		"allow all - accepts json obj": {
			filter: &AllowAllMessagesFilter{},
			src:    []byte(`{}`),
			exp:    true,
		},
		"allow all - accepts json array": {
			filter: &AllowAllMessagesFilter{},
			src:    []byte(`[{},{}]`),
			exp:    true,
		},
		"allow all - rejects non json msg": {
			filter: &AllowAllMessagesFilter{},
			src:    []byte(``),
			expErr: true,
		},
		"allowed key - single": {
			filter:         NewAcceptedMessageKeysFilter("foo"),
			src:            []byte(`{"foo": "bar"}`),
			exp:            true,
			expGasConsumed: sdk.Gas(len(`{"foo": "bar"}`)),
		},
		"allowed key - multiple": {
			filter:         NewAcceptedMessageKeysFilter("foo", "other"),
			src:            []byte(`{"other": "value"}`),
			exp:            true,
			expGasConsumed: sdk.Gas(len(`{"other": "value"}`)),
		},
		"allowed key - non accepted key": {
			filter:         NewAcceptedMessageKeysFilter("foo"),
			src:            []byte(`{"bar": "value"}`),
			exp:            false,
			expGasConsumed: sdk.Gas(len(`{"bar": "value"}`)),
		},
		"allowed key - unsupported array msg": {
			filter:         NewAcceptedMessageKeysFilter("foo", "other"),
			src:            []byte(`[{"foo":"bar"}]`),
			expErr:         false,
			expGasConsumed: sdk.Gas(len(`[{"foo":"bar"}]`)),
		},
		"allowed key - invalid msg": {
			filter: NewAcceptedMessageKeysFilter("foo", "other"),
			src:    []byte(`not a json msg`),
			expErr: true,
		},
		"allow message - single": {
			filter: NewAcceptedMessagesFilter([]byte(`{}`)),
			src:    []byte(`{}`),
			exp:    true,
		},
		"allow message - multiple": {
			filter: NewAcceptedMessagesFilter([]byte(`[{"foo":"bar"}]`), []byte(`{"other":"value"}`)),
			src:    []byte(`[{"foo":"bar"}]`),
			exp:    true,
		},
		"allow message - no match": {
			filter: NewAcceptedMessagesFilter([]byte(`{"foo":"bar"}`)),
			src:    []byte(`{"other":"value"}`),
			exp:    false,
		},
		"allow all message - always accept valid": {
			filter: NewAllowAllMessagesFilter(),
			src:    []byte(`{"other":"value"}`),
			exp:    true,
		},
		"allow all message - always reject invalid json": {
			filter: NewAllowAllMessagesFilter(),
			src:    []byte(`not json`),
			expErr: true,
		},
		"undefined - always errors": {
			filter: &UndefinedFilter{},
			src:    []byte(`{"foo":"bar"}`),
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gm := sdk.NewGasMeter(1_000_000)
			allowed, gotErr := spec.filter.Accept(sdk.Context{}.WithGasMeter(gm), spec.src)

			// then
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.exp, allowed)
			assert.Equal(t, spec.expGasConsumed, gm.GasConsumed())
		})
	}
}

func TestContractAuthzLimitValidate(t *testing.T) {
	oneToken := sdk.NewCoin(sdk.DefaultBondDenom, sdk.OneInt())
	specs := map[string]struct {
		src    ContractAuthzLimitX
		expErr bool
	}{
		"max calls": {
			src: NewMaxCallsLimit(1),
		},
		"max calls - max uint64": {
			src: NewMaxCallsLimit(math.MaxUint64),
		},
		"max calls - empty": {
			src:    NewMaxCallsLimit(0),
			expErr: true,
		},
		"max funds": {
			src: NewMaxFundsLimit(oneToken),
		},
		"max funds - empty coins": {
			src:    NewMaxFundsLimit(),
			expErr: true,
		},
		"max funds - duplicates": {
			src:    &MaxFundsLimit{Amounts: sdk.Coins{oneToken, oneToken}},
			expErr: true,
		},
		"max funds - contains empty value": {
			src:    &MaxFundsLimit{Amounts: sdk.Coins{oneToken, sdk.NewCoin("other", sdk.ZeroInt())}.Sort()},
			expErr: true,
		},
		"max funds - unsorted": {
			src:    &MaxFundsLimit{Amounts: sdk.Coins{oneToken, sdk.NewCoin("other", sdk.OneInt())}},
			expErr: true,
		},
		"combined": {
			src: NewCombinedLimit(1, oneToken),
		},
		"combined - empty calls": {
			src:    NewCombinedLimit(0, oneToken),
			expErr: true,
		},
		"combined - empty amounts": {
			src:    NewCombinedLimit(1),
			expErr: true,
		},
		"combined - invalid amounts": {
			src:    &CombinedLimit{CallsRemaining: 1, Amounts: sdk.Coins{oneToken, oneToken}},
			expErr: true,
		},
		"undefined": {
			src:    &UndefinedLimit{},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotErr := spec.src.ValidateBasic()
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
		})
	}
}

func TestContractAuthzLimitAccept(t *testing.T) {
	oneToken := sdk.NewCoin(sdk.DefaultBondDenom, sdk.OneInt())
	otherToken := sdk.NewCoin("other", sdk.OneInt())
	specs := map[string]struct {
		limit  ContractAuthzLimitX
		src    AuthzableWasmMsg
		exp    *ContractAuthzLimitAcceptResult
		expErr bool
	}{
		"max calls - updated": {
			limit: NewMaxCallsLimit(2),
			src:   &MsgExecuteContract{},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: true, UpdateLimit: NewMaxCallsLimit(1)},
		},
		"max calls - removed": {
			limit: NewMaxCallsLimit(1),
			src:   &MsgExecuteContract{},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: true, DeleteLimit: true},
		},
		"max calls - accepted with zero fund set": {
			limit: NewMaxCallsLimit(1),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.ZeroInt()))},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: true, DeleteLimit: true},
		},
		"max calls - rejected with some fund transfer": {
			limit: NewMaxCallsLimit(1),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: false},
		},
		"max calls - invalid": {
			limit:  &MaxCallsLimit{},
			src:    &MsgExecuteContract{},
			expErr: true,
		},
		"max funds - single updated": {
			limit: NewMaxFundsLimit(oneToken.Add(oneToken)),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: true, UpdateLimit: NewMaxFundsLimit(oneToken)},
		},
		"max funds - single removed": {
			limit: NewMaxFundsLimit(oneToken),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: true, DeleteLimit: true},
		},
		"max funds - single with unknown token": {
			limit: NewMaxFundsLimit(oneToken),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(otherToken)},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: false},
		},
		"max funds - single exceeds limit": {
			limit: NewMaxFundsLimit(oneToken),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(oneToken.Add(oneToken))},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: false},
		},
		"max funds - single with additional token send": {
			limit: NewMaxFundsLimit(oneToken),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(oneToken, otherToken)},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: false},
		},
		"max funds - multi with other left": {
			limit: NewMaxFundsLimit(oneToken, otherToken),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: true, UpdateLimit: NewMaxFundsLimit(otherToken)},
		},
		"max funds - multi with all used": {
			limit: NewMaxFundsLimit(oneToken, otherToken),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(oneToken, otherToken)},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: true, DeleteLimit: true},
		},
		"max funds - multi with no tokens sent": {
			limit: NewMaxFundsLimit(oneToken, otherToken),
			src:   &MsgExecuteContract{},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: true},
		},
		"max funds - multi with other exceeds limit": {
			limit: NewMaxFundsLimit(oneToken, otherToken),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(oneToken, otherToken.Add(otherToken))},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: false},
		},
		"max combined - multi amounts one consumed": {
			limit: NewCombinedLimit(2, oneToken, otherToken),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: true, UpdateLimit: NewCombinedLimit(1, otherToken)},
		},
		"max combined - multi amounts none consumed": {
			limit: NewCombinedLimit(2, oneToken, otherToken),
			src:   &MsgExecuteContract{},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: true, UpdateLimit: NewCombinedLimit(1, oneToken, otherToken)},
		},
		"max combined - removed on last execution": {
			limit: NewCombinedLimit(1, oneToken, otherToken),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: true, DeleteLimit: true},
		},
		"max combined - removed on last token": {
			limit: NewCombinedLimit(2, oneToken),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: true, DeleteLimit: true},
		},
		"max combined - update with token and calls remaining": {
			limit: NewCombinedLimit(2, oneToken, otherToken),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: true, UpdateLimit: NewCombinedLimit(1, otherToken)},
		},
		"max combined - multi with other exceeds limit": {
			limit: NewCombinedLimit(2, oneToken, otherToken),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(oneToken, otherToken.Add(otherToken))},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: false},
		},
		"max combined - with unknown token": {
			limit: NewCombinedLimit(2, oneToken),
			src:   &MsgExecuteContract{Funds: sdk.NewCoins(otherToken)},
			exp:   &ContractAuthzLimitAcceptResult{Accepted: false},
		},
		"undefined": {
			limit:  &UndefinedLimit{},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotResult, gotErr := spec.limit.Accept(sdk.Context{}, spec.src)
			// then
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.exp, gotResult)
		})
	}
}

func TestValidateContractGrant(t *testing.T) {
	specs := map[string]struct {
		setup  func(t *testing.T) *ContractGrant
		expErr bool
	}{
		"all good": {
			setup: func(t *testing.T) *ContractGrant {
				r, err := NewContractGrant(randBytes(ContractAddrLen), NewMaxCallsLimit(1), NewAllowAllMessagesFilter())
				require.NoError(t, err)
				return r
			},
		},
		"invalid address": {
			setup: func(t *testing.T) *ContractGrant {
				r, err := NewContractGrant([]byte{}, NewMaxCallsLimit(1), NewAllowAllMessagesFilter())
				require.NoError(t, err)
				return r
			},
			expErr: true,
		},
		"invalid limit": {
			setup: func(t *testing.T) *ContractGrant {
				r, err := NewContractGrant(randBytes(ContractAddrLen), NewMaxCallsLimit(0), NewAllowAllMessagesFilter())
				require.NoError(t, err)
				return r
			},
			expErr: true,
		},

		"invalid filter ": {
			setup: func(t *testing.T) *ContractGrant {
				r, err := NewContractGrant(randBytes(ContractAddrLen), NewMaxCallsLimit(1), NewAcceptedMessageKeysFilter())
				require.NoError(t, err)
				return r
			},
			expErr: true,
		},
		"empty limit": {
			setup: func(t *testing.T) *ContractGrant {
				r, err := NewContractGrant(randBytes(ContractAddrLen), NewMaxCallsLimit(0), NewAllowAllMessagesFilter())
				require.NoError(t, err)
				r.Limit = nil
				return r
			},
			expErr: true,
		},

		"empty filter ": {
			setup: func(t *testing.T) *ContractGrant {
				r, err := NewContractGrant(randBytes(ContractAddrLen), NewMaxCallsLimit(1), NewAcceptedMessageKeysFilter())
				require.NoError(t, err)
				r.Filter = nil
				return r
			},
			expErr: true,
		},
		"wrong limit type": {
			setup: func(t *testing.T) *ContractGrant {
				r, err := NewContractGrant(randBytes(ContractAddrLen), NewMaxCallsLimit(0), NewAllowAllMessagesFilter())
				require.NoError(t, err)
				r.Limit = r.Filter
				return r
			},
			expErr: true,
		},

		"wrong filter type": {
			setup: func(t *testing.T) *ContractGrant {
				r, err := NewContractGrant(randBytes(ContractAddrLen), NewMaxCallsLimit(1), NewAcceptedMessageKeysFilter())
				require.NoError(t, err)
				r.Filter = r.Limit
				return r
			},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotErr := spec.setup(t).ValidateBasic()
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
		})
	}
}
