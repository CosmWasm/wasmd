// Package xwasmvm is an extended wasmvm api for testing with mock impl only
package xwasmvm

import (
	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
)

type (
	IBCPacketAckedMsg    = wasmvmtypes.IBCPacketAckMsg
	IBCPacketTimedOutMsg = wasmvmtypes.IBCPacketTimeoutMsg
)

// var _ types.WasmerEngine = &XVM{}

type XVM struct {
	*wasmvm.VM
	OnIBCPacketAckedFn    func(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg IBCPacketAckedMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error)
	OnIBCPacketTimedOutFn func(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg IBCPacketTimedOutMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error)
}

func (m XVM) OnIBCPacketAcked(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg IBCPacketAckedMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
	if m.OnIBCPacketAckedFn == nil {
		panic("not expected to be called")
	}
	return m.OnIBCPacketAckedFn(checksum, env, msg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func (m XVM) OnIBCPacketTimedOut(checksum wasmvm.Checksum, env wasmvmtypes.Env, msg IBCPacketTimedOutMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
	return m.OnIBCPacketTimedOut(checksum, env, msg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

func NewVM(dataDir string, supportedCapabilities string, memoryLimit uint32, printDebug bool, cacheSize uint32) (*XVM, error) {
	vm, err := wasmvm.NewVM(dataDir, supportedCapabilities, memoryLimit, printDebug, cacheSize)
	if err != nil {
		return nil, err
	}
	return &XVM{
		VM: vm,
	}, nil
}
