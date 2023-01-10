package keeper

import (
	sdk "github.com/line/lbm-sdk/types"
	wasmvmtypes "github.com/line/wasmvm/types"
)

const (
	// DefaultEventAttributeDataCost is how much SDK gas is charged *per byte* for attribute data in events.
	// This is used with len(key) + len(value)
	DefaultEventAttributeDataCost uint64 = 1
	// DefaultContractMessageDataCost is how much SDK gas is charged *per byte* of the message that goes to the contract
	// This is used with len(msg). Note that the message is deserialized in the receiving contract and this is charged
	// with wasm gas already. The derserialization of results is also charged in wasmvm. I am unsure if we need to add
	// additional costs here.
	// Note: also used for error fields on reply, and data on reply. Maybe these should be pulled out to a different (non-zero) field
	DefaultContractMessageDataCost uint64 = 0
	// DefaultPerAttributeCost is how much SDK gas we charge per attribute count.
	DefaultPerAttributeCost uint64 = 10
	// DefaultPerCustomEventCost is how much SDK gas we charge per event count.
	DefaultPerCustomEventCost uint64 = 20
	// DefaultEventAttributeDataFreeTier number of bytes of total attribute data we do not charge.
	DefaultEventAttributeDataFreeTier = 100
)

// GasRegister abstract source for gas costs
type GasRegister interface {
	// NewContractInstanceCosts costs to crate a new contract instance from code
	// EventCosts costs to persist an event
	EventCosts(attrs []wasmvmtypes.EventAttribute, events wasmvmtypes.Events) sdk.Gas
}

// WasmGasRegisterConfig config type
type WasmGasRegisterConfig struct {
	// EventPerAttributeCost is how much SDK gas is charged *per byte* for attribute data in events.
	// This is used with len(key) + len(value)
	EventPerAttributeCost sdk.Gas
	// EventAttributeDataCost is how much SDK gas is charged *per byte* for attribute data in events.
	// This is used with len(key) + len(value)
	EventAttributeDataCost sdk.Gas
	// EventAttributeDataFreeTier number of bytes of total attribute data that is free of charge
	EventAttributeDataFreeTier uint64
	// ContractMessageDataCost SDK gas charged *per byte* of the message that goes to the contract
	// This is used with len(msg)
	ContractMessageDataCost sdk.Gas
	// CustomEventCost cost per custom event
	CustomEventCost uint64
}

// DefaultGasRegisterConfig default values
func DefaultGasRegisterConfig() WasmGasRegisterConfig {
	return WasmGasRegisterConfig{
		EventPerAttributeCost:      DefaultPerAttributeCost,
		CustomEventCost:            DefaultPerCustomEventCost,
		EventAttributeDataCost:     DefaultEventAttributeDataCost,
		EventAttributeDataFreeTier: DefaultEventAttributeDataFreeTier,
		ContractMessageDataCost:    DefaultContractMessageDataCost,
	}
}

// WasmGasRegister implements GasRegister interface
type WasmGasRegister struct {
	c WasmGasRegisterConfig
}

// NewDefaultWasmGasRegister creates instance with default values
func NewDefaultWasmGasRegister() WasmGasRegister {
	return NewWasmGasRegister(DefaultGasRegisterConfig())
}

// NewWasmGasRegister constructor
func NewWasmGasRegister(c WasmGasRegisterConfig) WasmGasRegister {
	return WasmGasRegister{
		c: c,
	}
}

// EventCosts costs to persist an event
func (g WasmGasRegister) EventCosts(attrs []wasmvmtypes.EventAttribute, events wasmvmtypes.Events) sdk.Gas {
	gas, remainingFreeTier := g.eventAttributeCosts(attrs, g.c.EventAttributeDataFreeTier)
	for _, e := range events {
		gas += g.c.CustomEventCost
		gas += sdk.Gas(len(e.Type)) * g.c.EventAttributeDataCost // no free tier with event type
		var attrCost sdk.Gas
		attrCost, remainingFreeTier = g.eventAttributeCosts(e.Attributes, remainingFreeTier)
		gas += attrCost
	}
	return gas
}

func (g WasmGasRegister) eventAttributeCosts(attrs []wasmvmtypes.EventAttribute, freeTier uint64) (sdk.Gas, uint64) {
	if len(attrs) == 0 {
		return 0, freeTier
	}
	var storedBytes uint64
	for _, l := range attrs {
		storedBytes += uint64(len(l.Key)) + uint64(len(l.Value))
	}
	storedBytes, freeTier = calcWithFreeTier(storedBytes, freeTier)
	// total Length * costs + attribute count * costs
	r := sdk.NewIntFromUint64(g.c.EventAttributeDataCost).Mul(sdk.NewIntFromUint64(storedBytes)).
		Add(sdk.NewIntFromUint64(g.c.EventPerAttributeCost).Mul(sdk.NewIntFromUint64(uint64(len(attrs)))))
	if !r.IsUint64() {
		panic(sdk.ErrorOutOfGas{Descriptor: "overflow"})
	}
	return r.Uint64(), freeTier
}

// apply free tier
func calcWithFreeTier(storedBytes uint64, freeTier uint64) (uint64, uint64) {
	if storedBytes <= freeTier {
		return 0, freeTier - storedBytes
	}
	storedBytes -= freeTier
	return storedBytes, 0
}
