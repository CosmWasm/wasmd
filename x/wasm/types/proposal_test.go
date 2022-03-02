package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypesv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestvalidateProposalCommons(t *testing.T) {
	type commonProposal struct {
		Title, Description string
	}

	specs := map[string]struct {
		src    commonProposal
		expErr bool
	}{
		"all good": {src: commonProposal{
			Title:       "Foo",
			Description: "Bar",
		}},
		"prevent empty title": {
			src: commonProposal{
				Description: "Bar",
			},
			expErr: true,
		},
		"prevent white space only title": {
			src: commonProposal{
				Title:       " ",
				Description: "Bar",
			},
			expErr: true,
		},
		"prevent leading white spaces in title": {
			src: commonProposal{
				Title:       " Foo",
				Description: "Bar",
			},
			expErr: true,
		},
		"prevent title exceeds max length ": {
			src: commonProposal{
				Title:       strings.Repeat("a", govtypesv1beta1.MaxTitleLength+1),
				Description: "Bar",
			},
			expErr: true,
		},
		"prevent empty description": {
			src: commonProposal{
				Title: "Foo",
			},
			expErr: true,
		},
		"prevent leading white spaces in description": {
			src: commonProposal{
				Title:       "Foo",
				Description: " Bar",
			},
			expErr: true,
		},
		"prevent white space only description": {
			src: commonProposal{
				Title:       "Foo",
				Description: " ",
			},
			expErr: true,
		},
		"prevent descr exceeds max length ": {
			src: commonProposal{
				Title:       "Foo",
				Description: strings.Repeat("a", govtypesv1beta1.MaxDescriptionLength+1),
			},
			expErr: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			err := validateProposalCommons(spec.src.Title, spec.src.Description)
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateMigrateContractProposal(t *testing.T) {
	var (
		invalidAddress = "invalid address2"
	)

	specs := map[string]struct {
		src    *MigrateContractProposal
		expErr bool
	}{
		"all good": {
			src: MigrateContractProposalFixture(),
		},
		"without migrate msg": {
			src: MigrateContractProposalFixture(func(p *MigrateContractProposal) {
				p.Msg = nil
			}),
			expErr: true,
		},
		"migrate msg with invalid json": {
			src: MigrateContractProposalFixture(func(p *MigrateContractProposal) {
				p.Msg = []byte("not a json message")
			}),
			expErr: true,
		},
		"base data missing": {
			src: MigrateContractProposalFixture(func(p *MigrateContractProposal) {
				p.Title = ""
			}),
			expErr: true,
		},
		"contract missing": {
			src: MigrateContractProposalFixture(func(p *MigrateContractProposal) {
				p.Contract = ""
			}),
			expErr: true,
		},
		"contract invalid": {
			src: MigrateContractProposalFixture(func(p *MigrateContractProposal) {
				p.Contract = invalidAddress
			}),
			expErr: true,
		},
		"code id empty": {
			src: MigrateContractProposalFixture(func(p *MigrateContractProposal) {
				p.CodeID = 0
			}),
			expErr: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			err := spec.src.ValidateBasic()
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateUpdateAdminProposal(t *testing.T) {
	var (
		invalidAddress = "invalid address"
	)

	specs := map[string]struct {
		src    *UpdateAdminProposal
		expErr bool
	}{
		"all good": {
			src: UpdateAdminProposalFixture(),
		},
		"base data missing": {
			src: UpdateAdminProposalFixture(func(p *UpdateAdminProposal) {
				p.Title = ""
			}),
			expErr: true,
		},
		"contract missing": {
			src: UpdateAdminProposalFixture(func(p *UpdateAdminProposal) {
				p.Contract = ""
			}),
			expErr: true,
		},
		"contract invalid": {
			src: UpdateAdminProposalFixture(func(p *UpdateAdminProposal) {
				p.Contract = invalidAddress
			}),
			expErr: true,
		},
		"admin missing": {
			src: UpdateAdminProposalFixture(func(p *UpdateAdminProposal) {
				p.NewAdmin = ""
			}),
			expErr: true,
		},
		"admin invalid": {
			src: UpdateAdminProposalFixture(func(p *UpdateAdminProposal) {
				p.NewAdmin = invalidAddress
			}),
			expErr: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			err := spec.src.ValidateBasic()
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateClearAdminProposal(t *testing.T) {
	var (
		invalidAddress = "invalid address"
	)

	specs := map[string]struct {
		src    *ClearAdminProposal
		expErr bool
	}{
		"all good": {
			src: ClearAdminProposalFixture(),
		},
		"base data missing": {
			src: ClearAdminProposalFixture(func(p *ClearAdminProposal) {
				p.Title = ""
			}),
			expErr: true,
		},
		"contract missing": {
			src: ClearAdminProposalFixture(func(p *ClearAdminProposal) {
				p.Contract = ""
			}),
			expErr: true,
		},
		"contract invalid": {
			src: ClearAdminProposalFixture(func(p *ClearAdminProposal) {
				p.Contract = invalidAddress
			}),
			expErr: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			err := spec.src.ValidateBasic()
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestProposalStrings(t *testing.T) {
	specs := map[string]struct {
		src govtypesv1beta1.Content
		exp string
	}{
		"migrate contract": {
			src: MigrateContractProposalFixture(),
			exp: `Migrate Contract Proposal:
  Title:       Foo
  Description: Bar
  Contract:    cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr
  Code id:     1
  Msg:         "{\"verifier\":\"cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqs2m6sx4\"}"
`,
		},
		"update admin": {
			src: UpdateAdminProposalFixture(),
			exp: `Update Contract Admin Proposal:
  Title:       Foo
  Description: Bar
  Contract:    cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr
  New Admin:   cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqs2m6sx4
`,
		},
		"clear admin": {
			src: ClearAdminProposalFixture(),
			exp: `Clear Contract Admin Proposal:
  Title:       Foo
  Description: Bar
  Contract:    cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr
`,
		},
		"pin codes": {
			src: &PinCodesProposal{
				Title:       "Foo",
				Description: "Bar",
				CodeIDs:     []uint64{1, 2, 3},
			},
			exp: `Pin Wasm Codes Proposal:
  Title:       Foo
  Description: Bar
  Codes:       [1 2 3]
`,
		},
		"unpin codes": {
			src: &UnpinCodesProposal{
				Title:       "Foo",
				Description: "Bar",
				CodeIDs:     []uint64{3, 2, 1},
			},
			exp: `Unpin Wasm Codes Proposal:
  Title:       Foo
  Description: Bar
  Codes:       [3 2 1]
`,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			assert.Equal(t, spec.exp, spec.src.String())
		})
	}
}

func TestProposalYaml(t *testing.T) {
	specs := map[string]struct {
		src govtypesv1beta1.Content
		exp string
	}{
		"migrate contract": {
			src: MigrateContractProposalFixture(),
			exp: `title: Foo
description: Bar
contract: cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr
code_id: 1
msg: '{"verifier":"cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqs2m6sx4"}'
`,
		},
		"update admin": {
			src: UpdateAdminProposalFixture(),
			exp: `title: Foo
description: Bar
new_admin: cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqs2m6sx4
contract: cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr
`,
		},
		"clear admin": {
			src: ClearAdminProposalFixture(),
			exp: `title: Foo
description: Bar
contract: cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr
`,
		},
		"pin codes": {
			src: &PinCodesProposal{
				Title:       "Foo",
				Description: "Bar",
				CodeIDs:     []uint64{1, 2, 3},
			},
			exp: `title: Foo
description: Bar
code_ids:
- 1
- 2
- 3
`,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			v, err := yaml.Marshal(&spec.src)
			require.NoError(t, err)
			assert.Equal(t, spec.exp, string(v))
		})
	}
}

func TestConvertToProposals(t *testing.T) {
	cases := map[string]struct {
		input     string
		isError   bool
		proposals []ProposalType
	}{
		"one proper item": {
			input:     "UpdateAdmin",
			proposals: []ProposalType{ProposalTypeUpdateAdmin},
		},
		"multiple proper items": {
			input:     "StoreCode,InstantiateContract,MigrateContract",
			proposals: []ProposalType{ProposalTypeStoreCode, ProposalTypeInstantiateContract, ProposalTypeMigrateContract},
		},
		"empty trailing item": {
			input:   "StoreCode,",
			isError: true,
		},
		"invalid item": {
			input:   "StoreCode,InvalidProposalType",
			isError: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			chunks := strings.Split(tc.input, ",")
			proposals, err := ConvertToProposals(chunks)
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, proposals, tc.proposals)
			}
		})
	}
}

func TestUnmarshalContentFromJson(t *testing.T) {
	specs := map[string]struct {
		src string
		got govtypesv1beta1.Content
		exp govtypesv1beta1.Content
	}{
		"instantiate ": {
			src: `
{
	"title": "foo",
	"description": "bar",
	"admin": "myAdminAddress",
	"code_id": 1,
	"funds": [{"denom": "ALX", "amount": "2"},{"denom": "BLX","amount": "3"}],
	"msg": {},
	"label": "testing",
	"run_as": "myRunAsAddress"
}`,
			got: &InstantiateContractProposal{},
			exp: &InstantiateContractProposal{
				Title:       "foo",
				Description: "bar",
				RunAs:       "myRunAsAddress",
				Admin:       "myAdminAddress",
				CodeID:      1,
				Label:       "testing",
				Msg:         []byte("{}"),
				Funds:       sdk.NewCoins(sdk.NewCoin("ALX", sdk.NewInt(2)), sdk.NewCoin("BLX", sdk.NewInt(3))),
			},
		},
		"migrate ": {
			src: `
{
	"title": "foo",
	"description": "bar",
	"code_id": 1,
	"contract": "myContractAddr",
	"msg": {},
	"run_as": "myRunAsAddress"
}`,
			got: &MigrateContractProposal{},
			exp: &MigrateContractProposal{
				Title:       "foo",
				Description: "bar",
				Contract:    "myContractAddr",
				CodeID:      1,
				Msg:         []byte("{}"),
			},
		},
	}
	for name, spec := range specs {
		fmt.Println(name)
		t.Run(name, func(t *testing.T) {
			require.NoError(t, json.Unmarshal([]byte(spec.src), spec.got))
			assert.Equal(t, spec.exp, spec.got)
		})
	}
}
func TestProposalJsonSignBytes(t *testing.T) {
	const myInnerMsg = `{"foo":"bar"}`
	specs := map[string]struct {
		src govtypesv1beta1.Content
		exp string
	}{
		"instantiate contract": {
			src: &InstantiateContractProposal{Msg: RawContractMessage(myInnerMsg)},
			exp: `
{
	"type":"cosmos-sdk/MsgSubmitProposal",
	"value":{"content":{"type":"wasm/InstantiateContractProposal","value":{"funds":[],"msg":{"foo":"bar"}}},"initial_deposit":[]}
}`,
		},
		"migrate contract": {
			src: &MigrateContractProposal{Msg: RawContractMessage(myInnerMsg)},
			exp: `
{
	"type":"cosmos-sdk/MsgSubmitProposal",
	"value":{"content":{"type":"wasm/MigrateContractProposal","value":{"msg":{"foo":"bar"}}},"initial_deposit":[]}
}`,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			msg, err := govtypesv1beta1.NewMsgSubmitProposal(spec.src, sdk.NewCoins(), []byte{})
			require.NoError(t, err)

			bz := msg.GetSignBytes()
			assert.JSONEq(t, spec.exp, string(bz), "raw: %s", string(bz))
		})
	}
}
