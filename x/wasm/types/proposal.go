package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

type ProposalType string

const (
	ProposalTypeStoreCode           ProposalType = "StoreCode"
	ProposalTypeInstantiateContract ProposalType = "InstantiateContract"
	ProposalTypeMigrateContract     ProposalType = "MigrateContract"
	ProposalTypeSudoContract        ProposalType = "SudoContract"
	ProposalTypeExecuteContract     ProposalType = "ExecuteContract"
	ProposalTypeUpdateAdmin         ProposalType = "UpdateAdmin"
	ProposalTypeClearAdmin          ProposalType = "ClearAdmin"
	ProposalTypePinCodes            ProposalType = "PinCodes"
	ProposalTypeUnpinCodes          ProposalType = "UnpinCodes"
)

// DisableAllProposals contains no wasm gov types.
var DisableAllProposals []ProposalType

// EnableAllProposals contains all wasm gov types as keys.
var EnableAllProposals = []ProposalType{
	ProposalTypeStoreCode,
	ProposalTypeInstantiateContract,
	ProposalTypeMigrateContract,
	ProposalTypeSudoContract,
	ProposalTypeExecuteContract,
	ProposalTypeUpdateAdmin,
	ProposalTypeClearAdmin,
	ProposalTypePinCodes,
	ProposalTypeUnpinCodes,
}

// ConvertToProposals maps each key to a ProposalType and returns a typed list.
// If any string is not a valid type (in this file), then return an error
func ConvertToProposals(keys []string) ([]ProposalType, error) {
	valid := make(map[string]bool, len(EnableAllProposals))
	for _, key := range EnableAllProposals {
		valid[string(key)] = true
	}

	proposals := make([]ProposalType, len(keys))
	for i, key := range keys {
		if _, ok := valid[key]; !ok {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "'%s' is not a valid ProposalType", key)
		}
		proposals[i] = ProposalType(key)
	}
	return proposals, nil
}

func init() { // register new content types with the sdk
	govtypesv1beta1.RegisterProposalType(string(ProposalTypeStoreCode))
	govtypesv1beta1.RegisterProposalType(string(ProposalTypeInstantiateContract))
	govtypesv1beta1.RegisterProposalType(string(ProposalTypeMigrateContract))
	govtypesv1beta1.RegisterProposalType(string(ProposalTypeSudoContract))
	govtypesv1beta1.RegisterProposalType(string(ProposalTypeExecuteContract))
	govtypesv1beta1.RegisterProposalType(string(ProposalTypeUpdateAdmin))
	govtypesv1beta1.RegisterProposalType(string(ProposalTypeClearAdmin))
	govtypesv1beta1.RegisterProposalType(string(ProposalTypePinCodes))
	govtypesv1beta1.RegisterProposalType(string(ProposalTypeUnpinCodes))
}

// ProposalRoute returns the routing key of a parameter change proposal.
func (p MigrateContractProposal) ProposalRoute() string { return RouterKey }

// GetTitle returns the title of the proposal
func (p *MigrateContractProposal) GetTitle() string { return p.Title }

// GetDescription returns the human readable description of the proposal
func (p MigrateContractProposal) GetDescription() string { return p.Description }

// ProposalType returns the type
func (p MigrateContractProposal) ProposalType() string { return string(ProposalTypeMigrateContract) }

// ValidateBasic validates the proposal
func (p MigrateContractProposal) ValidateBasic() error {
	if err := validateProposalCommons(p.Title, p.Description); err != nil {
		return err
	}
	if p.CodeID == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "code_id is required")
	}
	if _, err := sdk.AccAddressFromBech32(p.Contract); err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	if err := p.Msg.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "payload msg")
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
  Msg:         %q
`, p.Title, p.Description, p.Contract, p.CodeID, p.Msg)
}

// MarshalYAML pretty prints the migrate message
func (p MigrateContractProposal) MarshalYAML() (interface{}, error) {
	return struct {
		Title       string `yaml:"title"`
		Description string `yaml:"description"`
		Contract    string `yaml:"contract"`
		CodeID      uint64 `yaml:"code_id"`
		Msg         string `yaml:"msg"`
	}{
		Title:       p.Title,
		Description: p.Description,
		Contract:    p.Contract,
		CodeID:      p.CodeID,
		Msg:         string(p.Msg),
	}, nil
}

// ProposalRoute returns the routing key of a parameter change proposal.
func (p SudoContractProposal) ProposalRoute() string { return RouterKey }

// GetTitle returns the title of the proposal
func (p *SudoContractProposal) GetTitle() string { return p.Title }

// GetDescription returns the human readable description of the proposal
func (p SudoContractProposal) GetDescription() string { return p.Description }

// ProposalType returns the type
func (p SudoContractProposal) ProposalType() string { return string(ProposalTypeSudoContract) }

// ValidateBasic validates the proposal
func (p SudoContractProposal) ValidateBasic() error {
	if err := validateProposalCommons(p.Title, p.Description); err != nil {
		return err
	}
	if _, err := sdk.AccAddressFromBech32(p.Contract); err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	if err := p.Msg.ValidateBasic(); err != nil {
		return sdkerrors.Wrap(err, "payload msg")
	}
	return nil
}

// String implements the Stringer interface.
func (p SudoContractProposal) String() string {
	return fmt.Sprintf(`Migrate Contract Proposal:
  Title:       %s
  Description: %s
  Contract:    %s
  Msg:         %q
`, p.Title, p.Description, p.Contract, p.Msg)
}

// MarshalYAML pretty prints the migrate message
func (p SudoContractProposal) MarshalYAML() (interface{}, error) {
	return struct {
		Title       string `yaml:"title"`
		Description string `yaml:"description"`
		Contract    string `yaml:"contract"`
		Msg         string `yaml:"msg"`
	}{
		Title:       p.Title,
		Description: p.Description,
		Contract:    p.Contract,
		Msg:         string(p.Msg),
	}, nil
}

// ProposalRoute returns the routing key of a parameter change proposal.
func (p UpdateAdminProposal) ProposalRoute() string { return RouterKey }

// GetTitle returns the title of the proposal
func (p *UpdateAdminProposal) GetTitle() string { return p.Title }

// GetDescription returns the human readable description of the proposal
func (p UpdateAdminProposal) GetDescription() string { return p.Description }

// ProposalType returns the type
func (p UpdateAdminProposal) ProposalType() string { return string(ProposalTypeUpdateAdmin) }

// ValidateBasic validates the proposal
func (p UpdateAdminProposal) ValidateBasic() error {
	if err := validateProposalCommons(p.Title, p.Description); err != nil {
		return err
	}
	if _, err := sdk.AccAddressFromBech32(p.Contract); err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	if _, err := sdk.AccAddressFromBech32(p.NewAdmin); err != nil {
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

// ProposalRoute returns the routing key of a parameter change proposal.
func (p ClearAdminProposal) ProposalRoute() string { return RouterKey }

// GetTitle returns the title of the proposal
func (p *ClearAdminProposal) GetTitle() string { return p.Title }

// GetDescription returns the human readable description of the proposal
func (p ClearAdminProposal) GetDescription() string { return p.Description }

// ProposalType returns the type
func (p ClearAdminProposal) ProposalType() string { return string(ProposalTypeClearAdmin) }

// ValidateBasic validates the proposal
func (p ClearAdminProposal) ValidateBasic() error {
	if err := validateProposalCommons(p.Title, p.Description); err != nil {
		return err
	}
	if _, err := sdk.AccAddressFromBech32(p.Contract); err != nil {
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

// ProposalRoute returns the routing key of a parameter change proposal.
func (p PinCodesProposal) ProposalRoute() string { return RouterKey }

// GetTitle returns the title of the proposal
func (p *PinCodesProposal) GetTitle() string { return p.Title }

// GetDescription returns the human readable description of the proposal
func (p PinCodesProposal) GetDescription() string { return p.Description }

// ProposalType returns the type
func (p PinCodesProposal) ProposalType() string { return string(ProposalTypePinCodes) }

// ValidateBasic validates the proposal
func (p PinCodesProposal) ValidateBasic() error {
	if err := validateProposalCommons(p.Title, p.Description); err != nil {
		return err
	}
	if len(p.CodeIDs) == 0 {
		return sdkerrors.Wrap(ErrEmpty, "code ids")
	}
	return nil
}

// String implements the Stringer interface.
func (p PinCodesProposal) String() string {
	return fmt.Sprintf(`Pin Wasm Codes Proposal:
  Title:       %s
  Description: %s
  Codes:       %v
`, p.Title, p.Description, p.CodeIDs)
}

// ProposalRoute returns the routing key of a parameter change proposal.
func (p UnpinCodesProposal) ProposalRoute() string { return RouterKey }

// GetTitle returns the title of the proposal
func (p *UnpinCodesProposal) GetTitle() string { return p.Title }

// GetDescription returns the human readable description of the proposal
func (p UnpinCodesProposal) GetDescription() string { return p.Description }

// ProposalType returns the type
func (p UnpinCodesProposal) ProposalType() string { return string(ProposalTypeUnpinCodes) }

// ValidateBasic validates the proposal
func (p UnpinCodesProposal) ValidateBasic() error {
	if err := validateProposalCommons(p.Title, p.Description); err != nil {
		return err
	}
	if len(p.CodeIDs) == 0 {
		return sdkerrors.Wrap(ErrEmpty, "code ids")
	}
	return nil
}

// String implements the Stringer interface.
func (p UnpinCodesProposal) String() string {
	return fmt.Sprintf(`Unpin Wasm Codes Proposal:
  Title:       %s
  Description: %s
  Codes:       %v
`, p.Title, p.Description, p.CodeIDs)
}

func validateProposalCommons(title, description string) error {
	if strings.TrimSpace(title) != title {
		return sdkerrors.Wrap(govtypes.ErrInvalidProposalContent, "proposal title must not start/end with white spaces")
	}
	if len(title) == 0 {
		return sdkerrors.Wrap(govtypes.ErrInvalidProposalContent, "proposal title cannot be blank")
	}
	if len(title) > govtypesv1beta1.MaxTitleLength {
		return sdkerrors.Wrapf(govtypes.ErrInvalidProposalContent, "proposal title is longer than max length of %d", govtypesv1beta1.MaxTitleLength)
	}
	if strings.TrimSpace(description) != description {
		return sdkerrors.Wrap(govtypes.ErrInvalidProposalContent, "proposal description must not start/end with white spaces")
	}
	if len(description) == 0 {
		return sdkerrors.Wrap(govtypes.ErrInvalidProposalContent, "proposal description cannot be blank")
	}
	if len(description) > govtypesv1beta1.MaxDescriptionLength {
		return sdkerrors.Wrapf(govtypes.ErrInvalidProposalContent, "proposal description is longer than max length of %d", govtypesv1beta1.MaxDescriptionLength)
	}
	return nil
}
