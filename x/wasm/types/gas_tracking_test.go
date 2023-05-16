package types

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	db "github.com/tendermint/tm-db"
)

func newContractGasMeterByRef(gasLimt uint64, gasCalculationFn func(_ uint64, info GasConsumptionInfo) GasConsumptionInfo, contractAddress string, contractOperation uint64) *ContractSDKGasMeter {
	gasMeter := NewContractGasMeter(gasLimt, gasCalculationFn, contractAddress, contractOperation)
	return &gasMeter
}

func TestGasTracking(t *testing.T) {
	memDB := db.NewMemDB()
	cms := store.NewCommitMultiStore(memDB)
	ctx := sdk.NewContext(cms, tmproto.Header{}, false, nil)
	mainMeter := ctx.GasMeter()

	contracts := []string{"1contract", "2contract", "3contract"}

	initialGasMeter := NewContractGasMeter(30000000, func(_ uint64, info GasConsumptionInfo) GasConsumptionInfo {
		return GasConsumptionInfo{
			SDKGas: info.SDKGas * 2,
		}
	}, contracts[0], ContractOperationQuery)

	// { Initialization
	err := InitializeGasTracking(&ctx, &initialGasMeter)
	require.NoError(t, err, "There should not be any error")

	err = InitializeGasTracking(&ctx, &initialGasMeter)
	require.EqualError(t, err, "query gas tracking is already initialized", "We should not be able to initialize multiple time")

	require.Equal(t, uint64(0), mainMeter.GasConsumed(), "there should not be any consumption on main invokedGasMeter")

	// {{ Session 1 is created
	err = CreateNewSession(&ctx, 1000)
	require.NoError(t, err, "There should not be any error")

	require.Equal(t, uint64(0), mainMeter.GasConsumed(), "there should not be any consumption on main invokedGasMeter")

	ctx.GasMeter().ConsumeGas(400, "")

	// {{} Session 1 is destroyed
	err = DestroySession(&ctx)
	require.NoError(t, err, "There should not be any error")

	require.Equal(t, uint64(400*2), mainMeter.GasConsumed(), "main invokedGasMeter must have consumed 400*2 gas")

	// {{}{ Session 2 session is created
	err = CreateNewSession(&ctx, 5000)
	require.NoError(t, err, "There should not be an error")

	ctx.GasMeter().ConsumeGas(51, "")

	// {{}{ Session 2: Meter associated
	err = AssociateContractMeterWithCurrentSession(&ctx, newContractGasMeterByRef(5000, func(_ uint64, info GasConsumptionInfo) GasConsumptionInfo {
		return GasConsumptionInfo{
			SDKGas: info.SDKGas / 3,
		}
	}, contracts[1], ContractOperationIbcChannelConnect))
	require.NoError(t, err, "There should not be an error")

	// There should not be any error
	err = AssociateContractMeterWithCurrentSession(&ctx, newContractGasMeterByRef(5000, func(_ uint64, info GasConsumptionInfo) GasConsumptionInfo {
		return GasConsumptionInfo{
			SDKGas: info.SDKGas / 3,
		}
	}, contracts[1], ContractOperationIbcPacketAck))
	require.EqualError(t, err, "invokedGasMeter is associated already", "There should not be an error")

	ctx.GasMeter().ConsumeGas(100, "")

	// {{}{{ Session 3 created
	err = CreateNewSession(&ctx, 5000)
	require.NoError(t, err, "There should not be an error")

	ctx.GasMeter().ConsumeGas(33, "")

	// {{}{{ Session 3: Meter associated
	err = AssociateContractMeterWithCurrentSession(&ctx, newContractGasMeterByRef(5000, func(_ uint64, info GasConsumptionInfo) GasConsumptionInfo {
		return GasConsumptionInfo{
			SDKGas: info.SDKGas * 7,
		}
	}, contracts[2], ContractOperationIbcChannelOpen))
	require.NoError(t, err, "There should not be an error")

	ctx.GasMeter().ConsumeGas(140, "")

	ctx.GasMeter().ConsumeGas(3, "")

	// {{}{{ Session 3: Add vm record
	err = AddVMRecord(ctx, &VMRecord{
		OriginalVMGas: 1,
		ActualVMGas:   2,
	})
	require.NoError(t, err, "We should be able to add vm record")

	err = AddVMRecord(ctx, &VMRecord{
		OriginalVMGas: 1,
		ActualVMGas:   2,
	})
	require.EqualError(t, err, "gas information already present for current session", "We should be able to add vm record")

	ctx.GasMeter().ConsumeGas(4, "")

	// {{}{{} Session 3: Destroyed
	err = DestroySession(&ctx)
	require.NoError(t, err, "There should not be any error")

	require.Equal(t, uint64(147*7)+(400*2)+uint64(33/3), mainMeter.GasConsumed(), "main invokedGasMeter must have consumed 143*7 gas")

	ctx.GasMeter().ConsumeGas(210, "")

	// {{}{{} Session 2: VM Record added
	err = AddVMRecord(ctx, &VMRecord{
		OriginalVMGas: 3,
		ActualVMGas:   4,
	})
	require.NoError(t, err, "We should be able to add vm record")

	require.Equal(t, uint64(147*7)+(400*2)+uint64(33/3), mainMeter.GasConsumed(), "main invokedGasMeter must have consumed 143*7 gas")

	// {{}{{}} Session 2 Destroyed
	err = DestroySession(&ctx)
	require.NoError(t, err, "There should not be any error")

	require.Equal(t, uint64(147*7)+uint64(100/3)+uint64(210/3)+uint64(33/3)+uint64(400*2)+uint64(51*2), mainMeter.GasConsumed(), "main invokedGasMeter must have consumed 143*7 + 100/3 gas")

	// {{}{{}}{ Session 4 Created
	err = CreateNewSession(&ctx, 1000)
	require.NoError(t, err, "There should not be any error")

	ctx.GasMeter().ConsumeGas(400, "")

	ctx.GasMeter().ConsumeGas(100, "")

	// {{}{{}}{} Session 4 Destroyed
	err = DestroySession(&ctx)
	require.NoError(t, err, "There should not be any error")

	require.Equal(t, uint64(147*7)+uint64(100/3)+uint64(210/3)+uint64(33/3)+uint64(400*2)+uint64(51*2)+uint64(500*2), mainMeter.GasConsumed(), "main invokedGasMeter must have consumed 143*7 + 100/3 gas")

	// {{}{{}}{} VM Record added for initial gas invokedGasMeter
	err = AddVMRecord(ctx, &VMRecord{
		OriginalVMGas: 5,
		ActualVMGas:   6,
	})
	require.NoError(t, err, "We should be able to add vm record")

	ctx.GasMeter().ConsumeGas(200, "")

	// {{}{{}}{}} Terminated session
	sessionRecords, err := TerminateGasTracking(&ctx)
	require.NoError(t, err, "We should be able to terminate")

	require.Equal(t, uint64(147*7)+uint64(100/3)+uint64(210/3)+uint64(33/3)+uint64(400*2)+uint64(51*2)+uint64(500*2)+uint64(200*2), mainMeter.GasConsumed(), "main invokedGasMeter must have consumed 143*7 + 100/3 + 850*2 gas")

	require.Equal(t, 3, len(sessionRecords), "3 gas invokedGasMeter sessions were created")
	require.Equal(t, contracts[0], sessionRecords[0].ContractAddress)
	require.Equal(t, uint64(1151), sessionRecords[0].OriginalSDKGas)
	require.Equal(t, uint64(2302), sessionRecords[0].ActualSDKGas)
	require.Equal(t, uint64(5), sessionRecords[0].OriginalVMGas)
	require.Equal(t, uint64(6), sessionRecords[0].ActualVMGas)
	require.Equal(t, ContractOperationQuery, sessionRecords[0].ContractOperation)

	require.Equal(t, contracts[2], sessionRecords[1].ContractAddress)
	require.Equal(t, uint64(143+4), sessionRecords[1].OriginalSDKGas)
	require.Equal(t, uint64(143*7+4*7), sessionRecords[1].ActualSDKGas)
	require.Equal(t, uint64(1), sessionRecords[1].OriginalVMGas)
	require.Equal(t, uint64(2), sessionRecords[1].ActualVMGas)
	require.Equal(t, ContractOperationIbcChannelOpen, sessionRecords[1].ContractOperation)

	require.Equal(t, contracts[1], sessionRecords[2].ContractAddress)
	require.Equal(t, uint64(100+210+33), sessionRecords[2].OriginalSDKGas)
	require.Equal(t, uint64(33+70+11), sessionRecords[2].ActualSDKGas)
	require.Equal(t, uint64(3), sessionRecords[2].OriginalVMGas)
	require.Equal(t, uint64(4), sessionRecords[2].ActualVMGas)
	require.Equal(t, ContractOperationIbcChannelConnect, sessionRecords[2].ContractOperation)
}
