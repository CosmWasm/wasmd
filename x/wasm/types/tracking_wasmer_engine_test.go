package types

import (
	"fmt"
	"math/rand"
	"testing"

	cosmwasm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/cosmos/cosmos-sdk/store"
	stTypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	db "github.com/tendermint/tm-db"
)

type testError struct{}

func (t *testError) Error() string {
	return "Fail"
}

var errTestFail = &testError{}

type loggingVMLog struct {
	MethodName string
	Message    []byte
}

type loggingVMLogs []loggingVMLog

var _ QuerierWithCtx = &testQuerier{}
var _ BareWasmVM = &loggingVM{}
var _ ContractGasProcessor = &testGasProcessor{}

type testGasProcessor struct {
	ingestedRecords []ContractGasRecord
}

func (t *testGasProcessor) CalculateUpdatedGas(ctx sdk.Context, record ContractGasRecord) (GasConsumptionInfo, error) {
	return record.OriginalGas, nil
}

func (t *testGasProcessor) GetGasCalculationFn(ctx sdk.Context, contractAddress string) (func(operationId uint64, gasInfo GasConsumptionInfo) GasConsumptionInfo, error) {
	return func(operationId uint64, gasInfo GasConsumptionInfo) GasConsumptionInfo {
		return gasInfo
	}, nil
}

func (t *testGasProcessor) IngestGasRecord(ctx sdk.Context, records []ContractGasRecord) error {
	t.ingestedRecords = append(t.ingestedRecords, records...)
	return nil
}

type testQuerier struct {
	Ctx          sdk.Context
	Vm           WasmerEngine
	GasUsed      []uint64
	TotalGasUsed uint64
}

func (t *testQuerier) Query(request wasmvmtypes.QueryRequest, gasLimit uint64) ([]byte, error) {
	t.Ctx, _ = t.Ctx.CacheContext()
	if err := CreateNewSession(&t.Ctx, gasLimit); err != nil {
		return nil, err
	}
	response, gasUsed, err := t.Vm.Query(
		t.Ctx,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: request.Wasm.Raw.ContractAddr}},
		[]byte{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		t,
		sdk.NewInfiniteGasMeter(),
		gasLimit,
		wasmvmtypes.UFraction{},
	)
	if err := DestroySession(&t.Ctx); err != nil {
		return nil, err
	}
	t.GasUsed = append(t.GasUsed, gasUsed)
	t.TotalGasUsed += gasUsed
	return response, err
}

func (t *testQuerier) GasConsumed() uint64 {
	return t.TotalGasUsed
}

func (t *testQuerier) GetCtx() *sdk.Context {
	return &t.Ctx
}

type loggingVM struct {
	logs               loggingVMLogs
	GasUsed            []uint64
	Fail               bool
	ShouldEmulateQuery bool
	QueryGasUsage      uint64
	QueryContracts     []string
}

func (l *loggingVM) Create(code cosmwasm.WasmCode) (cosmwasm.Checksum, error) {
	if l.Fail {
		return cosmwasm.Checksum{}, errTestFail
	}
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "Create",
		Message:    nil,
	})
	return cosmwasm.Checksum{}, nil
}

func (l *loggingVM) AnalyzeCode(checksum cosmwasm.Checksum) (*wasmvmtypes.AnalysisReport, error) {
	if l.Fail {
		return nil, errTestFail
	}
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "AnalyzeCode",
		Message:    nil,
	})
	return nil, nil
}

func (l *loggingVM) Instantiate(checksum cosmwasm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, initMsg []byte, store cosmwasm.KVStore, goapi cosmwasm.GoAPI, querier cosmwasm.Querier, gasMeter cosmwasm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
	if l.Fail {
		return &wasmvmtypes.Response{}, 0, errTestFail
	}
	if l.ShouldEmulateQuery && len(l.QueryContracts) > 0 {
		querier.Query(wasmvmtypes.QueryRequest{Wasm: &wasmvmtypes.WasmQuery{Raw: &wasmvmtypes.RawQuery{ContractAddr: l.QueryContracts[0]}}}, uint64(len(l.QueryContracts))-1)
	}
	currentOperationGas := rand.Uint64() % 50000
	l.GasUsed = append(l.GasUsed, currentOperationGas)
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "Instantiate",
		Message:    initMsg,
	})
	return &wasmvmtypes.Response{}, currentOperationGas, nil
}

func (l *loggingVM) Execute(code cosmwasm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, executeMsg []byte, store cosmwasm.KVStore, goapi cosmwasm.GoAPI, querier cosmwasm.Querier, gasMeter cosmwasm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
	if l.Fail {
		return &wasmvmtypes.Response{}, 0, errTestFail
	}
	if l.ShouldEmulateQuery && len(l.QueryContracts) > 0 {
		querier.Query(wasmvmtypes.QueryRequest{Wasm: &wasmvmtypes.WasmQuery{Raw: &wasmvmtypes.RawQuery{ContractAddr: l.QueryContracts[0]}}}, uint64(len(l.QueryContracts))-1)
	}
	currentOperationGas := rand.Uint64() % 50000
	l.GasUsed = append(l.GasUsed, currentOperationGas)
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "Execute",
		Message:    executeMsg,
	})
	return &wasmvmtypes.Response{}, currentOperationGas, nil
}

func (l *loggingVM) Query(code cosmwasm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store cosmwasm.KVStore, goapi cosmwasm.GoAPI, querier cosmwasm.Querier, gasMeter cosmwasm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
	if l.Fail {
		return []byte{}, 0, errTestFail
	}
	if l.ShouldEmulateQuery && gasLimit >= 1 {
		querier.Query(wasmvmtypes.QueryRequest{Wasm: &wasmvmtypes.WasmQuery{Raw: &wasmvmtypes.RawQuery{ContractAddr: l.QueryContracts[uint64(len(l.QueryContracts))-gasLimit]}}}, gasLimit-1)
	}
	currentOperationGas := rand.Uint64() % 50000
	l.GasUsed = append(l.GasUsed, currentOperationGas)
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "Query",
		Message:    queryMsg,
	})
	return []byte{1}, currentOperationGas, nil
}

func (l *loggingVM) Migrate(checksum cosmwasm.Checksum, env wasmvmtypes.Env, migrateMsg []byte, store cosmwasm.KVStore, goapi cosmwasm.GoAPI, querier cosmwasm.Querier, gasMeter cosmwasm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
	if l.Fail {
		return &wasmvmtypes.Response{}, 0, errTestFail
	}
	if l.ShouldEmulateQuery && len(l.QueryContracts) > 0 {
		querier.Query(wasmvmtypes.QueryRequest{Wasm: &wasmvmtypes.WasmQuery{Raw: &wasmvmtypes.RawQuery{ContractAddr: l.QueryContracts[0]}}}, uint64(len(l.QueryContracts))-1)
	}
	currentOperationGas := rand.Uint64() % 50000
	l.GasUsed = append(l.GasUsed, currentOperationGas)
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "Migrate",
		Message:    migrateMsg,
	})
	return &wasmvmtypes.Response{}, currentOperationGas, nil
}

func (l *loggingVM) Sudo(checksum cosmwasm.Checksum, env wasmvmtypes.Env, sudoMsg []byte, store cosmwasm.KVStore, goapi cosmwasm.GoAPI, querier cosmwasm.Querier, gasMeter cosmwasm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
	if l.Fail {
		return &wasmvmtypes.Response{}, 0, errTestFail
	}
	if l.ShouldEmulateQuery && len(l.QueryContracts) > 0 {
		querier.Query(wasmvmtypes.QueryRequest{Wasm: &wasmvmtypes.WasmQuery{Raw: &wasmvmtypes.RawQuery{ContractAddr: l.QueryContracts[0]}}}, uint64(len(l.QueryContracts))-1)
	}
	currentOperationGas := rand.Uint64() % 50000
	l.GasUsed = append(l.GasUsed, currentOperationGas)
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "Sudo",
		Message:    sudoMsg,
	})
	return &wasmvmtypes.Response{}, currentOperationGas, nil
}

func (l *loggingVM) Reply(checksum cosmwasm.Checksum, env wasmvmtypes.Env, reply wasmvmtypes.Reply, store cosmwasm.KVStore, goapi cosmwasm.GoAPI, querier cosmwasm.Querier, gasMeter cosmwasm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
	if l.Fail {
		return &wasmvmtypes.Response{}, 0, errTestFail
	}
	if l.ShouldEmulateQuery && len(l.QueryContracts) > 0 {
		querier.Query(wasmvmtypes.QueryRequest{Wasm: &wasmvmtypes.WasmQuery{Raw: &wasmvmtypes.RawQuery{ContractAddr: l.QueryContracts[0]}}}, uint64(len(l.QueryContracts))-1)
	}
	currentOperationGas := rand.Uint64() % 50000
	l.GasUsed = append(l.GasUsed, currentOperationGas)
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "Reply",
		Message:    nil,
	})
	return &wasmvmtypes.Response{}, currentOperationGas, nil
}

func (l *loggingVM) GetCode(code cosmwasm.Checksum) (cosmwasm.WasmCode, error) {
	panic("not implemented in test")
}

func (l *loggingVM) Cleanup() {
	panic("not implemented in test")
}

func (l *loggingVM) IBCChannelOpen(checksum cosmwasm.Checksum, env wasmvmtypes.Env, channel wasmvmtypes.IBCChannelOpenMsg, store cosmwasm.KVStore, goapi cosmwasm.GoAPI, querier cosmwasm.Querier, gasMeter cosmwasm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBC3ChannelOpenResponse, uint64, error) {
	if l.Fail {
		return nil, 0, errTestFail
	}
	if l.ShouldEmulateQuery && len(l.QueryContracts) > 0 {
		querier.Query(wasmvmtypes.QueryRequest{Wasm: &wasmvmtypes.WasmQuery{Raw: &wasmvmtypes.RawQuery{ContractAddr: l.QueryContracts[0]}}}, uint64(len(l.QueryContracts))-1)
	}
	currentOperationGas := rand.Uint64() % 50000
	l.GasUsed = append(l.GasUsed, currentOperationGas)
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "IBCChannelOpen",
		Message:    nil,
	})
	return &wasmvmtypes.IBC3ChannelOpenResponse{}, currentOperationGas, nil
}

func (l *loggingVM) IBCChannelConnect(checksum cosmwasm.Checksum, env wasmvmtypes.Env, channel wasmvmtypes.IBCChannelConnectMsg, store cosmwasm.KVStore, goapi cosmwasm.GoAPI, querier cosmwasm.Querier, gasMeter cosmwasm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
	if l.Fail {
		return &wasmvmtypes.IBCBasicResponse{}, 0, errTestFail
	}
	if l.ShouldEmulateQuery && len(l.QueryContracts) > 0 {
		querier.Query(wasmvmtypes.QueryRequest{Wasm: &wasmvmtypes.WasmQuery{Raw: &wasmvmtypes.RawQuery{ContractAddr: l.QueryContracts[0]}}}, uint64(len(l.QueryContracts))-1)
	}
	currentOperationGas := rand.Uint64() % 50000
	l.GasUsed = append(l.GasUsed, currentOperationGas)
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "IBCChannelConnect",
		Message:    nil,
	})
	return &wasmvmtypes.IBCBasicResponse{}, currentOperationGas, nil
}

func (l *loggingVM) IBCChannelClose(checksum cosmwasm.Checksum, env wasmvmtypes.Env, channel wasmvmtypes.IBCChannelCloseMsg, store cosmwasm.KVStore, goapi cosmwasm.GoAPI, querier cosmwasm.Querier, gasMeter cosmwasm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
	if l.Fail {
		return &wasmvmtypes.IBCBasicResponse{}, 0, errTestFail
	}
	if l.ShouldEmulateQuery && len(l.QueryContracts) > 0 {
		querier.Query(wasmvmtypes.QueryRequest{Wasm: &wasmvmtypes.WasmQuery{Raw: &wasmvmtypes.RawQuery{ContractAddr: l.QueryContracts[0]}}}, uint64(len(l.QueryContracts))-1)
	}
	currentOperationGas := rand.Uint64() % 50000
	l.GasUsed = append(l.GasUsed, currentOperationGas)
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "IBCChannelClose",
		Message:    nil,
	})
	return &wasmvmtypes.IBCBasicResponse{}, currentOperationGas, nil
}

func (l *loggingVM) IBCPacketReceive(checksum cosmwasm.Checksum, env wasmvmtypes.Env, packet wasmvmtypes.IBCPacketReceiveMsg, store cosmwasm.KVStore, goapi cosmwasm.GoAPI, querier cosmwasm.Querier, gasMeter cosmwasm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCReceiveResult, uint64, error) {
	if l.Fail {
		return &wasmvmtypes.IBCReceiveResult{}, 0, errTestFail
	}
	if l.ShouldEmulateQuery && len(l.QueryContracts) > 0 {
		querier.Query(wasmvmtypes.QueryRequest{Wasm: &wasmvmtypes.WasmQuery{Raw: &wasmvmtypes.RawQuery{ContractAddr: l.QueryContracts[0]}}}, uint64(len(l.QueryContracts))-1)
	}
	currentOperationGas := rand.Uint64() % 50000
	l.GasUsed = append(l.GasUsed, currentOperationGas)
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "IBCPacketReceive",
		Message:    nil,
	})
	return &wasmvmtypes.IBCReceiveResult{}, currentOperationGas, nil
}

func (l *loggingVM) IBCPacketAck(checksum cosmwasm.Checksum, env wasmvmtypes.Env, ack wasmvmtypes.IBCPacketAckMsg, store cosmwasm.KVStore, goapi cosmwasm.GoAPI, querier cosmwasm.Querier, gasMeter cosmwasm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
	if l.Fail {
		return &wasmvmtypes.IBCBasicResponse{}, 0, errTestFail
	}
	if l.ShouldEmulateQuery && len(l.QueryContracts) > 0 {
		querier.Query(wasmvmtypes.QueryRequest{Wasm: &wasmvmtypes.WasmQuery{Raw: &wasmvmtypes.RawQuery{ContractAddr: l.QueryContracts[0]}}}, uint64(len(l.QueryContracts))-1)
	}
	currentOperationGas := rand.Uint64() % 50000
	l.GasUsed = append(l.GasUsed, currentOperationGas)
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "IBCPacketAck",
		Message:    nil,
	})
	return &wasmvmtypes.IBCBasicResponse{}, currentOperationGas, nil
}

func (l *loggingVM) IBCPacketTimeout(checksum cosmwasm.Checksum, env wasmvmtypes.Env, packet wasmvmtypes.IBCPacketTimeoutMsg, store cosmwasm.KVStore, goapi cosmwasm.GoAPI, querier cosmwasm.Querier, gasMeter cosmwasm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
	if l.Fail {
		return &wasmvmtypes.IBCBasicResponse{}, 0, errTestFail
	}
	if l.ShouldEmulateQuery && len(l.QueryContracts) > 0 {
		querier.Query(wasmvmtypes.QueryRequest{Wasm: &wasmvmtypes.WasmQuery{Raw: &wasmvmtypes.RawQuery{ContractAddr: l.QueryContracts[0]}}}, uint64(len(l.QueryContracts))-1)
	}
	currentOperationGas := rand.Uint64() % 50000
	l.GasUsed = append(l.GasUsed, currentOperationGas)
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "IBCPacketTimeout",
		Message:    nil,
	})
	return &wasmvmtypes.IBCBasicResponse{}, currentOperationGas, nil
}

func (l *loggingVM) Pin(checksum cosmwasm.Checksum) error {
	if l.Fail {
		return errTestFail
	}
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "Pin",
		Message:    nil,
	})
	return nil
}

func (l *loggingVM) Unpin(checksum cosmwasm.Checksum) error {
	if l.Fail {
		return errTestFail
	}
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "Unpin",
		Message:    nil,
	})
	return nil
}

func (l *loggingVM) GetMetrics() (*wasmvmtypes.Metrics, error) {
	if l.Fail {
		return nil, errTestFail
	}
	l.logs = append(l.logs, loggingVMLog{
		MethodName: "GetMetrics",
		Message:    nil,
	})
	return nil, nil
}

func (l *loggingVM) Reset() {
	l.logs = nil
	l.Fail = false
	l.GasUsed = nil
}

func TestGasTrackingVMInstantiateAndQuery(t *testing.T) {
	memDB := db.NewMemDB()
	cms := store.NewCommitMultiStore(memDB)
	emptyContext := sdk.NewContext(cms, tmproto.Header{}, false, nil)

	loggingVM := loggingVM{
		Fail:               false,
		ShouldEmulateQuery: true,
		QueryGasUsage:      50,
		QueryContracts:     []string{"1contract", "2contract", "3contract", "4contract"},
	}

	testGasRecorder := &testGasProcessor{}
	gasTrackingVm := TrackingWasmerEngine{vm: &loggingVM, gasProcessor: testGasRecorder}

	testQuerier := testQuerier{
		Ctx: emptyContext,
		Vm:  &gasTrackingVm,
	}
	testQuerier.Ctx = emptyContext

	_, gasUsed, err := gasTrackingVm.Instantiate(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.MessageInfo{},
		[]byte{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Instantiation should succeed")

	require.Equal(t, 5, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 4, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	for i, record := range testQuerier.GasUsed {
		require.Equal(t, loggingVM.GasUsed[i], record)
		require.Equal(t, testGasRecorder.ingestedRecords[i].OriginalGas.VMGas, record, "Ingested record's gas consumed must match querier's record")
		require.Equal(t, testGasRecorder.ingestedRecords[i].OperationId, ContractOperationQuery, "Operation must be query")
		require.Equal(t, testGasRecorder.ingestedRecords[i].ContractAddress, loggingVM.QueryContracts[(len(loggingVM.QueryContracts)-1)-i], "Contract address must be correct")
	}

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationInstantiate,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	loggingVM.Reset()
	testQuerier.GasUsed = nil
	testQuerier.Ctx = emptyContext
	testGasRecorder.ingestedRecords = nil

	_, gasUsed, err = gasTrackingVm.Query(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		[]byte{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		4,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Query should succeed")

	require.Nil(t, testGasRecorder.ingestedRecords, "Ingested gas records should be nil")

	require.Equal(t, 5, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 4, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	for i, record := range testQuerier.GasUsed {
		require.Equal(t, loggingVM.GasUsed[i], record)
		require.Equal(t, record, loggingVM.GasUsed[i], "Ingested record's gas consumed must match querier's record")
	}

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")

	loggingVM.Reset()
	testQuerier.GasUsed = nil
	testQuerier.Ctx = emptyContext
	testGasRecorder.ingestedRecords = nil

	loggingVM.ShouldEmulateQuery = false

	_, gasUsed, err = gasTrackingVm.Instantiate(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.MessageInfo{},
		[]byte{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Query should succeed")

	require.Equal(t, 1, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 0, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	require.Equal(t, 1, len(testGasRecorder.ingestedRecords), "There should be proper number of gas records")

	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationInstantiate,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")

	loggingVM.Reset()
	testQuerier.GasUsed = nil
	testQuerier.Ctx = emptyContext
	testGasRecorder.ingestedRecords = nil

	_, gasUsed, err = gasTrackingVm.Query(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		[]byte{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		0,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Query should succeed")

	require.Nil(t, testGasRecorder.ingestedRecords, "Ingested gas records should be nil")

	require.Equal(t, 1, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 0, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
}

func TestGasTrackingVMExecute(t *testing.T) {
	memDB := db.NewMemDB()
	cms := store.NewCommitMultiStore(memDB)
	emptyContext := sdk.NewContext(cms, tmproto.Header{}, false, nil)

	loggingVM := loggingVM{
		Fail:               false,
		ShouldEmulateQuery: true,
		QueryGasUsage:      50,
		QueryContracts:     []string{"1contract", "2contract", "3contract", "4contract"},
	}

	testGasRecorder := &testGasProcessor{}
	gasTrackingVm := TrackingWasmerEngine{vm: &loggingVM, gasProcessor: testGasRecorder}

	testQuerier := testQuerier{
		Ctx: emptyContext,
		Vm:  &gasTrackingVm,
	}
	testQuerier.Ctx = emptyContext

	_, gasUsed, err := gasTrackingVm.Execute(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.MessageInfo{},
		[]byte{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Instantiation should succeed")

	require.Equal(t, 5, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 4, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	for i, record := range testQuerier.GasUsed {
		require.Equal(t, loggingVM.GasUsed[i], record)
		require.Equal(t, testGasRecorder.ingestedRecords[i].OriginalGas.VMGas, record, "Ingested record's gas consumed must match querier's record")
		require.Equal(t, testGasRecorder.ingestedRecords[i].OperationId, ContractOperationQuery, "Operation must be query")
		require.Equal(t, testGasRecorder.ingestedRecords[i].ContractAddress, loggingVM.QueryContracts[(len(loggingVM.QueryContracts)-1)-i], "Contract address must be correct")
	}

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationExecute,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	loggingVM.Reset()
	testQuerier.GasUsed = nil
	testQuerier.Ctx = emptyContext
	testGasRecorder.ingestedRecords = nil

	loggingVM.ShouldEmulateQuery = false

	_, gasUsed, err = gasTrackingVm.Execute(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.MessageInfo{},
		[]byte{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Query should succeed")

	require.Equal(t, 1, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 0, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	require.Equal(t, 1, len(testGasRecorder.ingestedRecords), "There should be proper number of gas records")

	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationExecute,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
}

func TestGasTrackingVMMigrate(t *testing.T) {
	memDB := db.NewMemDB()
	cms := store.NewCommitMultiStore(memDB)
	emptyContext := sdk.NewContext(cms, tmproto.Header{}, false, nil)

	loggingVM := loggingVM{
		Fail:               false,
		ShouldEmulateQuery: true,
		QueryGasUsage:      50,
		QueryContracts:     []string{"1contract", "2contract", "3contract", "4contract"},
	}

	testGasRecorder := &testGasProcessor{}
	gasTrackingVm := TrackingWasmerEngine{vm: &loggingVM, gasProcessor: testGasRecorder}

	testQuerier := testQuerier{
		Ctx: emptyContext,
		Vm:  &gasTrackingVm,
	}
	testQuerier.Ctx = emptyContext

	_, gasUsed, err := gasTrackingVm.Migrate(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		[]byte{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Instantiation should succeed")

	require.Equal(t, 5, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 4, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	for i, record := range testQuerier.GasUsed {
		require.Equal(t, loggingVM.GasUsed[i], record)
		require.Equal(t, testGasRecorder.ingestedRecords[i].OriginalGas.VMGas, record, "Ingested record's gas consumed must match querier's record")
		require.Equal(t, testGasRecorder.ingestedRecords[i].OperationId, ContractOperationQuery, "Operation must be query")
		require.Equal(t, testGasRecorder.ingestedRecords[i].ContractAddress, loggingVM.QueryContracts[(len(loggingVM.QueryContracts)-1)-i], "Contract address must be correct")
	}

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationMigrate,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	loggingVM.Reset()
	testQuerier.GasUsed = nil
	testQuerier.Ctx = emptyContext
	testGasRecorder.ingestedRecords = nil

	loggingVM.ShouldEmulateQuery = false

	_, gasUsed, err = gasTrackingVm.Migrate(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		[]byte{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Query should succeed")

	require.Equal(t, 1, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 0, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	require.Equal(t, 1, len(testGasRecorder.ingestedRecords), "There should be proper number of gas records")

	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationMigrate,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
}

func TestGasTrackingVMSudo(t *testing.T) {
	memDB := db.NewMemDB()
	cms := store.NewCommitMultiStore(memDB)
	emptyContext := sdk.NewContext(cms, tmproto.Header{}, false, nil)

	loggingVM := loggingVM{
		Fail:               false,
		ShouldEmulateQuery: true,
		QueryGasUsage:      50,
		QueryContracts:     []string{"1contract", "2contract", "3contract", "4contract"},
	}

	testGasRecorder := &testGasProcessor{}
	gasTrackingVm := TrackingWasmerEngine{vm: &loggingVM, gasProcessor: testGasRecorder}

	testQuerier := testQuerier{
		Ctx: emptyContext,
		Vm:  &gasTrackingVm,
	}
	testQuerier.Ctx = emptyContext

	_, gasUsed, err := gasTrackingVm.Sudo(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		[]byte{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Instantiation should succeed")

	fmt.Println(testGasRecorder.ingestedRecords, loggingVM.GasUsed, testQuerier.GasUsed)

	require.Equal(t, 5, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 4, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	for i, record := range testQuerier.GasUsed {
		require.Equal(t, loggingVM.GasUsed[i], record)
		require.Equal(t, testGasRecorder.ingestedRecords[i].OriginalGas.VMGas, record, "Ingested record's gas consumed must match querier's record")
		require.Equal(t, testGasRecorder.ingestedRecords[i].OperationId, ContractOperationQuery, "Operation must be query")
		require.Equal(t, testGasRecorder.ingestedRecords[i].ContractAddress, loggingVM.QueryContracts[(len(loggingVM.QueryContracts)-1)-i], "Contract address must be correct")
	}

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationSudo,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	loggingVM.Reset()
	testQuerier.GasUsed = nil
	testQuerier.Ctx = emptyContext
	testGasRecorder.ingestedRecords = nil

	loggingVM.ShouldEmulateQuery = false

	_, gasUsed, err = gasTrackingVm.Sudo(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		[]byte{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Query should succeed")

	require.Equal(t, 1, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 0, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	require.Equal(t, 1, len(testGasRecorder.ingestedRecords), "There should be proper number of gas records")

	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationSudo,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
}

func TestGasTrackingVMReply(t *testing.T) {
	memDB := db.NewMemDB()
	cms := store.NewCommitMultiStore(memDB)
	emptyContext := sdk.NewContext(cms, tmproto.Header{}, false, nil)

	loggingVM := loggingVM{
		Fail:               false,
		ShouldEmulateQuery: true,
		QueryGasUsage:      50,
		QueryContracts:     []string{"1contract", "2contract", "3contract", "4contract"},
	}

	testGasRecorder := &testGasProcessor{}
	gasTrackingVm := TrackingWasmerEngine{vm: &loggingVM, gasProcessor: testGasRecorder}

	testQuerier := testQuerier{
		Ctx: emptyContext,
		Vm:  &gasTrackingVm,
	}
	testQuerier.Ctx = emptyContext

	_, gasUsed, err := gasTrackingVm.Reply(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.Reply{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Instantiation should succeed")

	require.Equal(t, 5, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 4, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	for i, record := range testQuerier.GasUsed {
		require.Equal(t, loggingVM.GasUsed[i], record)
		require.Equal(t, testGasRecorder.ingestedRecords[i].OriginalGas.VMGas, record, "Ingested record's gas consumed must match querier's record")
		require.Equal(t, testGasRecorder.ingestedRecords[i].OperationId, ContractOperationQuery, "Operation must be query")
		require.Equal(t, testGasRecorder.ingestedRecords[i].ContractAddress, loggingVM.QueryContracts[(len(loggingVM.QueryContracts)-1)-i], "Contract address must be correct")
	}

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationReply,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	loggingVM.Reset()
	testQuerier.GasUsed = nil
	testQuerier.Ctx = emptyContext
	testGasRecorder.ingestedRecords = nil

	loggingVM.ShouldEmulateQuery = false

	_, gasUsed, err = gasTrackingVm.Reply(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.Reply{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Query should succeed")

	require.Equal(t, 1, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 0, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	require.Equal(t, 1, len(testGasRecorder.ingestedRecords), "There should be proper number of gas records")

	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationReply,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
}

func TestGasTrackingVMIBCChannelOpen(t *testing.T) {
	memDB := db.NewMemDB()
	cms := store.NewCommitMultiStore(memDB)
	emptyContext := sdk.NewContext(cms, tmproto.Header{}, false, nil)

	loggingVM := loggingVM{
		Fail:               false,
		ShouldEmulateQuery: true,
		QueryGasUsage:      50,
		QueryContracts:     []string{"1contract", "2contract", "3contract", "4contract"},
	}

	testGasRecorder := &testGasProcessor{}
	gasTrackingVm := TrackingWasmerEngine{vm: &loggingVM, gasProcessor: testGasRecorder}

	testQuerier := testQuerier{
		Ctx: emptyContext,
		Vm:  &gasTrackingVm,
	}
	testQuerier.Ctx = emptyContext

	_, gasUsed, err := gasTrackingVm.IBCChannelOpen(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.IBCChannelOpenMsg{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Instantiation should succeed")

	require.Equal(t, 5, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 4, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	for i, record := range testQuerier.GasUsed {
		require.Equal(t, loggingVM.GasUsed[i], record)
		require.Equal(t, testGasRecorder.ingestedRecords[i].OriginalGas.VMGas, record, "Ingested record's gas consumed must match querier's record")
		require.Equal(t, testGasRecorder.ingestedRecords[i].OperationId, ContractOperationQuery, "Operation must be query")
		require.Equal(t, testGasRecorder.ingestedRecords[i].ContractAddress, loggingVM.QueryContracts[(len(loggingVM.QueryContracts)-1)-i], "Contract address must be correct")
	}

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationIbcChannelOpen,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	loggingVM.Reset()
	testQuerier.GasUsed = nil
	testQuerier.Ctx = emptyContext
	testGasRecorder.ingestedRecords = nil

	loggingVM.ShouldEmulateQuery = false

	_, gasUsed, err = gasTrackingVm.IBCChannelOpen(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.IBCChannelOpenMsg{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Query should succeed")

	require.Equal(t, 1, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 0, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	require.Equal(t, 1, len(testGasRecorder.ingestedRecords), "There should be proper number of gas records")

	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationIbcChannelOpen,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
}

func TestGasTrackingVMIBCChannelConnect(t *testing.T) {
	memDB := db.NewMemDB()
	cms := store.NewCommitMultiStore(memDB)
	emptyContext := sdk.NewContext(cms, tmproto.Header{}, false, nil)

	loggingVM := loggingVM{
		Fail:               false,
		ShouldEmulateQuery: true,
		QueryGasUsage:      50,
		QueryContracts:     []string{"1contract", "2contract", "3contract", "4contract"},
	}

	testGasRecorder := &testGasProcessor{}
	gasTrackingVm := TrackingWasmerEngine{vm: &loggingVM, gasProcessor: testGasRecorder}

	testQuerier := testQuerier{
		Ctx: emptyContext,
		Vm:  &gasTrackingVm,
	}
	testQuerier.Ctx = emptyContext

	_, gasUsed, err := gasTrackingVm.IBCChannelConnect(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.IBCChannelConnectMsg{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Instantiation should succeed")

	require.Equal(t, 5, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 4, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	for i, record := range testQuerier.GasUsed {
		require.Equal(t, loggingVM.GasUsed[i], record)
		require.Equal(t, testGasRecorder.ingestedRecords[i].OriginalGas.VMGas, record, "Ingested record's gas consumed must match querier's record")
		require.Equal(t, testGasRecorder.ingestedRecords[i].OperationId, ContractOperationQuery, "Operation must be query")
		require.Equal(t, testGasRecorder.ingestedRecords[i].ContractAddress, loggingVM.QueryContracts[(len(loggingVM.QueryContracts)-1)-i], "Contract address must be correct")
	}

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationIbcChannelConnect,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	loggingVM.Reset()
	testQuerier.GasUsed = nil
	testQuerier.Ctx = emptyContext
	testGasRecorder.ingestedRecords = nil

	loggingVM.ShouldEmulateQuery = false

	_, gasUsed, err = gasTrackingVm.IBCChannelConnect(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.IBCChannelConnectMsg{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Query should succeed")

	require.Equal(t, 1, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 0, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	require.Equal(t, 1, len(testGasRecorder.ingestedRecords), "There should be proper number of gas records")

	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationIbcChannelConnect,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
}

func TestGasTrackingVMIBCChannelClose(t *testing.T) {
	memDB := db.NewMemDB()
	cms := store.NewCommitMultiStore(memDB)
	emptyContext := sdk.NewContext(cms, tmproto.Header{}, false, nil)

	loggingVM := loggingVM{
		Fail:               false,
		ShouldEmulateQuery: true,
		QueryGasUsage:      50,
		QueryContracts:     []string{"1contract", "2contract", "3contract", "4contract"},
	}

	testGasRecorder := &testGasProcessor{}
	gasTrackingVm := TrackingWasmerEngine{vm: &loggingVM, gasProcessor: testGasRecorder}

	testQuerier := testQuerier{
		Ctx: emptyContext,
		Vm:  &gasTrackingVm,
	}
	testQuerier.Ctx = emptyContext

	_, gasUsed, err := gasTrackingVm.IBCChannelClose(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.IBCChannelCloseMsg{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Instantiation should succeed")

	require.Equal(t, 5, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 4, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	for i, record := range testQuerier.GasUsed {
		require.Equal(t, loggingVM.GasUsed[i], record)
		require.Equal(t, testGasRecorder.ingestedRecords[i].OriginalGas.VMGas, record, "Ingested record's gas consumed must match querier's record")
		require.Equal(t, testGasRecorder.ingestedRecords[i].OperationId, ContractOperationQuery, "Operation must be query")
		require.Equal(t, testGasRecorder.ingestedRecords[i].ContractAddress, loggingVM.QueryContracts[(len(loggingVM.QueryContracts)-1)-i], "Contract address must be correct")
	}

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationIbcChannelClose,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	loggingVM.Reset()
	testQuerier.GasUsed = nil
	testQuerier.Ctx = emptyContext
	testGasRecorder.ingestedRecords = nil

	loggingVM.ShouldEmulateQuery = false

	_, gasUsed, err = gasTrackingVm.IBCChannelClose(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.IBCChannelCloseMsg{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Query should succeed")

	require.Equal(t, 1, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 0, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	require.Equal(t, 1, len(testGasRecorder.ingestedRecords), "There should be proper number of gas records")

	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationIbcChannelClose,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
}

func TestGasTrackingVMIBCPacketReceive(t *testing.T) {
	memDB := db.NewMemDB()
	cms := store.NewCommitMultiStore(memDB)
	emptyContext := sdk.NewContext(cms, tmproto.Header{}, false, nil)

	loggingVM := loggingVM{
		Fail:               false,
		ShouldEmulateQuery: true,
		QueryGasUsage:      50,
		QueryContracts:     []string{"1contract", "2contract", "3contract", "4contract"},
	}

	testGasRecorder := &testGasProcessor{}
	gasTrackingVm := TrackingWasmerEngine{vm: &loggingVM, gasProcessor: testGasRecorder}

	testQuerier := testQuerier{
		Ctx: emptyContext,
		Vm:  &gasTrackingVm,
	}
	testQuerier.Ctx = emptyContext

	_, gasUsed, err := gasTrackingVm.IBCPacketReceive(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.IBCPacketReceiveMsg{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Instantiation should succeed")

	require.Equal(t, 5, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 4, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	for i, record := range testQuerier.GasUsed {
		require.Equal(t, loggingVM.GasUsed[i], record)
		require.Equal(t, testGasRecorder.ingestedRecords[i].OriginalGas.VMGas, record, "Ingested record's gas consumed must match querier's record")
		require.Equal(t, testGasRecorder.ingestedRecords[i].OperationId, ContractOperationQuery, "Operation must be query")
		require.Equal(t, testGasRecorder.ingestedRecords[i].ContractAddress, loggingVM.QueryContracts[(len(loggingVM.QueryContracts)-1)-i], "Contract address must be correct")
	}

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationIbcPacketReceive,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	loggingVM.Reset()
	testQuerier.GasUsed = nil
	testQuerier.Ctx = emptyContext
	testGasRecorder.ingestedRecords = nil

	loggingVM.ShouldEmulateQuery = false

	_, gasUsed, err = gasTrackingVm.IBCPacketReceive(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.IBCPacketReceiveMsg{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Query should succeed")

	require.Equal(t, 1, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 0, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	require.Equal(t, 1, len(testGasRecorder.ingestedRecords), "There should be proper number of gas records")

	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationIbcPacketReceive,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
}

func TestGasTrackingVMIBCPacketAck(t *testing.T) {
	memDB := db.NewMemDB()
	cms := store.NewCommitMultiStore(memDB)
	emptyContext := sdk.NewContext(cms, tmproto.Header{}, false, nil)

	loggingVM := loggingVM{
		Fail:               false,
		ShouldEmulateQuery: true,
		QueryGasUsage:      50,
		QueryContracts:     []string{"1contract", "2contract", "3contract", "4contract"},
	}

	testGasRecorder := &testGasProcessor{}
	gasTrackingVm := TrackingWasmerEngine{vm: &loggingVM, gasProcessor: testGasRecorder}

	testQuerier := testQuerier{
		Ctx: emptyContext,
		Vm:  &gasTrackingVm,
	}
	testQuerier.Ctx = emptyContext

	_, gasUsed, err := gasTrackingVm.IBCPacketAck(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.IBCPacketAckMsg{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Instantiation should succeed")

	require.Equal(t, 5, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 4, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	for i, record := range testQuerier.GasUsed {
		require.Equal(t, loggingVM.GasUsed[i], record)
		require.Equal(t, testGasRecorder.ingestedRecords[i].OriginalGas.VMGas, record, "Ingested record's gas consumed must match querier's record")
		require.Equal(t, testGasRecorder.ingestedRecords[i].OperationId, ContractOperationQuery, "Operation must be query")
		require.Equal(t, testGasRecorder.ingestedRecords[i].ContractAddress, loggingVM.QueryContracts[(len(loggingVM.QueryContracts)-1)-i], "Contract address must be correct")
	}

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationIbcPacketAck,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	loggingVM.Reset()
	testQuerier.GasUsed = nil
	testQuerier.Ctx = emptyContext
	testGasRecorder.ingestedRecords = nil

	loggingVM.ShouldEmulateQuery = false

	_, gasUsed, err = gasTrackingVm.IBCPacketAck(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.IBCPacketAckMsg{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Query should succeed")

	require.Equal(t, 1, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 0, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	require.Equal(t, 1, len(testGasRecorder.ingestedRecords), "There should be proper number of gas records")

	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationIbcPacketAck,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
}

func TestGasTrackingVMIBCPacketTimeout(t *testing.T) {
	memDB := db.NewMemDB()
	cms := store.NewCommitMultiStore(memDB)
	emptyContext := sdk.NewContext(cms, tmproto.Header{}, false, nil)

	loggingVM := loggingVM{
		Fail:               false,
		ShouldEmulateQuery: true,
		QueryGasUsage:      50,
		QueryContracts:     []string{"1contract", "2contract", "3contract", "4contract"},
	}

	testGasRecorder := &testGasProcessor{}
	gasTrackingVm := TrackingWasmerEngine{vm: &loggingVM, gasProcessor: testGasRecorder}

	testQuerier := testQuerier{
		Ctx: emptyContext,
		Vm:  &gasTrackingVm,
	}
	testQuerier.Ctx = emptyContext

	_, gasUsed, err := gasTrackingVm.IBCPacketTimeout(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.IBCPacketTimeoutMsg{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Instantiation should succeed")

	require.Equal(t, 5, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 4, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	for i, record := range testQuerier.GasUsed {
		require.Equal(t, loggingVM.GasUsed[i], record)
		require.Equal(t, testGasRecorder.ingestedRecords[i].OriginalGas.VMGas, record, "Ingested record's gas consumed must match querier's record")
		require.Equal(t, testGasRecorder.ingestedRecords[i].OperationId, ContractOperationQuery, "Operation must be query")
		require.Equal(t, testGasRecorder.ingestedRecords[i].ContractAddress, loggingVM.QueryContracts[(len(loggingVM.QueryContracts)-1)-i], "Contract address must be correct")
	}

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationIbcPacketTimeout,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	loggingVM.Reset()
	testQuerier.GasUsed = nil
	testQuerier.Ctx = emptyContext
	testGasRecorder.ingestedRecords = nil

	loggingVM.ShouldEmulateQuery = false

	_, gasUsed, err = gasTrackingVm.IBCPacketTimeout(
		emptyContext,
		cosmwasm.Checksum{},
		wasmvmtypes.Env{Contract: wasmvmtypes.ContractInfo{Address: "1"}},
		wasmvmtypes.IBCPacketTimeoutMsg{},
		PrefixStoreInfo{Store: store.NewCommitMultiStore(db.NewMemDB()).GetCommitKVStore(stTypes.NewKVStoreKey("test")), PrefixKey: []byte{0x1}},
		cosmwasm.GoAPI{},
		&testQuerier,
		sdk.NewInfiniteGasMeter(),
		50,
		wasmvmtypes.UFraction{},
	)

	require.NoError(t, err, "Query should succeed")

	require.Equal(t, 1, len(loggingVM.GasUsed), "There should be proper number of records of gas used")
	require.Equal(t, 0, len(testQuerier.GasUsed), "There should be proper number of records of query gas")

	require.Equal(t, 1, len(testGasRecorder.ingestedRecords), "There should be proper number of gas records")

	require.Equal(t, ContractGasRecord{
		OperationId:     ContractOperationIbcPacketTimeout,
		ContractAddress: "1",
		OriginalGas:     GasConsumptionInfo{VMGas: gasUsed},
	}, testGasRecorder.ingestedRecords[len(testGasRecorder.ingestedRecords)-1], "Last record must be correct")

	require.Equal(t, gasUsed, loggingVM.GasUsed[len(loggingVM.GasUsed)-1], "GasUsed received on response should be same as loggingVm's logs")
}
