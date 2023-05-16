package types

import (
	"fmt"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/cosmos/cosmos-sdk/store/gaskv"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	stypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type BareWasmVM interface {
	// Create will compile the wasm code, and store the resulting pre-compile
	// as well as the original code. Both can be referenced later via CodeID
	// This must be done one time for given code, after which it can be
	// instatitated many times, and each instance called many times.
	//
	// For example, the code for all ERC-20 contracts should be the same.
	// This function stores the code for that contract only once, but it can
	// be instantiated with custom inputs in the future.
	Create(code wasmvm.WasmCode) (wasmvm.Checksum, error)

	// AnalyzeCode will statically analyze the code.
	// Currently just reports if it exposes all IBC entry points.
	AnalyzeCode(checksum wasmvm.Checksum) (*wasmvmtypes.AnalysisReport, error)

	// Instantiate will create a new contract based on the given codeID.
	// We can set the initMsg (contract "genesis") here, and it then receives
	// an account and address and can be invoked (Execute) many times.
	//
	// Storage should be set with a PrefixedKVStore that this code can safely access.
	//
	// Under the hood, we may recompile the wasm, use a cached native compile, or even use a cached instance
	// for performance.
	Instantiate(
		checksum wasmvm.Checksum,
		env wasmvmtypes.Env,
		info wasmvmtypes.MessageInfo,
		initMsg []byte,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) (*wasmvmtypes.Response, uint64, error)

	// Execute calls a given contract. Since the only difference between contracts with the same CodeID is the
	// data in their local storage, and their address in the outside world, we need no ContractID here.
	// (That is a detail for the external, sdk-facing, side).
	//
	// The caller is responsible for passing the correct `store` (which must have been initialized exactly once),
	// and setting the env with relevant info on this instance (address, balance, etc)
	Execute(
		code wasmvm.Checksum,
		env wasmvmtypes.Env,
		info wasmvmtypes.MessageInfo,
		executeMsg []byte,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) (*wasmvmtypes.Response, uint64, error)

	// Query allows a client to execute a contract-specific query. If the result is not empty, it should be
	// valid json-encoded data to return to the client.
	// The meaning of path and data can be determined by the code. Path is the suffix of the abci.QueryRequest.Path
	Query(
		code wasmvm.Checksum,
		env wasmvmtypes.Env,
		queryMsg []byte,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) ([]byte, uint64, error)

	// Migrate will migrate an existing contract to a new code binary.
	// This takes storage of the data from the original contract and the CodeID of the new contract that should
	// replace it. This allows it to run a migration step if needed, or return an error if unable to migrate
	// the given data.
	//
	// MigrateMsg has some data on how to perform the migration.
	Migrate(
		checksum wasmvm.Checksum,
		env wasmvmtypes.Env,
		migrateMsg []byte,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) (*wasmvmtypes.Response, uint64, error)

	// Sudo runs an existing contract in read/write mode (like Execute), but is never exposed to external callers
	// (either transactions or government proposals), but can only be called by other native Go modules directly.
	//
	// This allows a contract to expose custom "super user" functions or priviledged operations that can be
	// deeply integrated with native modules.
	Sudo(
		checksum wasmvm.Checksum,
		env wasmvmtypes.Env,
		sudoMsg []byte,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) (*wasmvmtypes.Response, uint64, error)

	// Reply is called on the original dispatching contract after running a submessage
	Reply(
		checksum wasmvm.Checksum,
		env wasmvmtypes.Env,
		reply wasmvmtypes.Reply,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) (*wasmvmtypes.Response, uint64, error)

	// GetCode will load the original wasm code for the given code id.
	// This will only succeed if that code id was previously returned from
	// a call to Create.
	//
	// This can be used so that the (short) code id (hash) is stored in the iavl tree
	// and the larger binary blobs (wasm and pre-compiles) are all managed by the
	// rust library
	GetCode(code wasmvm.Checksum) (wasmvm.WasmCode, error)

	// Cleanup should be called when no longer using this to free resources on the rust-side
	Cleanup()

	// IBCChannelOpen is available on IBC-enabled contracts and is a hook to call into
	// during the handshake pahse
	IBCChannelOpen(
		checksum wasmvm.Checksum,
		env wasmvmtypes.Env,
		channel wasmvmtypes.IBCChannelOpenMsg,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) (*wasmvmtypes.IBC3ChannelOpenResponse, uint64, error)

	// IBCChannelConnect is available on IBC-enabled contracts and is a hook to call into
	// during the handshake pahse
	IBCChannelConnect(
		checksum wasmvm.Checksum,
		env wasmvmtypes.Env,
		channel wasmvmtypes.IBCChannelConnectMsg,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) (*wasmvmtypes.IBCBasicResponse, uint64, error)

	// IBCChannelClose is available on IBC-enabled contracts and is a hook to call into
	// at the end of the channel lifetime
	IBCChannelClose(
		checksum wasmvm.Checksum,
		env wasmvmtypes.Env,
		channel wasmvmtypes.IBCChannelCloseMsg,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) (*wasmvmtypes.IBCBasicResponse, uint64, error)

	// IBCPacketReceive is available on IBC-enabled contracts and is called when an incoming
	// packet is received on a channel belonging to this contract
	IBCPacketReceive(
		checksum wasmvm.Checksum,
		env wasmvmtypes.Env,
		packet wasmvmtypes.IBCPacketReceiveMsg,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) (*wasmvmtypes.IBCReceiveResult, uint64, error)

	// IBCPacketAck is available on IBC-enabled contracts and is called when an
	// the response for an outgoing packet (previously sent by this contract)
	// is received
	IBCPacketAck(
		checksum wasmvm.Checksum,
		env wasmvmtypes.Env,
		ack wasmvmtypes.IBCPacketAckMsg,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) (*wasmvmtypes.IBCBasicResponse, uint64, error)

	// IBCPacketTimeout is available on IBC-enabled contracts and is called when an
	// outgoing packet (previously sent by this contract) will provably never be executed.
	// Usually handled like ack returning an error
	IBCPacketTimeout(
		checksum wasmvm.Checksum,
		env wasmvmtypes.Env,
		packet wasmvmtypes.IBCPacketTimeoutMsg,
		store wasmvm.KVStore,
		goapi wasmvm.GoAPI,
		querier wasmvm.Querier,
		gasMeter wasmvm.GasMeter,
		gasLimit uint64,
		deserCost wasmvmtypes.UFraction,
	) (*wasmvmtypes.IBCBasicResponse, uint64, error)

	// Pin pins a code to an in-memory cache, such that is
	// always loaded quickly when executed.
	// Pin is idempotent.
	Pin(checksum wasmvm.Checksum) error

	// Unpin removes the guarantee of a contract to be pinned (see Pin).
	// After calling this, the code may or may not remain in memory depending on
	// the implementor's choice.
	// Unpin is idempotent.
	Unpin(checksum wasmvm.Checksum) error

	// GetMetrics some internal metrics for monitoring purposes.
	GetMetrics() (*wasmvmtypes.Metrics, error)
}

type QueryTrackingData struct {
	TrackingRecords []ContractTrackingData
}

type ContractTrackingData struct {
	ContractAddress string
	Operation       uint64
	GasInfo         GasConsumptionInfo
}

type TrackingVMError struct {
	VmError           error
	GasProcessorError error
}

func (t *TrackingVMError) Error() string {
	vmErrorString := "<nil>"
	gasProcessorErrorString := "<nil>"

	if t.VmError != nil {
		vmErrorString = t.VmError.Error()
	}

	if t.GasProcessorError != nil {
		gasProcessorErrorString = t.GasProcessorError.Error()
	}

	return fmt.Sprintf("error in invocation of tracking vm: WASM vm error: %s and Gas recording error: %s",
		vmErrorString, gasProcessorErrorString)
}

type TrackingWasmerEngine struct {
	vm           BareWasmVM
	gasProcessor ContractGasProcessor
}

func NewTrackingWasmerEngine(vm BareWasmVM, gasProcessor ContractGasProcessor) *TrackingWasmerEngine {
	return &TrackingWasmerEngine{
		vm,
		gasProcessor,
	}
}

func (t *TrackingWasmerEngine) SetGasRecorder(recorder ContractGasProcessor) {
	t.gasProcessor = recorder
}

func (t *TrackingWasmerEngine) getActualGas(ctx sdk.Context, operationId uint64, contractAddress string, gasInfo GasConsumptionInfo) (GasConsumptionInfo, error) {
	return t.gasProcessor.CalculateUpdatedGas(ctx, ContractGasRecord{
		OperationId:     operationId,
		ContractAddress: contractAddress,
		OriginalGas:     gasInfo,
	})
}

func (t *TrackingWasmerEngine) ingestGasRecords(ctx sdk.Context, sessionRecords []*SessionRecord) error {
	gasRecords := make([]ContractGasRecord, len(sessionRecords))

	for i, sessionRecord := range sessionRecords {
		gasRecords[i] = ContractGasRecord{
			OperationId: sessionRecord.ContractOperation,
			OriginalGas: GasConsumptionInfo{
				SDKGas: sessionRecord.OriginalSDKGas,
				VMGas:  sessionRecord.OriginalVMGas,
			},
			ContractAddress: sessionRecord.ContractAddress,
		}
	}

	return t.gasProcessor.IngestGasRecord(ctx, gasRecords)
}

func (t *TrackingWasmerEngine) Create(code wasmvm.WasmCode) (wasmvm.Checksum, error) {
	return t.vm.Create(code)
}

func (t *TrackingWasmerEngine) AnalyzeCode(checksum wasmvm.Checksum) (*wasmvmtypes.AnalysisReport, error) {
	return t.vm.AnalyzeCode(checksum)
}

func (t *TrackingWasmerEngine) Query(ctx sdk.Context, code wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store PrefixStoreInfo, goapi wasmvm.GoAPI, querier QuerierWithCtx, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
	const CurrentOperation = ContractOperationQuery
	var contractAddress = env.Contract.Address

	gasCalcFn, err := t.gasProcessor.GetGasCalculationFn(ctx, contractAddress)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	contractMeter := NewContractGasMeter(gasLimit, gasCalcFn, contractAddress, CurrentOperation)

	if IsGasTrackingInitialized(*querier.GetCtx()) {
		err = AssociateContractMeterWithCurrentSession(querier.GetCtx(), &contractMeter)
		if err != nil {
			return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
		}
	} else {
		err = InitializeGasTracking(querier.GetCtx(), &contractMeter)
		if err != nil {
			return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
		}
	}

	prefixStore := prefix.NewStore(gaskv.NewStore(store.Store, &contractMeter, stypes.KVGasConfig()), store.PrefixKey)

	response, vmGasUsed, err := t.vm.Query(code, env, queryMsg, NewStoreAdapter(prefixStore), goapi, querier, gasMeter, gasLimit, deserCost)

	updatedGasInfo, trackingErr := t.getActualGas(ctx, CurrentOperation, contractAddress, GasConsumptionInfo{
		SDKGas: 0,
		VMGas:  vmGasUsed,
	})
	if trackingErr != nil {
		return response, 0, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	trackingErr = AddVMRecord(*querier.GetCtx(), &VMRecord{
		OriginalVMGas: vmGasUsed,
		ActualVMGas:   updatedGasInfo.VMGas,
	})

	if err != nil && trackingErr == nil {
		return response, updatedGasInfo.VMGas, err
	} else if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	} else {
		return response, updatedGasInfo.VMGas, nil
	}
}

func (t *TrackingWasmerEngine) Instantiate(ctx sdk.Context, checksum wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, initMsg []byte, store PrefixStoreInfo, goapi wasmvm.GoAPI, querier QuerierWithCtx, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
	const CurrentOperation = ContractOperationInstantiate
	var contractAddress = env.Contract.Address

	gasCalcFn, err := t.gasProcessor.GetGasCalculationFn(ctx, contractAddress)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	contractMeter := NewContractGasMeter(gasLimit, gasCalcFn, contractAddress, CurrentOperation)

	err = InitializeGasTracking(querier.GetCtx(), &contractMeter)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	prefixStore := prefix.NewStore(gaskv.NewStore(store.Store, &contractMeter, stypes.KVGasConfig()), store.PrefixKey)

	response, vmGasUsed, err := t.vm.Instantiate(checksum, env, info, initMsg, NewStoreAdapter(prefixStore), goapi, querier, gasMeter, gasLimit, deserCost)

	updatedGasInfo, trackingErr := t.getActualGas(ctx, CurrentOperation, contractAddress, GasConsumptionInfo{
		SDKGas: 0,
		VMGas:  vmGasUsed,
	})
	if trackingErr != nil {
		return response, 0, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	trackingErr = AddVMRecord(*querier.GetCtx(), &VMRecord{
		OriginalVMGas: vmGasUsed,
		ActualVMGas:   updatedGasInfo.VMGas,
	})
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	sessionRecords, trackingErr := TerminateGasTracking(querier.GetCtx())
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	if err != nil && trackingErr == nil {
		return response, updatedGasInfo.VMGas, err
	} else if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	ingestingErr := t.ingestGasRecords(ctx, sessionRecords)

	if ingestingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: ingestingErr}
	} else {
		return response, updatedGasInfo.VMGas, nil
	}
}

func (t *TrackingWasmerEngine) Execute(ctx sdk.Context, code wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, executeMsg []byte, store PrefixStoreInfo, goapi wasmvm.GoAPI, querier QuerierWithCtx, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
	const CurrentOperation = ContractOperationExecute
	var contractAddress = env.Contract.Address

	gasCalcFn, err := t.gasProcessor.GetGasCalculationFn(ctx, contractAddress)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	contractMeter := NewContractGasMeter(gasLimit, gasCalcFn, contractAddress, CurrentOperation)

	err = InitializeGasTracking(querier.GetCtx(), &contractMeter)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	prefixStore := prefix.NewStore(gaskv.NewStore(store.Store, &contractMeter, stypes.KVGasConfig()), store.PrefixKey)

	response, vmGasUsed, err := t.vm.Execute(code, env, info, executeMsg, NewStoreAdapter(prefixStore), goapi, querier, gasMeter, gasLimit, deserCost)

	updatedGasInfo, trackingErr := t.getActualGas(ctx, CurrentOperation, contractAddress, GasConsumptionInfo{
		SDKGas: 0,
		VMGas:  vmGasUsed,
	})
	if trackingErr != nil {
		return response, 0, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	trackingErr = AddVMRecord(*querier.GetCtx(), &VMRecord{
		OriginalVMGas: vmGasUsed,
		ActualVMGas:   updatedGasInfo.VMGas,
	})
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	sessionRecords, trackingErr := TerminateGasTracking(querier.GetCtx())
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	if err != nil && trackingErr == nil {
		return response, updatedGasInfo.VMGas, err
	} else if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	ingestingErr := t.ingestGasRecords(ctx, sessionRecords)

	if ingestingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: ingestingErr}
	} else {
		return response, updatedGasInfo.VMGas, nil
	}
}

func (t *TrackingWasmerEngine) Migrate(ctx sdk.Context, checksum wasmvm.Checksum, env wasmvmtypes.Env, migrateMsg []byte, store PrefixStoreInfo, goapi wasmvm.GoAPI, querier QuerierWithCtx, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
	const CurrentOperation = ContractOperationMigrate
	var contractAddress = env.Contract.Address

	gasCalcFn, err := t.gasProcessor.GetGasCalculationFn(ctx, contractAddress)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	contractMeter := NewContractGasMeter(gasLimit, gasCalcFn, contractAddress, CurrentOperation)

	err = InitializeGasTracking(querier.GetCtx(), &contractMeter)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	prefixStore := prefix.NewStore(gaskv.NewStore(store.Store, &contractMeter, stypes.KVGasConfig()), store.PrefixKey)

	response, vmGasUsed, err := t.vm.Migrate(checksum, env, migrateMsg, NewStoreAdapter(prefixStore), goapi, querier, gasMeter, gasLimit, deserCost)

	updatedGasInfo, trackingErr := t.getActualGas(ctx, CurrentOperation, contractAddress, GasConsumptionInfo{
		SDKGas: 0,
		VMGas:  vmGasUsed,
	})
	if trackingErr != nil {
		return response, 0, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	trackingErr = AddVMRecord(*querier.GetCtx(), &VMRecord{
		OriginalVMGas: vmGasUsed,
		ActualVMGas:   updatedGasInfo.VMGas,
	})
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	sessionRecords, trackingErr := TerminateGasTracking(querier.GetCtx())
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	if err != nil && trackingErr == nil {
		return response, updatedGasInfo.VMGas, err
	} else if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	ingestingErr := t.ingestGasRecords(ctx, sessionRecords)

	if ingestingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: ingestingErr}
	} else {
		return response, updatedGasInfo.VMGas, nil
	}
}

func (t *TrackingWasmerEngine) Sudo(ctx sdk.Context, checksum wasmvm.Checksum, env wasmvmtypes.Env, sudoMsg []byte, store PrefixStoreInfo, goapi wasmvm.GoAPI, querier QuerierWithCtx, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
	const CurrentOperation = ContractOperationSudo
	var contractAddress = env.Contract.Address

	gasCalcFn, err := t.gasProcessor.GetGasCalculationFn(ctx, contractAddress)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	contractMeter := NewContractGasMeter(gasLimit, gasCalcFn, contractAddress, CurrentOperation)

	err = InitializeGasTracking(querier.GetCtx(), &contractMeter)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	prefixStore := prefix.NewStore(gaskv.NewStore(store.Store, &contractMeter, stypes.KVGasConfig()), store.PrefixKey)

	response, vmGasUsed, err := t.vm.Sudo(checksum, env, sudoMsg, NewStoreAdapter(prefixStore), goapi, querier, gasMeter, gasLimit, deserCost)

	updatedGasInfo, trackingErr := t.getActualGas(ctx, CurrentOperation, contractAddress, GasConsumptionInfo{
		SDKGas: 0,
		VMGas:  vmGasUsed,
	})
	if trackingErr != nil {
		return response, 0, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	trackingErr = AddVMRecord(*querier.GetCtx(), &VMRecord{
		OriginalVMGas: vmGasUsed,
		ActualVMGas:   updatedGasInfo.VMGas,
	})
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	sessionRecords, trackingErr := TerminateGasTracking(querier.GetCtx())
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	if err != nil && trackingErr == nil {
		return response, updatedGasInfo.VMGas, err
	} else if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	ingestingErr := t.ingestGasRecords(ctx, sessionRecords)

	if ingestingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: ingestingErr}
	} else {
		return response, updatedGasInfo.VMGas, nil
	}
}

func (t *TrackingWasmerEngine) Reply(ctx sdk.Context, checksum wasmvm.Checksum, env wasmvmtypes.Env, reply wasmvmtypes.Reply, store PrefixStoreInfo, goapi wasmvm.GoAPI, querier QuerierWithCtx, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
	const CurrentOperation = ContractOperationReply
	var contractAddress = env.Contract.Address

	gasCalcFn, err := t.gasProcessor.GetGasCalculationFn(ctx, contractAddress)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	contractMeter := NewContractGasMeter(gasLimit, gasCalcFn, contractAddress, CurrentOperation)

	err = InitializeGasTracking(querier.GetCtx(), &contractMeter)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	prefixStore := prefix.NewStore(gaskv.NewStore(store.Store, &contractMeter, stypes.KVGasConfig()), store.PrefixKey)

	response, vmGasUsed, err := t.vm.Reply(checksum, env, reply, NewStoreAdapter(prefixStore), goapi, querier, gasMeter, gasLimit, deserCost)

	updatedGasInfo, trackingErr := t.getActualGas(ctx, CurrentOperation, contractAddress, GasConsumptionInfo{
		SDKGas: 0,
		VMGas:  vmGasUsed,
	})
	if trackingErr != nil {
		return response, 0, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	trackingErr = AddVMRecord(*querier.GetCtx(), &VMRecord{
		OriginalVMGas: vmGasUsed,
		ActualVMGas:   updatedGasInfo.VMGas,
	})
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	sessionRecords, trackingErr := TerminateGasTracking(querier.GetCtx())
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	if err != nil && trackingErr == nil {
		return response, updatedGasInfo.VMGas, err
	} else if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	ingestingErr := t.ingestGasRecords(ctx, sessionRecords)

	if ingestingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: ingestingErr}
	} else {
		return response, updatedGasInfo.VMGas, nil
	}
}

func (t *TrackingWasmerEngine) GetCode(code wasmvm.Checksum) (wasmvm.WasmCode, error) {
	return t.vm.GetCode(code)
}

func (t *TrackingWasmerEngine) Cleanup() {
	t.vm.Cleanup()
}

func (t *TrackingWasmerEngine) IBCChannelOpen(ctx sdk.Context, checksum wasmvm.Checksum, env wasmvmtypes.Env, channel wasmvmtypes.IBCChannelOpenMsg, store PrefixStoreInfo, goapi wasmvm.GoAPI, querier QuerierWithCtx, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBC3ChannelOpenResponse, uint64, error) {
	const CurrentOperation = ContractOperationIbcChannelOpen
	var contractAddress = env.Contract.Address

	gasCalcFn, err := t.gasProcessor.GetGasCalculationFn(ctx, contractAddress)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	contractMeter := NewContractGasMeter(gasLimit, gasCalcFn, contractAddress, CurrentOperation)

	err = InitializeGasTracking(querier.GetCtx(), &contractMeter)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	prefixStore := prefix.NewStore(gaskv.NewStore(store.Store, &contractMeter, stypes.KVGasConfig()), store.PrefixKey)

	response, vmGasUsed, err := t.vm.IBCChannelOpen(checksum, env, channel, NewStoreAdapter(prefixStore), goapi, querier, gasMeter, gasLimit, deserCost)

	updatedGasInfo, trackingErr := t.getActualGas(ctx, CurrentOperation, contractAddress, GasConsumptionInfo{
		SDKGas: 0,
		VMGas:  vmGasUsed,
	})
	if trackingErr != nil {
		return response, 0, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	trackingErr = AddVMRecord(*querier.GetCtx(), &VMRecord{
		OriginalVMGas: vmGasUsed,
		ActualVMGas:   updatedGasInfo.VMGas,
	})
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	sessionRecords, trackingErr := TerminateGasTracking(querier.GetCtx())
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	if err != nil && trackingErr == nil {
		return response, updatedGasInfo.VMGas, err
	} else if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	ingestingErr := t.ingestGasRecords(ctx, sessionRecords)

	if ingestingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: ingestingErr}
	} else {
		return response, updatedGasInfo.VMGas, nil
	}
}

func (t *TrackingWasmerEngine) IBCChannelConnect(ctx sdk.Context, checksum wasmvm.Checksum, env wasmvmtypes.Env, channel wasmvmtypes.IBCChannelConnectMsg, store PrefixStoreInfo, goapi wasmvm.GoAPI, querier QuerierWithCtx, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
	const CurrentOperation = ContractOperationIbcChannelConnect
	var contractAddress = env.Contract.Address

	gasCalcFn, err := t.gasProcessor.GetGasCalculationFn(ctx, contractAddress)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	contractMeter := NewContractGasMeter(gasLimit, gasCalcFn, contractAddress, CurrentOperation)

	err = InitializeGasTracking(querier.GetCtx(), &contractMeter)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	prefixStore := prefix.NewStore(gaskv.NewStore(store.Store, &contractMeter, stypes.KVGasConfig()), store.PrefixKey)

	response, vmGasUsed, err := t.vm.IBCChannelConnect(checksum, env, channel, NewStoreAdapter(prefixStore), goapi, querier, gasMeter, gasLimit, deserCost)

	updatedGasInfo, trackingErr := t.getActualGas(ctx, CurrentOperation, contractAddress, GasConsumptionInfo{
		SDKGas: 0,
		VMGas:  vmGasUsed,
	})
	if trackingErr != nil {
		return response, 0, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	trackingErr = AddVMRecord(*querier.GetCtx(), &VMRecord{
		OriginalVMGas: vmGasUsed,
		ActualVMGas:   updatedGasInfo.VMGas,
	})
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	sessionRecords, trackingErr := TerminateGasTracking(querier.GetCtx())
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	if err != nil && trackingErr == nil {
		return response, updatedGasInfo.VMGas, err
	} else if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	ingestingErr := t.ingestGasRecords(ctx, sessionRecords)

	if ingestingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: ingestingErr}
	} else {
		return response, updatedGasInfo.VMGas, nil
	}
}

func (t *TrackingWasmerEngine) IBCChannelClose(ctx sdk.Context, checksum wasmvm.Checksum, env wasmvmtypes.Env, channel wasmvmtypes.IBCChannelCloseMsg, store PrefixStoreInfo, goapi wasmvm.GoAPI, querier QuerierWithCtx, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
	const CurrentOperation = ContractOperationIbcChannelClose
	var contractAddress = env.Contract.Address

	gasCalcFn, err := t.gasProcessor.GetGasCalculationFn(ctx, contractAddress)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	contractMeter := NewContractGasMeter(gasLimit, gasCalcFn, contractAddress, CurrentOperation)

	err = InitializeGasTracking(querier.GetCtx(), &contractMeter)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	prefixStore := prefix.NewStore(gaskv.NewStore(store.Store, &contractMeter, stypes.KVGasConfig()), store.PrefixKey)

	response, vmGasUsed, err := t.vm.IBCChannelClose(checksum, env, channel, NewStoreAdapter(prefixStore), goapi, querier, gasMeter, gasLimit, deserCost)

	updatedGasInfo, trackingErr := t.getActualGas(ctx, CurrentOperation, contractAddress, GasConsumptionInfo{
		SDKGas: 0,
		VMGas:  vmGasUsed,
	})
	if trackingErr != nil {
		return response, 0, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	trackingErr = AddVMRecord(*querier.GetCtx(), &VMRecord{
		OriginalVMGas: vmGasUsed,
		ActualVMGas:   updatedGasInfo.VMGas,
	})
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	sessionRecords, trackingErr := TerminateGasTracking(querier.GetCtx())
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	if err != nil && trackingErr == nil {
		return response, updatedGasInfo.VMGas, err
	} else if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	ingestingErr := t.ingestGasRecords(ctx, sessionRecords)

	if ingestingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: ingestingErr}
	} else {
		return response, updatedGasInfo.VMGas, nil
	}
}

func (t *TrackingWasmerEngine) IBCPacketReceive(ctx sdk.Context, checksum wasmvm.Checksum, env wasmvmtypes.Env, packet wasmvmtypes.IBCPacketReceiveMsg, store PrefixStoreInfo, goapi wasmvm.GoAPI, querier QuerierWithCtx, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCReceiveResult, uint64, error) {
	const CurrentOperation = ContractOperationIbcPacketReceive
	var contractAddress = env.Contract.Address

	gasCalcFn, err := t.gasProcessor.GetGasCalculationFn(ctx, contractAddress)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	contractMeter := NewContractGasMeter(gasLimit, gasCalcFn, contractAddress, CurrentOperation)

	err = InitializeGasTracking(querier.GetCtx(), &contractMeter)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	prefixStore := prefix.NewStore(gaskv.NewStore(store.Store, &contractMeter, stypes.KVGasConfig()), store.PrefixKey)

	response, vmGasUsed, err := t.vm.IBCPacketReceive(checksum, env, packet, NewStoreAdapter(prefixStore), goapi, querier, gasMeter, gasLimit, deserCost)

	updatedGasInfo, trackingErr := t.getActualGas(ctx, CurrentOperation, contractAddress, GasConsumptionInfo{
		SDKGas: 0,
		VMGas:  vmGasUsed,
	})
	if trackingErr != nil {
		return response, 0, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	trackingErr = AddVMRecord(*querier.GetCtx(), &VMRecord{
		OriginalVMGas: vmGasUsed,
		ActualVMGas:   updatedGasInfo.VMGas,
	})
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	sessionRecords, trackingErr := TerminateGasTracking(querier.GetCtx())
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	if err != nil && trackingErr == nil {
		return response, updatedGasInfo.VMGas, err
	} else if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	ingestingErr := t.ingestGasRecords(ctx, sessionRecords)

	if ingestingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: ingestingErr}
	} else {
		return response, updatedGasInfo.VMGas, nil
	}
}

func (t *TrackingWasmerEngine) IBCPacketAck(ctx sdk.Context, checksum wasmvm.Checksum, env wasmvmtypes.Env, ack wasmvmtypes.IBCPacketAckMsg, store PrefixStoreInfo, goapi wasmvm.GoAPI, querier QuerierWithCtx, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
	const CurrentOperation = ContractOperationIbcPacketAck
	var contractAddress = env.Contract.Address

	gasCalcFn, err := t.gasProcessor.GetGasCalculationFn(ctx, contractAddress)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	contractMeter := NewContractGasMeter(gasLimit, gasCalcFn, contractAddress, CurrentOperation)

	err = InitializeGasTracking(querier.GetCtx(), &contractMeter)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	prefixStore := prefix.NewStore(gaskv.NewStore(store.Store, &contractMeter, stypes.KVGasConfig()), store.PrefixKey)

	response, vmGasUsed, err := t.vm.IBCPacketAck(checksum, env, ack, NewStoreAdapter(prefixStore), goapi, querier, gasMeter, gasLimit, deserCost)

	updatedGasInfo, trackingErr := t.getActualGas(ctx, CurrentOperation, contractAddress, GasConsumptionInfo{
		SDKGas: 0,
		VMGas:  vmGasUsed,
	})
	if trackingErr != nil {
		return response, 0, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	trackingErr = AddVMRecord(*querier.GetCtx(), &VMRecord{
		OriginalVMGas: vmGasUsed,
		ActualVMGas:   updatedGasInfo.VMGas,
	})
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	sessionRecords, trackingErr := TerminateGasTracking(querier.GetCtx())
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	if err != nil && trackingErr == nil {
		return response, updatedGasInfo.VMGas, err
	} else if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	ingestingErr := t.ingestGasRecords(ctx, sessionRecords)

	if ingestingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: ingestingErr}
	} else {
		return response, updatedGasInfo.VMGas, nil
	}
}

func (t *TrackingWasmerEngine) IBCPacketTimeout(ctx sdk.Context, checksum wasmvm.Checksum, env wasmvmtypes.Env, packet wasmvmtypes.IBCPacketTimeoutMsg, store PrefixStoreInfo, goapi wasmvm.GoAPI, querier QuerierWithCtx, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
	const CurrentOperation = ContractOperationIbcPacketTimeout
	var contractAddress = env.Contract.Address

	gasCalcFn, err := t.gasProcessor.GetGasCalculationFn(ctx, contractAddress)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	contractMeter := NewContractGasMeter(gasLimit, gasCalcFn, contractAddress, CurrentOperation)

	err = InitializeGasTracking(querier.GetCtx(), &contractMeter)
	if err != nil {
		return nil, 0, &TrackingVMError{GasProcessorError: err, VmError: nil}
	}

	prefixStore := prefix.NewStore(gaskv.NewStore(store.Store, &contractMeter, stypes.KVGasConfig()), store.PrefixKey)

	response, vmGasUsed, err := t.vm.IBCPacketTimeout(checksum, env, packet, NewStoreAdapter(prefixStore), goapi, querier, gasMeter, gasLimit, deserCost)

	updatedGasInfo, trackingErr := t.getActualGas(ctx, CurrentOperation, contractAddress, GasConsumptionInfo{
		SDKGas: 0,
		VMGas:  vmGasUsed,
	})
	if trackingErr != nil {
		return response, 0, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	trackingErr = AddVMRecord(*querier.GetCtx(), &VMRecord{
		OriginalVMGas: vmGasUsed,
		ActualVMGas:   updatedGasInfo.VMGas,
	})
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	sessionRecords, trackingErr := TerminateGasTracking(querier.GetCtx())
	if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	if err != nil && trackingErr == nil {
		return response, updatedGasInfo.VMGas, err
	} else if trackingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: trackingErr}
	}

	ingestingErr := t.ingestGasRecords(ctx, sessionRecords)

	if ingestingErr != nil {
		return response, updatedGasInfo.VMGas, &TrackingVMError{VmError: err, GasProcessorError: ingestingErr}
	} else {
		return response, updatedGasInfo.VMGas, nil
	}
}

func (t *TrackingWasmerEngine) Pin(checksum wasmvm.Checksum) error {
	return t.vm.Pin(checksum)
}

func (t *TrackingWasmerEngine) Unpin(checksum wasmvm.Checksum) error {
	return t.vm.Unpin(checksum)
}

func (t *TrackingWasmerEngine) GetMetrics() (*wasmvmtypes.Metrics, error) {
	return t.vm.GetMetrics()
}

var _ WasmerEngine = &TrackingWasmerEngine{}
