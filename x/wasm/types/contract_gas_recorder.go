package types

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	ContractOperationInstantiate uint64 = iota
	ContractOperationExecute
	ContractOperationQuery
	ContractOperationMigrate
	ContractOperationSudo
	ContractOperationReply
	ContractOperationIbcChannelOpen
	ContractOperationIbcChannelConnect
	ContractOperationIbcChannelClose
	ContractOperationIbcPacketReceive
	ContractOperationIbcPacketAck
	ContractOperationIbcPacketTimeout
	ContractOperationUnknown
)

type GasConsumptionInfo struct {
	VMGas  uint64
	SDKGas uint64
}

type ContractGasRecord struct {
	OperationId     uint64
	ContractAddress string
	OriginalGas     GasConsumptionInfo
}

type ContractGasProcessor interface {
	IngestGasRecord(ctx sdk.Context, records []ContractGasRecord) error
	CalculateUpdatedGas(ctx sdk.Context, record ContractGasRecord) (GasConsumptionInfo, error)
	GetGasCalculationFn(ctx sdk.Context, contractAddress string) (func(operationId uint64, gasInfo GasConsumptionInfo) GasConsumptionInfo, error)
}

// NoOpContractGasProcessor is no-op gas processor
type NoOpContractGasProcessor struct {
}

var _ ContractGasProcessor = &NoOpContractGasProcessor{}

func (n *NoOpContractGasProcessor) GetGasCalculationFn(ctx sdk.Context, contractAddress string) (func(operationId uint64, gasInfo GasConsumptionInfo) GasConsumptionInfo, error) {
	return func(operationId uint64, gasConsumptionInfo GasConsumptionInfo) GasConsumptionInfo {
		return gasConsumptionInfo
	}, nil
}

func (n *NoOpContractGasProcessor) IngestGasRecord(_ sdk.Context, _ []ContractGasRecord) error {
	return nil
}

func (n *NoOpContractGasProcessor) CalculateUpdatedGas(_ sdk.Context, record ContractGasRecord) (GasConsumptionInfo, error) {
	return record.OriginalGas, nil
}
