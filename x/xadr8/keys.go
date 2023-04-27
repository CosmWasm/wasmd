package xadr8

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	SubModuleName = "adr8ibc"
	StoreKey      = SubModuleName
	Version       = "adr8-demo-1"
)

var CallbackKeyPrefix = []byte{0x01}

func BuildCallbackKey(p PacketId) []byte {
	sep := []byte{0xff}
	return append(CallbackKeyPrefix, bytes.Join([][]byte{[]byte(p.PortId), []byte(p.ChannelId), sdk.Uint64ToBigEndian(p.Sequence)}, sep)...)
}
