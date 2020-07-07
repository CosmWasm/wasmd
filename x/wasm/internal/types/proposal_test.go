package types

import (
	"bytes"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWasmProposal(t *testing.T) {
	specs := map[string]struct {
		src    WasmProposal
		expErr bool
	}{
		"all good": {src: WasmProposal{
			Title:       "Foo",
			Description: "Bar",
		}},
		"prevent empty title": {
			src: WasmProposal{
				Description: "Bar",
			},
			expErr: true,
		},
		"prevent white space only title": {
			src: WasmProposal{
				Title:       " ",
				Description: "Bar",
			},
			expErr: true,
		},
		"prevent leading white spaces in title": {
			src: WasmProposal{
				Title:       " Foo",
				Description: "Bar",
			},
			expErr: true,
		},
		"prevent title exceeds max length ": {
			src: WasmProposal{
				Title:       strings.Repeat("a", govtypes.MaxTitleLength+1),
				Description: "Bar",
			},
			expErr: true,
		},
		"prevent empty description": {
			src: WasmProposal{
				Title: "Foo",
			},
			expErr: true,
		},
		"prevent leading white spaces in description": {
			src: WasmProposal{
				Title:       "Foo",
				Description: " Bar",
			},
			expErr: true,
		},
		"prevent white space only description": {
			src: WasmProposal{
				Title:       "Foo",
				Description: " ",
			},
			expErr: true,
		},
		"prevent descr exceeds max length ": {
			src: WasmProposal{
				Title:       "Foo",
				Description: strings.Repeat("a", govtypes.MaxDescriptionLength+1),
			},
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

func TestValidateStoreCodeProposal(t *testing.T) {
	var (
		invalidAddress sdk.AccAddress = bytes.Repeat([]byte{0x1}, sdk.AddrLen-1)
	)

	specs := map[string]struct {
		src    StoreCodeProposal
		expErr bool
	}{
		"all good": {
			src: StoreCodeProposalFixture(),
		},
		"without source": {
			src: StoreCodeProposalFixture(func(p *StoreCodeProposal) {
				p.Source = ""
			}),
		},
		"base data missing": {
			src: StoreCodeProposalFixture(func(p *StoreCodeProposal) {
				p.WasmProposal = WasmProposal{}
			}),
			expErr: true,
		},
		"creator missing": {
			src: StoreCodeProposalFixture(func(p *StoreCodeProposal) {
				p.Creator = nil
			}),
			expErr: true,
		},
		"creator invalid": {
			src: StoreCodeProposalFixture(func(p *StoreCodeProposal) {
				p.Creator = invalidAddress
			}),
			expErr: true,
		},
		"wasm code missing": {
			src: StoreCodeProposalFixture(func(p *StoreCodeProposal) {
				p.WASMByteCode = nil
			}),
			expErr: true,
		},
		"wasm code invalid": {
			src: StoreCodeProposalFixture(func(p *StoreCodeProposal) {
				p.WASMByteCode = bytes.Repeat([]byte{0x0}, MaxWasmSize+1)
			}),
			expErr: true,
		},
		"source invalid": {
			src: StoreCodeProposalFixture(func(p *StoreCodeProposal) {
				p.Source = "not an url"
			}),
			expErr: true,
		},
		"builder invalid": {
			src: StoreCodeProposalFixture(func(p *StoreCodeProposal) {
				p.Builder = "not a builder"
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

func TestValidateInstantiateContractProposal(t *testing.T) {
	var (
		invalidAddress sdk.AccAddress = bytes.Repeat([]byte{0x1}, sdk.AddrLen-1)
	)

	specs := map[string]struct {
		src    InstantiateContractProposal
		expErr bool
	}{
		"all good": {
			src: InstantiateContractProposalFixture(),
		},
		"without admin": {
			src: InstantiateContractProposalFixture(func(p *InstantiateContractProposal) {
				p.Admin = nil
			}),
		},
		"without init msg": {
			src: InstantiateContractProposalFixture(func(p *InstantiateContractProposal) {
				p.InitMsg = nil
			}),
		},
		"without init funds": {
			src: InstantiateContractProposalFixture(func(p *InstantiateContractProposal) {
				p.InitFunds = nil
			}),
		},
		"base data missing": {
			src: InstantiateContractProposalFixture(func(p *InstantiateContractProposal) {
				p.WasmProposal = WasmProposal{}
			}),
			expErr: true,
		},
		"creator missing": {
			src: InstantiateContractProposalFixture(func(p *InstantiateContractProposal) {
				p.Creator = nil
			}),
			expErr: true,
		},
		"creator invalid": {
			src: InstantiateContractProposalFixture(func(p *InstantiateContractProposal) {
				p.Creator = invalidAddress
			}),
			expErr: true,
		},
		"admin invalid": {
			src: InstantiateContractProposalFixture(func(p *InstantiateContractProposal) {
				p.Admin = invalidAddress
			}),
			expErr: true,
		},
		"code id empty": {
			src: InstantiateContractProposalFixture(func(p *InstantiateContractProposal) {
				p.Code = 0
			}),
			expErr: true,
		},
		"label empty": {
			src: InstantiateContractProposalFixture(func(p *InstantiateContractProposal) {
				p.Label = ""
			}),
			expErr: true,
		},
		"init funds negative": {
			src: InstantiateContractProposalFixture(func(p *InstantiateContractProposal) {
				p.InitFunds = sdk.Coins{{Denom: "foo", Amount: sdk.NewInt(-1)}}
			}),
			expErr: true,
		},
		"init funds with duplicates": {
			src: InstantiateContractProposalFixture(func(p *InstantiateContractProposal) {
				p.InitFunds = sdk.Coins{{Denom: "foo", Amount: sdk.NewInt(1)}, {Denom: "foo", Amount: sdk.NewInt(2)}}
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

func TestValidateMigrateContractProposal(t *testing.T) {
	var (
		invalidAddress sdk.AccAddress = bytes.Repeat([]byte{0x1}, sdk.AddrLen-1)
	)

	specs := map[string]struct {
		src    MigrateContractProposal
		expErr bool
	}{
		"all good": {
			src: MigrateContractProposalFixture(),
		},
		"without migrate msg": {
			src: MigrateContractProposalFixture(func(p *MigrateContractProposal) {
				p.MigrateMsg = nil
			}),
		},
		"base data missing": {
			src: MigrateContractProposalFixture(func(p *MigrateContractProposal) {
				p.WasmProposal = WasmProposal{}
			}),
			expErr: true,
		},
		"contract missing": {
			src: MigrateContractProposalFixture(func(p *MigrateContractProposal) {
				p.Contract = nil
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
				p.Code = 0
			}),
			expErr: true,
		},
		"sender missing": {
			src: MigrateContractProposalFixture(func(p *MigrateContractProposal) {
				p.Sender = nil
			}),
			expErr: true,
		},
		"sender invalid": {
			src: MigrateContractProposalFixture(func(p *MigrateContractProposal) {
				p.Sender = invalidAddress
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
		invalidAddress sdk.AccAddress = bytes.Repeat([]byte{0x1}, sdk.AddrLen-1)
	)

	specs := map[string]struct {
		src    UpdateAdminProposal
		expErr bool
	}{
		"all good": {
			src: UpdateAdminProposalFixture(),
		},
		"base data missing": {
			src: UpdateAdminProposalFixture(func(p *UpdateAdminProposal) {
				p.WasmProposal = WasmProposal{}
			}),
			expErr: true,
		},
		"contract missing": {
			src: UpdateAdminProposalFixture(func(p *UpdateAdminProposal) {
				p.Contract = nil
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
				p.NewAdmin = nil
			}),
			expErr: true,
		},
		"admin invalid": {
			src: UpdateAdminProposalFixture(func(p *UpdateAdminProposal) {
				p.NewAdmin = invalidAddress
			}),
			expErr: true,
		},
		"sender missing": {
			src: UpdateAdminProposalFixture(func(p *UpdateAdminProposal) {
				p.Sender = nil
			}),
			expErr: true,
		},
		"sender invalid": {
			src: UpdateAdminProposalFixture(func(p *UpdateAdminProposal) {
				p.Sender = invalidAddress
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
		invalidAddress sdk.AccAddress = bytes.Repeat([]byte{0x1}, sdk.AddrLen-1)
	)

	specs := map[string]struct {
		src    ClearAdminProposal
		expErr bool
	}{
		"all good": {
			src: ClearAdminProposalFixture(),
		},
		"base data missing": {
			src: ClearAdminProposalFixture(func(p *ClearAdminProposal) {
				p.WasmProposal = WasmProposal{}
			}),
			expErr: true,
		},
		"contract missing": {
			src: ClearAdminProposalFixture(func(p *ClearAdminProposal) {
				p.Contract = nil
			}),
			expErr: true,
		},
		"contract invalid": {
			src: ClearAdminProposalFixture(func(p *ClearAdminProposal) {
				p.Contract = invalidAddress
			}),
			expErr: true,
		},
		"sender missing": {
			src: ClearAdminProposalFixture(func(p *ClearAdminProposal) {
				p.Sender = nil
			}),
			expErr: true,
		},
		"sender invalid": {
			src: ClearAdminProposalFixture(func(p *ClearAdminProposal) {
				p.Sender = invalidAddress
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
		src gov.Content
		exp string
	}{
		"store code": {
			src: StoreCodeProposalFixture(func(p *StoreCodeProposal) {
				p.WASMByteCode = []byte{01, 02, 03, 04, 05, 06, 07, 0x08, 0x09, 0x0a}
			}),
			exp: `Store Code Proposal:
  Title:       Foo
  Description: Bar
  Creator:     cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du
  WasmCode:    0102030405060708090A
  Source:      https://example.com/code
  Builder:     foo/bar:latest
`,
		},
		"instantiate contract": {
			src: InstantiateContractProposalFixture(func(p *InstantiateContractProposal) {
				p.InitFunds = sdk.Coins{{Denom: "foo", Amount: sdk.NewInt(1)}, {Denom: "bar", Amount: sdk.NewInt(2)}}
			}),
			exp: `Instantiate Code Proposal:
  Title:       Foo
  Description: Bar
  Creator:     cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du
  Admin:       cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du
  Code id:     1
  Label:       testing
  InitMsg:     "{\"verifier\":\"cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du\",\"beneficiary\":\"cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du\"}"
  InitFunds:   1foo,2bar
`,
		},
		"instantiate contract without funds": {
			src: InstantiateContractProposalFixture(func(p *InstantiateContractProposal) { p.InitFunds = nil }),
			exp: `Instantiate Code Proposal:
  Title:       Foo
  Description: Bar
  Creator:     cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du
  Admin:       cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du
  Code id:     1
  Label:       testing
  InitMsg:     "{\"verifier\":\"cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du\",\"beneficiary\":\"cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du\"}"
  InitFunds:   
`,
		},
		"instantiate contract without admin": {
			src: InstantiateContractProposalFixture(func(p *InstantiateContractProposal) { p.Admin = nil }),
			exp: `Instantiate Code Proposal:
  Title:       Foo
  Description: Bar
  Creator:     cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du
  Admin:       
  Code id:     1
  Label:       testing
  InitMsg:     "{\"verifier\":\"cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du\",\"beneficiary\":\"cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du\"}"
  InitFunds:   
`,
		},
		"migrate contract": {
			src: MigrateContractProposalFixture(),
			exp: `Migrate Contract Proposal:
  Title:       Foo
  Description: Bar
  Contract:    cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5
  Code id:     1
  Sender:      cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du
  MigrateMsg   "{\"verifier\":\"cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du\"}"
`,
		},
		"update admin": {
			src: UpdateAdminProposalFixture(),
			exp: `Update Contract Admin Proposal:
  Title:       Foo
  Description: Bar
  Contract:    cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5
  Sender:      cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du
  New Admin:   cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du
`,
		},
		"clear admin": {
			src: ClearAdminProposalFixture(),
			exp: `Clear Contract Admin Proposal:
  Title:       Foo
  Description: Bar
  Contract:    cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5
  Sender:      cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du
`,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			assert.Equal(t, spec.exp, spec.src.String())
		})
	}

}
