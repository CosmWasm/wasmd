package wasm

import (
	"encoding/json"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ibcfees "github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

var _ IBCAppVersionDecoder = AppVersionDecoderFn(nil)

// AppVersionDecoderFn custom type that implements IBCAppVersionDecoder
type AppVersionDecoderFn func(ctx sdk.Context, rawVersion, portID, channelID string) (string, error)

// Decode implements IBCAppVersionDecoder.Decode
func (a AppVersionDecoderFn) Decode(ctx sdk.Context, rawVersion, portID, channelID string) (string, error) {
	return a(ctx, rawVersion, portID, channelID)
}

// AppVersionDecoderChain set up a chain of decoders where output of decoder n is feeded into n+1 as raw version
// until the last one is reached
func AppVersionDecoderChain(decoders ...IBCAppVersionDecoder) IBCAppVersionDecoder {
	if len(decoders) == 0 {
		panic("decoders must not be empty")
	}
	return AppVersionDecoderFn(func(ctx sdk.Context, rawVersion, portID, channelID string) (string, error) {
		version := rawVersion
		var err error
		for _, d := range decoders {
			version, err = d.Decode(ctx, version, portID, channelID)
			if err != nil {
				return "", err
			}
		}
		return version, nil
	})
}

// ICS29AppVersionDecoder decodes the ibc app version from an ics-29 fee middleware version when supported
// fallback is raw version when it can not be determined
func ICS29AppVersionDecoder(channelSource interface {
	IsFeeEnabled(ctx sdk.Context, portID, channelID string) bool
},
) IBCAppVersionDecoder {
	return AppVersionDecoderFn(func(ctx sdk.Context, rawVersion, portID, channelID string) (string, error) {
		if channelSource.IsFeeEnabled(ctx, portID, channelID) {
			var meta ibcfees.Metadata
			if err := types.ModuleCdc.UnmarshalJSON([]byte(rawVersion), &meta); err != nil {
				return "", channeltypes.ErrInvalidChannelVersion
			}
			return meta.AppVersion, nil
		}
		if strings.TrimSpace(rawVersion) != "" && json.Valid([]byte(rawVersion)) {
			// check for ics-29 middleware versions
			var versionMetadata ibcfees.Metadata
			if err := types.ModuleCdc.UnmarshalJSON([]byte(rawVersion), &versionMetadata); err == nil {
				if strings.HasPrefix(versionMetadata.FeeVersion, "ics29-") { // sanity check
					return versionMetadata.AppVersion, nil
				}
			}
		}
		return rawVersion, nil
	})
}
