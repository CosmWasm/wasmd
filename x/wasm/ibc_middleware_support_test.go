package wasm

import (
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppVersionDecoderChain(t *testing.T) {
	dropLastCharDec := AppVersionDecoderFn(func(_ sdk.Context, rawVersion, _, _ string) (string, error) {
		return rawVersion[0 : len(rawVersion)-1], nil
	})
	alwaysErrDec := AppVersionDecoderFn(func(_ sdk.Context, rawVersion, _, _ string) (string, error) {
		return "", errors.New("testing")
	})
	specs := map[string]struct {
		dec        IBCAppVersionDecoder
		rawVersion string
		expVersion string
		expErr     bool
	}{
		"single decoder": {
			dec:        AppVersionDecoderChain(ICS29AppVersionDecoder(IsFeeEnabledMock{true})),
			rawVersion: `{"fee_version":"ics29-1", "app_version":"my version"}`,
			expVersion: "my version",
		},
		"multiple decoders": {
			dec:        AppVersionDecoderChain(ICS29AppVersionDecoder(IsFeeEnabledMock{true}), dropLastCharDec, dropLastCharDec),
			rawVersion: `{"fee_version":"ics29-1", "app_version":"123"}`,
			expVersion: "1",
		},
		"multiple decoders with err": {
			dec:    AppVersionDecoderChain(ICS29AppVersionDecoder(IsFeeEnabledMock{true}), alwaysErrDec, dropLastCharDec),
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotVersion, gotErr := spec.dec.Decode(sdk.Context{}, spec.rawVersion, "foo", "bar")
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expVersion, gotVersion)
		})
	}
}

func TestICS29AppVersionDecoder(t *testing.T) {
	specs := map[string]struct {
		rawVersion   string
		isFeeChannel bool
		expVersion   string
		expErr       bool
	}{
		"raw version": {
			rawVersion: "my version",
			expVersion: "my version",
		},
		"ics29 version on fee channel": {
			rawVersion:   `{"fee_version":"ics29-1", "app_version":"my version"}`,
			isFeeChannel: true,
			expVersion:   "my version",
		},
		"invalid ics29 version on fee channel": {
			rawVersion:   `not-a-json-string`,
			isFeeChannel: true,
			expErr:       true,
		},
		"ics29 version on non fee channel": {
			rawVersion: `{"fee_version":"ics29-1", "app_version":"my version"}`,
			expVersion: "my version",
		},
		"non ics29 version on non fee channel": {
			rawVersion: `{"fee_version":"alx29-1", "app_version":"my version"}`,
			expVersion: `{"fee_version":"alx29-1", "app_version":"my version"}`,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotVersion, gotErr := ICS29AppVersionDecoder(IsFeeEnabledMock{spec.isFeeChannel}).
				Decode(sdk.Context{}, spec.rawVersion, "foo", "bar")
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expVersion, gotVersion)
		})
	}
}

type IsFeeEnabledMock struct {
	result bool
}

func (f IsFeeEnabledMock) IsFeeEnabled(_ sdk.Context, _, _ string) bool {
	return f.result
}
