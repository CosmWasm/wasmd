package wasm

import (
	"testing"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type mockICS4Wrapper struct {
	porttypes.ICS4Wrapper
	sentData []byte
}

func (m *mockICS4Wrapper) SendPacket(_ sdk.Context, _, _ string, _ clienttypes.Height, _ uint64, data []byte) (uint64, error) {
	m.sentData = data
	return 1, nil
}

func TestIBCDedupMiddlewareOnRecvPacket(t *testing.T) {
	specs := map[string]struct {
		memo    map[string]any
		rawData []byte
		expFail bool
	}{
		"wasm + dest_callback (same dest side) rejected": {
			memo: map[string]any{
				"wasm":          map[string]any{"contract": "cosmos1ccc", "msg": map[string]any{}},
				"dest_callback": map[string]any{"address": "cosmos1ccc"},
			},
			expFail: true,
		},
		"wasm + src_callback (cross-side) passes through": {
			memo: map[string]any{
				"wasm":         map[string]any{"contract": "cosmos1ccc", "msg": map[string]any{}},
				"src_callback": map[string]any{"address": "cosmos1ccc"},
			},
		},
		"dest_callback alone passes through":  {memo: map[string]any{"dest_callback": map[string]any{"address": "cosmos1ccc"}}},
		"non-transfer payload passes through": {rawData: []byte("not-a-transfer-packet")},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			inner := &recordingIBCModule{}
			m := NewIBCDedupMiddleware(inner, &mockICS4Wrapper{})

			data := spec.rawData
			if data == nil {
				data = transferPacketFixture(mustMarshalJSON(t, spec.memo)).Data
			}
			pkt := channeltypes.Packet{
				Sequence: 1, SourcePort: "transfer", SourceChannel: "channel-0",
				DestinationPort: "transfer", DestinationChannel: "channel-1",
				Data:          data,
				TimeoutHeight: clienttypes.Height{RevisionHeight: 100},
			}

			gotAck := m.OnRecvPacket(sdk.Context{}, "ics20-1", pkt, sdk.AccAddress("relayer"))
			require.NotNil(t, gotAck)

			if spec.expFail {
				assert.False(t, gotAck.Success())
				assert.Nil(t, inner.received)
				return
			}
			assert.Equal(t, data, inner.received)
		})
	}
}

func TestIBCDedupMiddlewareSendPacket(t *testing.T) {
	specs := map[string]struct {
		memo    map[string]any
		rawData []byte
		expFail bool
	}{
		"ibc_callback + src_callback (same src side) rejected": {
			memo: map[string]any{
				"ibc_callback": "cosmos1ccc",
				"src_callback": map[string]any{"address": "cosmos1ccc"},
			},
			expFail: true,
		},
		"wasm + src_callback (cross-side) passes through": {
			memo: map[string]any{
				"wasm":         map[string]any{"contract": "cosmos1ccc", "msg": map[string]any{}},
				"src_callback": map[string]any{"address": "cosmos1ccc"},
			},
		},
		"src_callback alone passes through":   {memo: map[string]any{"src_callback": map[string]any{"address": "cosmos1ccc"}}},
		"non-transfer payload passes through": {rawData: []byte("not-a-transfer-packet")},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			ics4 := &mockICS4Wrapper{}
			m := NewIBCDedupMiddleware(&recordingIBCModule{}, ics4)

			data := spec.rawData
			if data == nil {
				data = transferPacketFixture(mustMarshalJSON(t, spec.memo)).Data
			}

			_, gotErr := m.SendPacket(sdk.Context{}, "transfer", "channel-0",
				clienttypes.Height{RevisionHeight: 100}, 0, data)

			if spec.expFail {
				require.Error(t, gotErr)
				assert.Nil(t, ics4.sentData)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, data, ics4.sentData)
		})
	}
}

func transferPacketFixture(memo string) channeltypes.Packet {
	td := transfertypes.NewFungibleTokenPacketData("uosmo", "1000", "cosmos1sender", "cosmos1receiver", memo)
	return channeltypes.Packet{
		Sequence:           1,
		SourcePort:         "transfer",
		SourceChannel:      "channel-0",
		DestinationPort:    "transfer",
		DestinationChannel: "channel-1",
		Data:               td.GetBytes(),
		TimeoutHeight:      clienttypes.Height{RevisionHeight: 100},
	}
}
