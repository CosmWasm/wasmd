package types

import (
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.GasMeter = &ContractSDKGasMeter{}

type ContractSDKGasMeter struct {
	actualGasConsumed          sdk.Gas
	originalGas                sdk.Gas
	underlyingGasMeter         sdk.GasMeter
	contractAddress            string
	contractOperation          uint64
	contractGasCalculationFunc func(operationId uint64, info GasConsumptionInfo) GasConsumptionInfo
}

func NewContractGasMeter(gasLimit uint64, gasCalculationFunc func(uint64, GasConsumptionInfo) GasConsumptionInfo, contractAddress string, contractOperation uint64) ContractSDKGasMeter {
	return ContractSDKGasMeter{
		actualGasConsumed:          0,
		originalGas:                0,
		contractGasCalculationFunc: gasCalculationFunc,
		underlyingGasMeter:         sdk.NewGasMeter(gasLimit),
		contractAddress:            contractAddress,
		contractOperation:          contractOperation,
	}
}

func (c *ContractSDKGasMeter) GetContractAddress() string {
	return c.contractAddress
}

func (c *ContractSDKGasMeter) GetContractOperation() uint64 {
	return c.contractOperation
}

func (c *ContractSDKGasMeter) GetOriginalGas() sdk.Gas {
	return c.originalGas
}

func (c *ContractSDKGasMeter) GetActualGas() sdk.Gas {
	return c.actualGasConsumed
}

func (c *ContractSDKGasMeter) GasConsumed() storetypes.Gas {
	return c.underlyingGasMeter.GasConsumed()
}

func (c *ContractSDKGasMeter) GasConsumedToLimit() storetypes.Gas {
	return c.underlyingGasMeter.GasConsumedToLimit()
}

func (c *ContractSDKGasMeter) Limit() storetypes.Gas {
	return c.underlyingGasMeter.Limit()
}

func (c *ContractSDKGasMeter) ConsumeGas(amount storetypes.Gas, descriptor string) {
	updatedGasInfo := c.contractGasCalculationFunc(c.contractOperation, GasConsumptionInfo{SDKGas: amount})
	c.underlyingGasMeter.ConsumeGas(updatedGasInfo.SDKGas, descriptor)
	c.originalGas += amount
	c.actualGasConsumed += updatedGasInfo.SDKGas
}

func (c *ContractSDKGasMeter) RefundGas(amount storetypes.Gas, descriptor string) {
	updatedGasInfo := c.contractGasCalculationFunc(c.contractOperation, GasConsumptionInfo{SDKGas: amount})
	c.underlyingGasMeter.RefundGas(updatedGasInfo.SDKGas, descriptor)
	c.originalGas -= amount
	c.actualGasConsumed -= updatedGasInfo.SDKGas
}

func (c *ContractSDKGasMeter) IsPastLimit() bool {
	return c.underlyingGasMeter.IsPastLimit()
}

func (c *ContractSDKGasMeter) IsOutOfGas() bool {
	return c.underlyingGasMeter.IsOutOfGas()
}

func (c *ContractSDKGasMeter) String() string {
	return c.underlyingGasMeter.String()
}

func (c *ContractSDKGasMeter) CloneWithNewLimit(gasLimit uint64, description string) *ContractSDKGasMeter {
	newContractGasMeter := NewContractGasMeter(gasLimit, c.contractGasCalculationFunc, c.contractAddress, c.contractOperation)
	return &newContractGasMeter
}
