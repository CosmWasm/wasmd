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

var (
	CallbackKeyPrefix = []byte{0x01}
	separator         = []byte{0xff}
)

func BuildCallbackKey(p PacketId, actor string) []byte {
	return bytes.Join([][]byte{BuildCallbackKeyPrefix(p), []byte(actor)}, separator)
}

func BuildCallbackKeyPrefix(p PacketId) []byte {
	return append(CallbackKeyPrefix, bytes.Join([][]byte{[]byte(p.PortId), []byte(p.ChannelId), sdk.Uint64ToBigEndian(p.Sequence)}, separator)...)
}

// prefixRange turns a prefix into (start, end) to create
// and iterator
// copied from: https://github.com/iov-one/weave/blob/74fafaa757cbd510a61cc9e8b32ad8c9185d0d2e/orm/query.go#L32
func PrefixRange(prefix []byte) ([]byte, []byte) {
	// special case: no prefix is whole range
	if len(prefix) == 0 {
		return nil, nil
	}

	// copy the prefix and update last byte
	end := make([]byte, len(prefix))
	copy(end, prefix)
	l := len(end) - 1
	end[l]++

	// wait, what if that overflowed?....
	for end[l] == 0 && l > 0 {
		l--
		end[l]++
	}

	// okay, funny guy, you gave us FFF, no end to this range...
	if l == 0 && end[0] == 0 {
		end = nil
	}
	return prefix, end
}
