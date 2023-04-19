package types

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/Finschia/finschia-sdk/types"
	"github.com/Finschia/finschia-sdk/x/auth/legacy/legacytx"

	wasmTypes "github.com/Finschia/wasmd/x/wasm/types"
)

func NewMsgStoreCodeAndInstantiateContract(fromAddr sdk.AccAddress) *MsgStoreCodeAndInstantiateContract {
	return &MsgStoreCodeAndInstantiateContract{Sender: fromAddr.String()}
}

func TestStoreCodeAndInstantiateContractValidation(t *testing.T) {
	bad, err := sdk.AccAddressFromHex("012345")
	require.NoError(t, err)
	badAddress := bad.String()
	require.NoError(t, err)
	// proper address size
	goodAddress := sdk.AccAddress(make([]byte, wasmTypes.ContractAddrLen)).String()
	sdk.GetConfig().SetAddressVerifier(wasmTypes.VerifyAddressLen())

	cases := map[string]struct {
		msg   MsgStoreCodeAndInstantiateContract
		valid bool
	}{
		"empty": {
			msg:   MsgStoreCodeAndInstantiateContract{},
			valid: false,
		},
		"correct minimal": {
			msg: MsgStoreCodeAndInstantiateContract{
				Sender:       goodAddress,
				WASMByteCode: []byte("foo"),
				Label:        "foo",
				Msg:          []byte("{}"),
			},
			valid: true,
		},
		"missing code": {
			msg: MsgStoreCodeAndInstantiateContract{
				Sender: goodAddress,
				Label:  "foo",
				Msg:    []byte("{}"),
			},
			valid: false,
		},
		"missing label": {
			msg: MsgStoreCodeAndInstantiateContract{
				Sender:       goodAddress,
				WASMByteCode: []byte("foo"),
				Msg:          []byte("{}"),
			},
			valid: false,
		},
		"missing init message": {
			msg: MsgStoreCodeAndInstantiateContract{
				Sender:       goodAddress,
				WASMByteCode: []byte("foo"),
				Label:        "foo",
			},
			valid: false,
		},
		"correct maximal": {
			msg: MsgStoreCodeAndInstantiateContract{
				Sender:       goodAddress,
				WASMByteCode: []byte("foo"),
				Label:        "foo",
				Msg:          []byte(`{"some": "data"}`),
				Funds:        sdk.Coins{sdk.Coin{Denom: "foobar", Amount: sdk.NewInt(200)}},
			},
			valid: true,
		},
		"invalid InstantiatePermission": {
			msg: MsgStoreCodeAndInstantiateContract{
				Sender:                goodAddress,
				WASMByteCode:          []byte("foo"),
				InstantiatePermission: &wasmTypes.AccessConfig{Permission: wasmTypes.AccessTypeOnlyAddress, Address: badAddress},
				Label:                 "foo",
				Msg:                   []byte(`{"some": "data"}`),
				Funds:                 sdk.Coins{sdk.Coin{Denom: "foobar", Amount: sdk.NewInt(200)}},
			},
			valid: false,
		},
		"negative funds": {
			msg: MsgStoreCodeAndInstantiateContract{
				Sender:       goodAddress,
				WASMByteCode: []byte("foo"),
				Msg:          []byte(`{"some": "data"}`),
				// we cannot use sdk.NewCoin() constructors as they panic on creating invalid data (before we can test)
				Funds: sdk.Coins{sdk.Coin{Denom: "foobar", Amount: sdk.NewInt(-200)}},
			},
			valid: false,
		},
		"non json init msg": {
			msg: MsgStoreCodeAndInstantiateContract{
				Sender:       goodAddress,
				WASMByteCode: []byte("foo"),
				Label:        "foo",
				Msg:          []byte("invalid-json"),
			},
			valid: false,
		},
		"bad sender minimal": {
			msg: MsgStoreCodeAndInstantiateContract{
				Sender:       badAddress,
				WASMByteCode: []byte("foo"),
				Label:        "foo",
				Msg:          []byte("{}"),
			},
			valid: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestNewMsgStoreCodeAndInstantiateContractGetSigners(t *testing.T) {
	res := NewMsgStoreCodeAndInstantiateContract(sdk.AccAddress([]byte("input111111111111111"))).GetSigners()
	bytes := sdk.MustAccAddressFromBech32(res[0].String())
	require.Equal(t, "696e707574313131313131313131313131313131", fmt.Sprintf("%v", hex.EncodeToString(bytes)))
}

func TestMsgJsonSignBytes(t *testing.T) {
	const myInnerMsg = `{"foo":"bar"}`
	specs := map[string]struct {
		src legacytx.LegacyMsg
		exp string
	}{
		"MsgInstantiateContract with every field": {
			src: &MsgStoreCodeAndInstantiateContract{
				Sender: "sender1", WASMByteCode: []byte{89, 69, 76, 76, 79, 87, 32, 83, 85, 66, 77, 65, 82, 73, 78, 69},
				InstantiatePermission: &wasmTypes.AccessConfig{Permission: wasmTypes.AccessTypeAnyOfAddresses, Addresses: []string{"address1", "address2"}},
				Admin:                 "admin1", Label: "My", Msg: wasmTypes.RawContractMessage(myInnerMsg), Funds: sdk.Coins{{Denom: "denom1", Amount: sdk.NewInt(1)}},
			},
			exp: `
{
	"type":"wasm/MsgStoreCodeAndInstantiateContract",
	"value": {"admin":"admin1","funds":[{"amount":"1","denom":"denom1"}],"instantiate_permission":{"addresses":["address1","address2"],
		"permission":"AnyOfAddresses"},"label":"My","msg":{"foo":"bar"},"sender":"sender1","wasm_byte_code":"WUVMTE9XIFNVQk1BUklORQ=="}
}`,
		},
		"MsgInstantiateContract with minimum field": {
			src: &MsgStoreCodeAndInstantiateContract{},
			exp: `
{
	"type":"wasm/MsgStoreCodeAndInstantiateContract",
	"value": {"funds":[]}
}`,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			bz := spec.src.GetSignBytes()
			assert.JSONEq(t, spec.exp, string(bz), "raw: %s", string(bz))
		})
	}
}
