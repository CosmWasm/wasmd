package types

import (
	storetypes "github.com/line/lbm-sdk/store/types"
	wasmvm "github.com/line/wasmvm"
)

var _ wasmvm.KVStore = (*WasmStore)(nil)

// WasmStore is a wrapper struct of `KVStore`
// It translates from cosmos KVStore to wasmvm-defined KVStore.
// The spec of interface `Iterator` is a bit different so we cannot use cosmos KVStore directly.
type WasmStore struct {
	storetypes.KVStore
}

// Iterator re-define for wasmvm's `Iterator`
func (s WasmStore) Iterator(start, end []byte) wasmvm.Iterator {
	return s.KVStore.Iterator(start, end)
}

// ReverseIterator re-define for wasmvm's `Iterator`
func (s WasmStore) ReverseIterator(start, end []byte) wasmvm.Iterator {
	return s.KVStore.ReverseIterator(start, end)
}

// NewWasmStore creates a instance of WasmStore
func NewWasmStore(kvStore storetypes.KVStore) WasmStore {
	return WasmStore{kvStore}
}
