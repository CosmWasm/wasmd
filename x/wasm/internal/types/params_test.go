package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestValidateParams(t *testing.T) {
	var (
		anyAddress     = make([]byte, sdk.AddrLen)
		invalidAddress = make([]byte, sdk.AddrLen-1)
	)

	specs := map[string]struct {
		src    Params
		expErr bool
	}{
		"all good with defaults": {
			src: DefaultParams(),
		},
		"all good with nobody": {
			src: Params{
				UploadAccess:                 AllowNobody,
				InstantiateDefaultPermission: Nobody,
			},
		},
		"all good with everybody": {
			src: Params{
				UploadAccess:                 AllowEverybody,
				InstantiateDefaultPermission: Everybody,
			},
		},
		"all good with only address": {
			src: Params{
				UploadAccess:                 OnlyAddress.With(anyAddress),
				InstantiateDefaultPermission: OnlyAddress,
			},
		},
		"reject empty type in instantiate permission": {
			src: Params{
				UploadAccess:                 AllowNobody,
				InstantiateDefaultPermission: 0,
			},
			expErr: true,
		},
		"reject unknown type in instantiate": {
			src: Params{
				UploadAccess:                 AllowNobody,
				InstantiateDefaultPermission: 4,
			},
			expErr: true,
		},
		"reject invalid address in only address": {
			src: Params{
				UploadAccess:                 AccessConfig{Type: OnlyAddress, Address: invalidAddress},
				InstantiateDefaultPermission: OnlyAddress,
			},
			expErr: true,
		},
		"reject AccessConfig Everybody with obsolete address": {
			src: Params{
				UploadAccess:                 AccessConfig{Type: Everybody, Address: anyAddress},
				InstantiateDefaultPermission: OnlyAddress,
			},
			expErr: true,
		},
		"reject AccessConfig Nobody with obsolete address": {
			src: Params{
				UploadAccess:                 AccessConfig{Type: Nobody, Address: anyAddress},
				InstantiateDefaultPermission: OnlyAddress,
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
