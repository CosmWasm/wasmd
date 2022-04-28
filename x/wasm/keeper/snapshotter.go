package keeper

import (
	"encoding/hex"
	"io"

	snapshot "github.com/cosmos/cosmos-sdk/snapshots/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	protoio "github.com/gogo/protobuf/io"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/CosmWasm/wasmd/x/wasm/ioutils"
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

// Format 1 is just gzipped wasm byte code for each item payload. No protobuf envelope, no metadata.
const SnapshotFormat = 1

type WasmSnapshotter struct {
	wasm *Keeper
	cms  sdk.MultiStore
}

func NewWasmSnapshotter(cms sdk.MultiStore, wasm *Keeper) *WasmSnapshotter {
	return &WasmSnapshotter{
		wasm: wasm,
		cms:  cms,
	}
}

func (ws *WasmSnapshotter) SnapshotName() string {
	return types.ModuleName
}

func (ws *WasmSnapshotter) SnapshotFormat() uint32 {
	return SnapshotFormat
}

func (ws *WasmSnapshotter) SupportedFormats() []uint32 {
	// If we support older formats, add them here and handle them in Restore
	return []uint32{SnapshotFormat}
}

func (ws *WasmSnapshotter) Snapshot(height uint64, protoWriter protoio.Writer) error {
	// TODO: This seems more correct (historical info), but kills my tests
	// Since codeIDs and wasm are immutible, it is never wrong to return new wasm data than the
	// user requests
	// ------
	// cacheMS, err := ws.cms.CacheMultiStoreWithVersion(int64(height))
	// if err != nil {
	// 	return err
	// }
	cacheMS := ws.cms.CacheMultiStore()

	ctx := sdk.NewContext(cacheMS, tmproto.Header{}, false, log.NewNopLogger())
	uniqueHashes := make(map[string]bool)
	var rerr error

	ws.wasm.IterateCodeInfos(ctx, func(id uint64, info types.CodeInfo) bool {
		// Many code ids may point to the same code hash... only sync it once
		hexHash := hex.EncodeToString(info.CodeHash)
		// if uniqueHashes, just skip this one and move to the next
		if uniqueHashes[hexHash] {
			return false
		}
		uniqueHashes[hexHash] = true

		// load code and abort on error
		wasmBytes, err := ws.wasm.GetByteCode(ctx, id)
		if err != nil {
			rerr = err
			return true
		}

		compressedWasm, err := ioutils.GzipIt(wasmBytes)
		if err != nil {
			rerr = err
			return true
		}

		err = snapshot.WriteExtensionItem(protoWriter, compressedWasm)
		if err != nil {
			rerr = err
			return true
		}

		return false
	})

	return rerr
}

func (ws *WasmSnapshotter) Restore(
	height uint64, format uint32, protoReader protoio.Reader,
) (snapshot.SnapshotItem, error) {
	if format == 1 {
		return ws.processAllItems(height, protoReader, restoreV1, finalizeV1)
	}
	return snapshot.SnapshotItem{}, snapshot.ErrUnknownFormat
}

func restoreV1(ctx sdk.Context, k *Keeper, compressedCode []byte) error {
	wasmCode, err := ioutils.Uncompress(compressedCode, k.GetMaxWasmCodeSize(ctx))
	if err != nil {
		return sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	}

	// FIXME: check which codeIDs the checksum matches??
	_, err = k.wasmVM.Create(wasmCode)
	if err != nil {
		return sdkerrors.Wrap(types.ErrCreateFailed, err.Error())
	}
	return nil
}

func finalizeV1(ctx sdk.Context, k *Keeper) error {
	// FIXME: ensure all codes have been uploaded?
	return k.InitializePinnedCodes(ctx)
}

func (ws *WasmSnapshotter) processAllItems(
	height uint64,
	protoReader protoio.Reader,
	cb func(sdk.Context, *Keeper, []byte) error,
	finalize func(sdk.Context, *Keeper) error,
) (snapshot.SnapshotItem, error) {
	ctx := sdk.NewContext(ws.cms, tmproto.Header{}, false, log.NewNopLogger())

	// keep the last item here... if we break, it will either be empty (if we hit io.EOF)
	// or contain the last item (if we hit payload == nil)
	var item snapshot.SnapshotItem
	for {
		item = snapshot.SnapshotItem{}
		err := protoReader.ReadMsg(&item)
		if err == io.EOF {
			break
		} else if err != nil {
			return snapshot.SnapshotItem{}, sdkerrors.Wrap(err, "invalid protobuf message")
		}

		// if it is not another ExtensionPayload message, then it is not for us.
		// we should return it an let the manager handle this one
		payload := item.GetExtensionPayload()
		if payload == nil {
			break
		}

		if err := cb(ctx, ws.wasm, payload.Payload); err != nil {
			return snapshot.SnapshotItem{}, sdkerrors.Wrap(err, "processing snapshot item")
		}
	}

	return item, finalize(ctx, ws.wasm)
}
