package keeper

import (
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// DefaultGasMultiplier is how many cosmwasm gas points = 1 sdk gas point
	// SDK reference costs can be found here: https://github.com/cosmos/cosmos-sdk/blob/02c6c9fafd58da88550ab4d7d494724a477c8a68/store/types/gas.go#L153-L164
	// A write at ~3000 gas and ~200us = 10 gas per us (microsecond) cpu/io
	// Rough timing have 88k gas at 90us, which is equal to 1k sdk gas... (one read)
	//
	// Please note that all gas prices returned to the wasmer engine should have this multiplied
	DefaultGasMultiplier uint64 = 100
	// DefaultInstanceCost is how much SDK gas we charge each time we load a WASM instance.
	// Creating a new instance is costly, and this helps put a recursion limit to contracts calling contracts.
	DefaultInstanceCost uint64 = 40_000
	// DefaultCompileCost is how much SDK gas we charge *per byte* for compiling WASM code.
	DefaultCompileCost uint64 = 2
	// DefaultEventAttributeDataCost is how much SDK gas we charge *per byte* for attribute data in events.
	// This is len(key) + len(value)
	DefaultEventAttributeDataCost uint64 = 1
	// DefaultPerAttributeCost is how much SDK gas we charge per attribute count.
	DefaultPerAttributeCost uint64 = 10
	// DefaultEventAttributeDataFreeTier number of bytes of attribute data we do not charge.
	DefaultEventAttributeDataFreeTier = 100
)

// GasRegister abstract source for gas costs
type GasRegister interface {
	// NewContractInstanceCosts costs to crate a new contract instance from code
	NewContractInstanceCosts(pinned bool, msgLen int, labelLength int) sdk.Gas
	// CompileCosts costs to persist and "compile" a new wasm contract
	CompileCosts(byteLength int, sourceCodeUrlLen int, builderLen int) sdk.Gas
	// InstantiateContractCosts costs when interacting with a wasm contract
	InstantiateContractCosts(pinned bool, msgLen int) sdk.Gas
	// ReplyCosts costs to to handle a message reply
	ReplyCosts(pinned bool, reply wasmvmtypes.Reply) sdk.Gas
	// EventCosts costs to persist an event
	EventCosts(evts []wasmvmtypes.EventAttribute) sdk.Gas
	// ToWasmVMGas converts from sdk gas to wasmvm gas
	ToWasmVMGas(source sdk.Gas) uint64
	// FromWasmVMGas converts from wasmvm gas to sdk gas
	FromWasmVMGas(source uint64) sdk.Gas
}

// WasmGasRegister implements GasRegister
type WasmGasRegister struct {
	instanceCost  sdk.Gas
	compileCost   sdk.Gas
	gasMultiplier sdk.Gas

	eventPerAttributeCost      sdk.Gas
	eventAttributeDataCost     sdk.Gas
	eventAttributeDataFreeTier int
}

// NewDefaultWasmGasRegister creates instance with default values
func NewDefaultWasmGasRegister() WasmGasRegister {
	return WasmGasRegister{
		instanceCost:               DefaultInstanceCost,
		compileCost:                DefaultCompileCost,
		gasMultiplier:              DefaultGasMultiplier,
		eventPerAttributeCost:      DefaultPerAttributeCost,
		eventAttributeDataCost:     DefaultEventAttributeDataCost,
		eventAttributeDataFreeTier: DefaultEventAttributeDataFreeTier,
	}
}

// NewWasmGasRegister constructor
func NewWasmGasRegister(
	instanceCost sdk.Gas,
	compileCost sdk.Gas,
	gasMultiplier sdk.Gas,
	eventAttributeCountCost sdk.Gas,
	eventAttributeLengthCost sdk.Gas,
	freeTierAttributeData int,
) WasmGasRegister {
	return WasmGasRegister{
		instanceCost:               instanceCost,
		compileCost:                compileCost,
		gasMultiplier:              gasMultiplier,
		eventPerAttributeCost:      eventAttributeCountCost,
		eventAttributeDataCost:     eventAttributeLengthCost,
		eventAttributeDataFreeTier: freeTierAttributeData,
	}
}

func (g WasmGasRegister) NewContractInstanceCosts(pinned bool, msgLen int, labelLength int) storetypes.Gas {
	return g.InstantiateContractCosts(pinned, msgLen)
}

func (g WasmGasRegister) CompileCosts(byteLength int, sourceCodeUrlLen int, builderLen int) storetypes.Gas {
	return g.compileCost * uint64(byteLength)
}

func (g WasmGasRegister) InstantiateContractCosts(pinned bool, msgLen int) sdk.Gas {
	if pinned {
		return 0
	}
	return g.instanceCost
}

func (g WasmGasRegister) ReplyCosts(pinned bool, reply wasmvmtypes.Reply) sdk.Gas {
	var eventGas sdk.Gas
	msgLen := len(reply.Result.Err)
	if reply.Result.Ok != nil {
		msgLen += len(reply.Result.Ok.Data)
		for _, e := range reply.Result.Ok.Events {
			msgLen += len(e.Type)
			eventGas += g.EventCosts(e.Attributes)
		}
	}
	return eventGas + g.InstantiateContractCosts(pinned, msgLen)
}

func (g WasmGasRegister) EventCosts(evts []wasmvmtypes.EventAttribute) sdk.Gas {
	if len(evts) == 0 {
		return 0
	}
	var storedBytes int
	for _, l := range evts {
		storedBytes += len(l.Key) + len(l.Value)
	}
	// apply free tier
	if storedBytes <= g.eventAttributeDataFreeTier {
		storedBytes = 0
	} else {
		storedBytes -= g.eventAttributeDataFreeTier
	}
	// total Length * costs + attribute count * costs
	r := sdk.NewIntFromUint64(g.eventAttributeDataCost).Mul(sdk.NewIntFromUint64(uint64(storedBytes))).
		Add(sdk.NewIntFromUint64(g.eventPerAttributeCost).Mul(sdk.NewIntFromUint64(uint64(len(evts)))))
	if !r.IsUint64() {
		panic(sdk.ErrorOutOfGas{Descriptor: "overflow"})
	}
	return r.Uint64()
}

// ToWasmVMGas convert to wasmVM contract runtime gas unit
func (g WasmGasRegister) ToWasmVMGas(source storetypes.Gas) uint64 {
	return source * g.gasMultiplier
}

// FromWasmVMGas converts to SDK gas unit
func (g WasmGasRegister) FromWasmVMGas(source uint64) sdk.Gas {
	return source / g.gasMultiplier
}
