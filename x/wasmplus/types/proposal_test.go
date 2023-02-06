package types

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/line/lbm-sdk/types"

	wasmtypes "github.com/line/wasmd/x/wasm/types"
)

func TestValidateDeactivateContractProposal(t *testing.T) {
	var anyAddress sdk.AccAddress = bytes.Repeat([]byte{0x0}, wasmtypes.ContractAddrLen)

	specs := map[string]struct {
		src    DeactivateContractProposal
		expErr bool
	}{
		"all good": {
			src: DeactivateContractProposal{
				Title:       "Foo",
				Description: "Bar",
				Contract:    anyAddress.String(),
			},
		},
		"invalid address": {
			src: DeactivateContractProposal{
				Title:       "Foo",
				Description: "Bar",
				Contract:    "invalid_address",
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

func TestValidateActivateContractProposal(t *testing.T) {
	var anyAddress sdk.AccAddress = bytes.Repeat([]byte{0x0}, wasmtypes.ContractAddrLen)

	specs := map[string]struct {
		src    ActivateContractProposal
		expErr bool
	}{
		"all good": {
			src: ActivateContractProposal{
				Title:       "Foo",
				Description: "Bar",
				Contract:    anyAddress.String(),
			},
		},
		"invalid address": {
			src: ActivateContractProposal{
				Title:       "Foo",
				Description: "Bar",
				Contract:    "invalid_address",
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
