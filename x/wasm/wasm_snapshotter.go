package wasm

import (
	"io"

	snapshot "github.com/cosmos/cosmos-sdk/snapshots/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	protoio "github.com/gogo/protobuf/io"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

/*
API to implement:

// Snapshotter is something that can create and restore snapshots, consisting of streamed binary
// chunks - all of which must be read from the channel and closed. If an unsupported format is
// given, it must return ErrUnknownFormat (possibly wrapped with fmt.Errorf).
type Snapshotter interface {
	// Snapshot writes snapshot items into the protobuf writer.
	Snapshot(height uint64, protoWriter protoio.Writer) error

	// Restore restores a state snapshot from the protobuf items read from the reader.
	// If the ready channel is non-nil, it returns a ready signal (by being closed) once the
	// restorer is ready to accept chunks.
	Restore(height uint64, format uint32, protoReader protoio.Reader) (SnapshotItem, error)
}

// ExtensionSnapshotter is an extension Snapshotter that is appended to the snapshot stream.
// ExtensionSnapshotter has an unique name and manages it's own internal formats.
type ExtensionSnapshotter interface {
	Snapshotter

	// SnapshotName returns the name of snapshotter, it should be unique in the manager.
	SnapshotName() string

	// SnapshotFormat returns the default format the extension snapshotter use to encode the
	// payloads when taking a snapshot.
	// It's defined within the extension, different from the global format for the whole state-sync snapshot.
	SnapshotFormat() uint32

	// SupportedFormats returns a list of formats it can restore from.
	SupportedFormats() []uint32
}
*/

type WasmSnapshotter struct {
	wasm keeper.Keeper
	cms  sdk.CommitMultiStore
}

func NewWasmSnapshotter(cms sdk.CommitMultiStore, wasm keeper.Keeper) *WasmSnapshotter {
	return &WasmSnapshotter{
		wasm: wasm,
		cms:  cms,
	}
}

func (ws *WasmSnapshotter) SnapshotName() string {
	return "WASM Code Snapshot"
}

func (ws *WasmSnapshotter) SnapshotFormat() uint32 {
	return 1
}

func (ws *WasmSnapshotter) SupportedFormats() []uint32 {
	return []uint32{1}
}

func (ws *WasmSnapshotter) Snapshot(height uint64, protoWriter protoio.Writer) error {
	var rerr error
	ctx := sdk.NewContext(ws.cms, tmproto.Header{}, false, log.NewNopLogger())

	ws.wasm.IterateCodeInfos(ctx, func(id uint64, info types.CodeInfo) bool {
		// load code and abort on error
		wasmBytes, err := ws.wasm.GetByteCode(ctx, id)
		if err != nil {
			rerr = err
			return true
		}

		// TODO: compress wasm bytes
		// TODO: embed in a protobuf message with more data
		snapshot.WriteExtensionItem(protoWriter, wasmBytes)

		return false
	})

	return rerr
}

func (ws *WasmSnapshotter) Restore(
	height uint64, format uint32, protoReader protoio.Reader,
) (snapshot.SnapshotItem, error) {
	if format == 1 {
		err := ws.processAllItems(height, protoReader, restoreV1)
		return snapshot.SnapshotItem{}, err
	}
	return snapshot.SnapshotItem{}, snapshot.ErrUnknownFormat
}

func restoreV1(ctx sdk.Context, k keeper.Keeper, payload []byte) error {
	panic("TODO")
	// wasmCode, err = uncompress(wasmCode, k.GetMaxWasmCodeSize(ctx))
	// if err != nil {
	// 	return 0, sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	// }
	// checksum, err := k.wasmVM.Create(wasmCode)
	// if err != nil {
	// 	return 0, sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	// }
}

func (ws *WasmSnapshotter) processAllItems(
	height uint64, protoReader protoio.Reader, cb func(sdk.Context, keeper.Keeper, []byte) error,
) error {
	ctx := sdk.NewContext(ws.cms, tmproto.Header{}, false, log.NewNopLogger())

	for {
		item := &snapshot.SnapshotItem{}
		err := protoReader.ReadMsg(item)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return sdkerrors.Wrap(err, "invalid protobuf message")
		}

		payload := item.GetExtensionPayload()
		if payload == nil {
			return sdkerrors.Wrap(err, "invalid protobuf message")
		}

		if err := cb(ctx, ws.wasm, payload.Payload); err != nil {
			return sdkerrors.Wrap(err, "processing snapshot item")
		}
	}
}

// 		// snapshotItem is 64 bytes of the file name, then the actual WASM bytes
// 		if len(payload.Payload) < 64 {
// 			return snapshot.SnapshotItem{}, sdkerrors.Wrapf(err, "wasm snapshot must be at least 64 bytes, got %v bytes", len(payload.Payload))
// 		}

// 		wasmBytes := payload.Payload[64:]

// 		err = ioutil.WriteFile(wasmFilePath, wasmBytes, 0664 /* -rw-rw-r-- */)
// 		if err != nil {
// 			return snapshot.SnapshotItem{}, sdkerrors.Wrapf(err, "failed to write wasm file '%v' to disk", wasmFilePath)
// 		}
// 	}

// 	return snapshot.SnapshotItem{}, nil
// }
