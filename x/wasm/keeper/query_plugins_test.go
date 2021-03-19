package keeper

import (
	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
	"github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIBCQuerier(t *testing.T) {
	myExampleChannels := []channeltypes.IdentifiedChannel{
		{
			State:    channeltypes.OPEN,
			Ordering: channeltypes.ORDERED,
			Counterparty: channeltypes.Counterparty{
				PortId:    "counterPartyPortID",
				ChannelId: "counterPartyChannelID",
			},
			ConnectionHops: []string{"one"},
			Version:        "v1",
			PortId:         "myPortID",
			ChannelId:      "myChannelID",
		},
		{
			State:    channeltypes.INIT,
			Ordering: channeltypes.UNORDERED,
			Counterparty: channeltypes.Counterparty{
				PortId:    "otherCounterPartyPortID",
				ChannelId: "otherCounterPartyChannelID",
			},
			ConnectionHops: []string{"other", "second"},
			Version:        "otherVersion",
			PortId:         "otherPortID",
			ChannelId:      "otherChannelID",
		},
	}
	specs := map[string]struct {
		srcQuery      *wasmvmtypes.IBCQuery
		wasmKeeper    *wasmKeeperMock
		channelKeeper *wasmtesting.MockChannelKeeper
		expJsonResult string
		expErr        *sdkerrors.Error
	}{
		"query port id": {
			srcQuery: &wasmvmtypes.IBCQuery{
				PortID: &wasmvmtypes.PortIDQuery{},
			},
			wasmKeeper: newWasmKeeperMock(
				func(ctx sdk.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
					return &types.ContractInfo{IBCPortID: "myIBCPortID"}
				},
			),
			channelKeeper: &wasmtesting.MockChannelKeeper{},
			expJsonResult: `{"port_id":"myIBCPortID"}`,
		},
		"query list channels - all": {
			srcQuery: &wasmvmtypes.IBCQuery{
				ListChannels: &wasmvmtypes.ListChannelsQuery{},
			},
			channelKeeper: &wasmtesting.MockChannelKeeper{
				IterateChannelsFn: wasmtesting.MockChannelKeeperIterator(myExampleChannels),
			},
			expJsonResult: `{
  "channels": [
    {
      "endpoint": {
        "port_id": "myPortID",
        "channel_id": "myChannelID"
      },
      "counterparty_endpoint": {
        "port_id": "counterPartyPortID",
        "channel_id": "counterPartyChannelID"
      },
      "order": "ORDER_ORDERED",
      "version": "v1",
      "connection_id": "one"
    },
    {
      "endpoint": {
        "port_id": "otherPortID",
        "channel_id": "otherChannelID"
      },
      "counterparty_endpoint": {
        "port_id": "otherCounterPartyPortID",
        "channel_id": "otherCounterPartyChannelID"
      },
      "order": "ORDER_UNORDERED",
      "version": "otherVersion",
      "connection_id": "other"
    }
  ]
}`,
		},
		"query list channels - filtered": {
			srcQuery: &wasmvmtypes.IBCQuery{
				ListChannels: &wasmvmtypes.ListChannelsQuery{
					PortID: "otherPortID",
				},
			},
			channelKeeper: &wasmtesting.MockChannelKeeper{
				IterateChannelsFn: wasmtesting.MockChannelKeeperIterator(myExampleChannels),
			},
			expJsonResult: `{
  "channels": [
    {
      "endpoint": {
        "port_id": "otherPortID",
        "channel_id": "otherChannelID"
      },
      "counterparty_endpoint": {
        "port_id": "otherCounterPartyPortID",
        "channel_id": "otherCounterPartyChannelID"
      },
      "order": "ORDER_UNORDERED",
      "version": "otherVersion",
      "connection_id": "other"
    }
  ]
}`,
		},
		"query list channels - filtered empty": {
			srcQuery: &wasmvmtypes.IBCQuery{
				ListChannels: &wasmvmtypes.ListChannelsQuery{
					PortID: "none-existing",
				},
			},
			channelKeeper: &wasmtesting.MockChannelKeeper{
				IterateChannelsFn: wasmtesting.MockChannelKeeperIterator(myExampleChannels),
			},
			expJsonResult: `{"channels": []}`,
		},
		"query channel": {
			srcQuery: &wasmvmtypes.IBCQuery{
				Channel: &wasmvmtypes.ChannelQuery{
					PortID:    "myQueryPortID",
					ChannelID: "myQueryChannelID",
				},
			},
			channelKeeper: &wasmtesting.MockChannelKeeper{
				GetChannelFn: func(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool) {
					return channeltypes.Channel{
						State:    channeltypes.INIT,
						Ordering: channeltypes.UNORDERED,
						Counterparty: channeltypes.Counterparty{
							PortId:    "counterPartyPortID",
							ChannelId: "otherCounterPartyChannelID",
						},
						ConnectionHops: []string{"one"},
						Version:        "version",
					}, true
				},
			},
			expJsonResult: `{
  "channel": {
    "endpoint": {
      "port_id": "myQueryPortID",
      "channel_id": "myQueryChannelID"
    },
    "counterparty_endpoint": {
      "port_id": "counterPartyPortID",
      "channel_id": "otherCounterPartyChannelID"
    },
    "order": "ORDER_UNORDERED",
    "version": "version",
    "connection_id": "one"
  }
}`,
		},
		"query channel - without port set": {
			srcQuery: &wasmvmtypes.IBCQuery{
				Channel: &wasmvmtypes.ChannelQuery{
					ChannelID: "myQueryChannelID",
				},
			},
			wasmKeeper: newWasmKeeperMock(func(ctx sdk.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
				return &types.ContractInfo{IBCPortID: "myLoadedPortID"}
			}),
			channelKeeper: &wasmtesting.MockChannelKeeper{
				GetChannelFn: func(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool) {
					return channeltypes.Channel{
						State:    channeltypes.INIT,
						Ordering: channeltypes.UNORDERED,
						Counterparty: channeltypes.Counterparty{
							PortId:    "counterPartyPortID",
							ChannelId: "otherCounterPartyChannelID",
						},
						ConnectionHops: []string{"one"},
						Version:        "version",
					}, true
				},
			},
			expJsonResult: `{
  "channel": {
    "endpoint": {
      "port_id": "myLoadedPortID",
      "channel_id": "myQueryChannelID"
    },
    "counterparty_endpoint": {
      "port_id": "counterPartyPortID",
      "channel_id": "otherCounterPartyChannelID"
    },
    "order": "ORDER_UNORDERED",
    "version": "version",
    "connection_id": "one"
  }
}`,
		},
		"query channel - empty result": {
			srcQuery: &wasmvmtypes.IBCQuery{
				Channel: &wasmvmtypes.ChannelQuery{
					PortID:    "myQueryPortID",
					ChannelID: "myQueryChannelID",
				},
			},
			channelKeeper: &wasmtesting.MockChannelKeeper{
				GetChannelFn: func(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool) {
					return channeltypes.Channel{}, false
				},
			},
			expJsonResult: "{}",
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			h := IBCQuerier(spec.wasmKeeper, spec.channelKeeper)
			gotResult, gotErr := h(sdk.Context{}, RandomAccountAddress(t), spec.srcQuery)
			require.True(t, spec.expErr.Is(gotErr), "exp %v but got %#+v", spec.expErr, gotErr)
			if spec.expErr != nil {
				return
			}
			assert.JSONEq(t, spec.expJsonResult, string(gotResult), string(gotResult))
		})
	}

}

type wasmKeeperMock struct {
	GetContractInfoFn func(ctx sdk.Context, contractAddress sdk.AccAddress) *types.ContractInfo
}

func newWasmKeeperMock(f func(ctx sdk.Context, contractAddress sdk.AccAddress) *types.ContractInfo) *wasmKeeperMock {
	return &wasmKeeperMock{GetContractInfoFn: f}
}

func (m wasmKeeperMock) GetContractInfo(ctx sdk.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
	if m.GetContractInfoFn == nil {
		panic("not expected to be called")
	}
	return m.GetContractInfoFn(ctx, contractAddress)
}
