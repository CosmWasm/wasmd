package ibctesting

import (
	"encoding/hex"
	"strconv"
	"strings"

	clienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

func getSendPackets(evts []abci.Event) []channeltypes.Packet {
	var res []channeltypes.Packet
	for _, evt := range evts {
		if evt.Type == channeltypes.EventTypeSendPacket {
			packet := parsePacketFromEvent(evt)
			res = append(res, packet)
		}
	}
	return res
}

// Used for various debug statements above when needed... do not remove
// func showEvent(evt abci.Event) {
//	fmt.Printf("evt.Type: %s\n", evt.Type)
//	for _, attr := range evt.Attributes {
//		fmt.Printf("  %s = %s\n", string(attr.Key), string(attr.Value))
//	}
//}

func parsePacketFromEvent(evt abci.Event) channeltypes.Packet {
	return channeltypes.Packet{
		Sequence:           getUintField(evt, channeltypes.AttributeKeySequence),
		SourcePort:         getField(evt, channeltypes.AttributeKeySrcPort),
		SourceChannel:      getField(evt, channeltypes.AttributeKeySrcChannel),
		DestinationPort:    getField(evt, channeltypes.AttributeKeyDstPort),
		DestinationChannel: getField(evt, channeltypes.AttributeKeyDstChannel),
		Data:               getHexField(evt, channeltypes.AttributeKeyDataHex),
		TimeoutHeight:      parseTimeoutHeight(getField(evt, channeltypes.AttributeKeyTimeoutHeight)),
		TimeoutTimestamp:   getUintField(evt, channeltypes.AttributeKeyTimeoutTimestamp),
	}
}

func getHexField(evt abci.Event, key string) []byte {
	got := getField(evt, key)
	if got == "" {
		return nil
	}
	bz, err := hex.DecodeString(got)
	if err != nil {
		panic(err)
	}
	return bz
}

// return the value for the attribute with the given name
func getField(evt abci.Event, key string) string {
	for _, attr := range evt.Attributes {
		if string(attr.Key) == key {
			return string(attr.Value)
		}
	}
	return ""
}

func getUintField(evt abci.Event, key string) uint64 {
	raw := getField(evt, key)
	return toUint64(raw)
}

func toUint64(raw string) uint64 {
	if raw == "" {
		return 0
	}
	i, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		panic(err)
	}
	return i
}

func parseTimeoutHeight(raw string) clienttypes.Height {
	chunks := strings.Split(raw, "-")
	return clienttypes.Height{
		RevisionNumber: toUint64(chunks[0]),
		RevisionHeight: toUint64(chunks[1]),
	}
}
