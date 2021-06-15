package keeper

import (
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGasMultiplier is how many cosmwasm gas points = 1 sdk gas point
// SDK reference costs can be found here: https://github.com/cosmos/cosmos-sdk/blob/02c6c9fafd58da88550ab4d7d494724a477c8a68/store/types/gas.go#L153-L164
// A write at ~3000 gas and ~200us = 10 gas per us (microsecond) cpu/io
// Rough timing have 88k gas at 90us, which is equal to 1k sdk gas... (one read)
//
// Please note that all gas prices returned to the wasmer engine should have this multiplied
const DefaultGasMultiplier uint64 = 100

// DefaultInstanceCost is how much SDK gas we charge each time we load a WASM instance.
// Creating a new instance is costly, and this helps put a recursion limit to contracts calling contracts.
const DefaultInstanceCost uint64 = 40_000

// DefaultCompileCost is how much SDK gas we charge *per byte* for compiling WASM code.
const DefaultCompileCost uint64 = 2

// DefaultEventAttributeCost is how much SDK gas we charge *per byte* for attribute data in events
const DefaultEventAttributeCost uint64 = 10

type GasRegister struct {
	instanceCost  sdk.Gas
	compileCost   sdk.Gas
	gasMultiplier sdk.Gas

	eventAttributeCountCost  sdk.Gas
	eventAttributeLengthCost sdk.Gas
	freeTierAttributeLength  int
}

func DefaultGasRegister() GasRegister {
	return GasRegister{
		instanceCost:             DefaultInstanceCost,
		compileCost:              DefaultCompileCost,
		gasMultiplier:            DefaultGasMultiplier,
		eventAttributeCountCost:  0,
		eventAttributeLengthCost: DefaultEventAttributeCost,
		freeTierAttributeLength:  0,
	}
}
func NewGasRegister(instanceCost sdk.Gas, compileCost sdk.Gas, gasMultiplier sdk.Gas, eventAttributeCountCost sdk.Gas, eventAttributeLengthCost sdk.Gas, freeTierAttributeLength int) GasRegister {
	return GasRegister{
		instanceCost:             instanceCost,
		compileCost:              compileCost,
		gasMultiplier:            gasMultiplier,
		eventAttributeCountCost:  eventAttributeCountCost,
		eventAttributeLengthCost: eventAttributeLengthCost,
		freeTierAttributeLength:  freeTierAttributeLength,
	}
}

func (g GasRegister) NewContractInstanceCost(pinned bool, msgLen int, labelLength int) storetypes.Gas {
	return g.InstantiateContractCost(pinned, msgLen)
}

func (g GasRegister) CompileCost(byteLength int, sourceCodeUrlLen int, builderLen int) storetypes.Gas {
	return g.compileCost * uint64(byteLength)
}

func (g GasRegister) InstantiateContractCost(pinned bool, msgLen int) sdk.Gas {
	if pinned {
		return 0
	}
	return g.instanceCost
}

func (g GasRegister) ReplyCost(pinned bool, reply wasmvmtypes.Reply) sdk.Gas {
	var eventGas sdk.Gas
	msgLen := len(reply.Result.Err)
	if reply.Result.Ok != nil {
		msgLen += len(reply.Result.Ok.Data)
		for _, e := range reply.Result.Ok.Events {
			msgLen += len(e.Type)
			eventGas += g.EventCosts(e.Attributes)
		}
	}
	return eventGas + g.InstantiateContractCost(pinned, msgLen)
}

func (g GasRegister) EventCosts(evts []wasmvmtypes.EventAttribute) sdk.Gas {
	if len(evts) == 0 {
		return 0
	}
	var storedBytes int
	for _, l := range evts {
		storedBytes += len(l.Key) + len(l.Value)
	}
	// apply free tier
	if storedBytes <= g.freeTierAttributeLength {
		storedBytes = 0
	} else {
		storedBytes -= g.freeTierAttributeLength
	}
	// total Length * costs + attribute count * costs
	r := sdk.NewIntFromUint64(g.eventAttributeLengthCost).Mul(sdk.NewIntFromUint64(uint64(storedBytes))).
		Add(sdk.NewIntFromUint64(g.eventAttributeCountCost).Mul(sdk.NewIntFromUint64(uint64(len(evts)))))
	if !r.IsUint64() {
		panic(sdk.ErrorOutOfGas{Descriptor: "overflow"})
	}
	return r.Uint64()
}

// ToWasmVMGas convert to wasmVM contract runtime gas unit
func (g GasRegister) ToWasmVMGas(source storetypes.Gas) uint64 {
	return source * g.gasMultiplier
}

// FromWasmVMGas converts to SDK gas unit
func (g GasRegister) FromWasmVMGas(source uint64) sdk.Gas {
	return source / g.gasMultiplier
}
