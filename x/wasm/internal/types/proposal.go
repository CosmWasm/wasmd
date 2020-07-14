package types

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	ProposalTypeStoreCode                = "StoreCode"
	ProposalTypeStoreInstantiateContract = "InstantiateContract"
	ProposalTypeMigrateContract          = "MigrateContract"
	ProposalTypeUpdateAdmin              = "UpdateAdmin"
	ProposalTypeClearAdmin               = "ClearAdmin"
)

var DefaultEnabledProposals = map[string]struct{}{
	ProposalTypeStoreCode:                {},
	ProposalTypeStoreInstantiateContract: {},
	ProposalTypeMigrateContract:          {},
	ProposalTypeUpdateAdmin:              {},
	ProposalTypeClearAdmin:               {},
}

func init() { // register new content types with the sdk
	govtypes.RegisterProposalType(ProposalTypeStoreCode)
	govtypes.RegisterProposalType(ProposalTypeStoreInstantiateContract)
	govtypes.RegisterProposalType(ProposalTypeMigrateContract)
	govtypes.RegisterProposalType(ProposalTypeUpdateAdmin)
	govtypes.RegisterProposalType(ProposalTypeClearAdmin)
	govtypes.RegisterProposalTypeCodec(StoreCodeProposal{}, "wasm/store-proposal")
	govtypes.RegisterProposalTypeCodec(InstantiateContractProposal{}, "wasm/instantiate-proposal")
	govtypes.RegisterProposalTypeCodec(MigrateContractProposal{}, "wasm/migrate-proposal")
	govtypes.RegisterProposalTypeCodec(UpdateAdminProposal{}, "wasm/update-admin-proposal")
	govtypes.RegisterProposalTypeCodec(ClearAdminProposal{}, "wasm/clear-admin-proposal")
}

type WasmProposal struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
}

// GetTitle returns the title of a parameter change proposal.
func (p WasmProposal) GetTitle() string { return p.Title }

// GetDescription returns the description of a parameter change proposal.
func (p WasmProposal) GetDescription() string { return p.Description }

// ProposalRoute returns the routing key of a parameter change proposal.
func (p WasmProposal) ProposalRoute() string { return RouterKey }

// ValidateBasic validates the proposal
func (p WasmProposal) ValidateBasic() error {
	if strings.TrimSpace(p.Title) != p.Title {
		return sdkerrors.Wrap(govtypes.ErrInvalidProposalContent, "proposal title must not start/end with white spaces")
	}
	if len(p.Title) == 0 {
		return sdkerrors.Wrap(govtypes.ErrInvalidProposalContent, "proposal title cannot be blank")
	}
	if len(p.Title) > govtypes.MaxTitleLength {
		return sdkerrors.Wrapf(govtypes.ErrInvalidProposalContent, "proposal title is longer than max length of %d", govtypes.MaxTitleLength)
	}
	if strings.TrimSpace(p.Description) != p.Description {
		return sdkerrors.Wrap(govtypes.ErrInvalidProposalContent, "proposal description must not start/end with white spaces")
	}
	if len(p.Description) == 0 {
		return sdkerrors.Wrap(govtypes.ErrInvalidProposalContent, "proposal description cannot be blank")
	}
	if len(p.Description) > govtypes.MaxDescriptionLength {
		return sdkerrors.Wrapf(govtypes.ErrInvalidProposalContent, "proposal description is longer than max length of %d", govtypes.MaxDescriptionLength)
	}
	return nil
}

type StoreCodeProposal struct {
	WasmProposal
	// RunAs is the address that "owns" the code object
	RunAs sdk.AccAddress `json:"run_as"`
	// WASMByteCode can be raw or gzip compressed
	WASMByteCode []byte `json:"wasm_byte_code"`
	// Source is a valid absolute HTTPS URI to the contract's source code, optional
	Source string `json:"source"`
	// Builder is a valid docker image name with tag, optional
	Builder string `json:"builder"`
	// InstantiatePermission to apply on contract creation, optional
	InstantiatePermission *AccessConfig `json:"instantiate_permission"`
}

// ProposalType returns the type
func (p StoreCodeProposal) ProposalType() string { return ProposalTypeStoreCode }

// ValidateBasic validates the proposal
func (p StoreCodeProposal) ValidateBasic() error {
	if err := p.WasmProposal.ValidateBasic(); err != nil {
		return err
	}
	if err := sdk.VerifyAddressFormat(p.RunAs); err != nil {
		return sdkerrors.Wrap(err, "run as")
	}

	if err := validateWasmCode(p.WASMByteCode); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "code bytes %s", err.Error())
	}

	if err := validateSourceURL(p.Source); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "source %s", err.Error())
	}

	if err := validateBuilder(p.Builder); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "builder %s", err.Error())
	}
	if p.InstantiatePermission != nil {
		if err := p.InstantiatePermission.ValidateBasic(); err != nil {
			return sdkerrors.Wrap(err, "instantiate permission")
		}
	}
	return nil
}

// String implements the Stringer interface.
func (p StoreCodeProposal) String() string {
	return fmt.Sprintf(`Store Code Proposal:
  Title:       %s
  Description: %s
  Run as:      %s
  WasmCode:    %X
  Source:      %s
  Builder:     %s
`, p.Title, p.Description, p.RunAs, p.WASMByteCode, p.Source, p.Builder)
}

func (p StoreCodeProposal) MarshalYAML() (interface{}, error) {
	return struct {
		WasmProposal          `yaml:",inline"`
		RunAs                 sdk.AccAddress `yaml:"run_as"`
		WASMByteCode          string         `yaml:"wasm_byte_code"`
		Source                string         `yaml:"source"`
		Builder               string         `yaml:"builder"`
		InstantiatePermission *AccessConfig  `yaml:"instantiate_permission"`
	}{
		WasmProposal:          p.WasmProposal,
		RunAs:                 p.RunAs,
		WASMByteCode:          base64.StdEncoding.EncodeToString(p.WASMByteCode),
		Source:                p.Source,
		Builder:               p.Builder,
		InstantiatePermission: p.InstantiatePermission,
	}, nil
}

type InstantiateContractProposal struct {
	WasmProposal
	// RunAs is the address that pays the init funds
	RunAs sdk.AccAddress `json:"run_as"`
	// Admin is an optional address that can execute migrations
	Admin     sdk.AccAddress  `json:"admin,omitempty"`
	Code      uint64          `json:"code_id"`
	Label     string          `json:"label"`
	InitMsg   json.RawMessage `json:"init_msg"`
	InitFunds sdk.Coins       `json:"init_funds"`
}

// ProposalType returns the type
func (p InstantiateContractProposal) ProposalType() string {
	return ProposalTypeStoreInstantiateContract
}

// ValidateBasic validates the proposal
func (p InstantiateContractProposal) ValidateBasic() error {
	if err := p.WasmProposal.ValidateBasic(); err != nil {
		return err
	}
	if err := sdk.VerifyAddressFormat(p.RunAs); err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "run as")
	}

	if p.Code == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "code id is required")
	}

	if err := validateLabel(p.Label); err != nil {
		return err
	}

	if !p.InitFunds.IsValid() {
		return sdkerrors.ErrInvalidCoins
	}

	if len(p.Admin) != 0 {
		if err := sdk.VerifyAddressFormat(p.Admin); err != nil {
			return err
		}
	}
	return nil

}

// String implements the Stringer interface.
func (p InstantiateContractProposal) String() string {
	return fmt.Sprintf(`Instantiate Code Proposal:
  Title:       %s
  Description: %s
  Run as:      %s
  Admin:       %s
  Code id:     %d
  Label:       %s
  InitMsg:     %q
  InitFunds:   %s
`, p.Title, p.Description, p.RunAs, p.Admin, p.Code, p.Label, p.InitMsg, p.InitFunds)
}

func (p InstantiateContractProposal) MarshalYAML() (interface{}, error) {
	return struct {
		WasmProposal `yaml:",inline"`
		RunAs        sdk.AccAddress `yaml:"run_as"`
		Admin        sdk.AccAddress `yaml:"admin"`
		Code         uint64         `yaml:"code_id"`
		Label        string         `yaml:"label"`
		InitMsg      string         `yaml:"init_msg"`
		InitFunds    sdk.Coins      `yaml:"init_funds"`
	}{
		WasmProposal: p.WasmProposal,
		RunAs:        p.RunAs,
		Admin:        p.Admin,
		Code:         p.Code,
		Label:        p.Label,
		InitMsg:      string(p.InitMsg),
		InitFunds:    p.InitFunds,
	}, nil
}

type MigrateContractProposal struct {
	WasmProposal `yaml:",inline"`
	Contract     sdk.AccAddress  `json:"contract"`
	Code         uint64          `json:"code_id"`
	MigrateMsg   json.RawMessage `json:"msg"`
	// RunAs is the address that is passed to the contract's environment as sender
	RunAs sdk.AccAddress `json:"run_as"`
}

// ProposalType returns the type
func (p MigrateContractProposal) ProposalType() string { return ProposalTypeMigrateContract }

// ValidateBasic validates the proposal
func (p MigrateContractProposal) ValidateBasic() error {
	if err := p.WasmProposal.ValidateBasic(); err != nil {
		return err
	}
	if p.Code == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "code_id is required")
	}
	if err := sdk.VerifyAddressFormat(p.Contract); err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	if err := sdk.VerifyAddressFormat(p.RunAs); err != nil {
		return sdkerrors.Wrap(err, "run as")
	}
	return nil
}

// String implements the Stringer interface.
func (p MigrateContractProposal) String() string {
	return fmt.Sprintf(`Migrate Contract Proposal:
  Title:       %s
  Description: %s
  Contract:    %s
  Code id:     %d
  Run as:      %s
  MigrateMsg   %q
`, p.Title, p.Description, p.Contract, p.Code, p.RunAs, p.MigrateMsg)
}

func (p MigrateContractProposal) MarshalYAML() (interface{}, error) {
	return struct {
		WasmProposal `yaml:",inline"`
		Contract     sdk.AccAddress `yaml:"contract"`
		Code         uint64         `yaml:"code_id"`
		MigrateMsg   string         `yaml:"msg"`
		RunAs        sdk.AccAddress `yaml:"run_as"`
	}{
		WasmProposal: p.WasmProposal,
		Contract:     p.Contract,
		Code:         p.Code,
		MigrateMsg:   string(p.MigrateMsg),
		RunAs:        p.RunAs,
	}, nil
}

type UpdateAdminProposal struct {
	WasmProposal `yaml:",inline"`
	NewAdmin     sdk.AccAddress `json:"new_admin" yaml:"new_admin"`
	Contract     sdk.AccAddress `json:"contract" yaml:"contract"`
}

// ProposalType returns the type
func (p UpdateAdminProposal) ProposalType() string { return ProposalTypeUpdateAdmin }

// ValidateBasic validates the proposal
func (p UpdateAdminProposal) ValidateBasic() error {
	if err := p.WasmProposal.ValidateBasic(); err != nil {
		return err
	}
	if err := sdk.VerifyAddressFormat(p.Contract); err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	if err := sdk.VerifyAddressFormat(p.NewAdmin); err != nil {
		return sdkerrors.Wrap(err, "new admin")
	}
	return nil
}

// String implements the Stringer interface.
func (p UpdateAdminProposal) String() string {
	return fmt.Sprintf(`Update Contract Admin Proposal:
  Title:       %s
  Description: %s
  Contract:    %s
  New Admin:   %s
`, p.Title, p.Description, p.Contract, p.NewAdmin)
}

type ClearAdminProposal struct {
	WasmProposal `yaml:",inline"`

	Contract sdk.AccAddress `json:"contract" yaml:"contract"`
}

// ProposalType returns the type
func (p ClearAdminProposal) ProposalType() string { return ProposalTypeClearAdmin }

// ValidateBasic validates the proposal
func (p ClearAdminProposal) ValidateBasic() error {
	if err := p.WasmProposal.ValidateBasic(); err != nil {
		return err
	}
	if err := sdk.VerifyAddressFormat(p.Contract); err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	return nil
}

// String implements the Stringer interface.
func (p ClearAdminProposal) String() string {
	return fmt.Sprintf(`Clear Contract Admin Proposal:
  Title:       %s
  Description: %s
  Contract:    %s
`, p.Title, p.Description, p.Contract)
}
