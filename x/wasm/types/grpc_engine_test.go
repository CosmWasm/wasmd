package types

import (
	"context"
	"encoding/hex"
	fmt "fmt"
	"net"
	"sync"
	"testing"
	"time"

	wasmgrpc "github.com/CosmWasm/wasmvm/v3/rpc"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// Helper functions for GoAPI
func mockHumanizeAddress(canonical []byte) (string, uint64, error) {
	return string(canonical), 0, nil
}

func mockCanonicalizeAddress(human string) ([]byte, uint64, error) {
	return []byte(human), 0, nil
}

func mockValidateAddress(human string) (uint64, error) {
	return 0, nil
}

// MockQuerier implements a basic Querier for testing
type MockQuerier struct {
	QueryFn       func(request wasmvmtypes.QueryRequest, gasLimit uint64) ([]byte, error)
	GasConsumedFn func() uint64
	gasConsumed   uint64
}

func (m *MockQuerier) Query(request wasmvmtypes.QueryRequest, gasLimit uint64) ([]byte, error) {
	if m.QueryFn != nil {
		return m.QueryFn(request, gasLimit)
	}
	return []byte(`{"result":"mock"}`), nil
}

func (m *MockQuerier) GasConsumed() uint64 {
	if m.GasConsumedFn != nil {
		return m.GasConsumedFn()
	}
	return m.gasConsumed
}

// MockGasMeter implements a basic GasMeter for testing
type MockGasMeter struct {
	gasConsumed uint64
	gasLimit    uint64
}

func (m *MockGasMeter) GasConsumed() uint64 {
	return m.gasConsumed
}

func (m *MockGasMeter) ConsumeGas(amount uint64, descriptor string) {
	m.gasConsumed += amount
	if m.gasConsumed > m.gasLimit {
		panic("out of gas")
	}
}

func (m *MockGasMeter) SetGasLimit(limit uint64) {
	m.gasLimit = limit
}

// MockKVStore implements a basic KVStore for testing
type MockKVStore struct {
	data map[string][]byte
	mu   sync.RWMutex
}

func NewMockKVStore() *MockKVStore {
	return &MockKVStore{
		data: make(map[string][]byte),
	}
}

func (m *MockKVStore) Get(key []byte) []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data[string(key)]
}

func (m *MockKVStore) Set(key, value []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[string(key)] = value
}

func (m *MockKVStore) Delete(key []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, string(key))
}

func (m *MockKVStore) Iterator(start, end []byte) wasmvmtypes.Iterator {
	return nil // Not implemented for basic tests
}

func (m *MockKVStore) ReverseIterator(start, end []byte) wasmvmtypes.Iterator {
	return nil // Not implemented for basic tests
}

// MockWasmVMServer implements the WasmVMService for testing
type MockWasmVMServer struct {
	wasmgrpc.UnimplementedWasmVMServiceServer

	// Test data
	storedModules map[string][]byte
	pinnedModules map[string]bool
	contracts     map[string]*ContractState
	mu            sync.RWMutex

	// Behavior flags
	shouldFailLoadModule       bool
	shouldFailAnalyze          bool
	shouldFailInstantiate      bool
	shouldFailExecute          bool
	shouldFailQuery            bool
	shouldFailMigrate          bool
	shouldFailSudo             bool
	shouldFailReply            bool
	shouldFailGetCode          bool
	shouldFailPin              bool
	shouldFailUnpin            bool
	shouldFailGetMetrics       bool
	shouldFailGetPinnedMetrics bool
	shouldFailIBCOperations    bool

	// Delay simulation
	operationDelay time.Duration

	// Call counters
	loadModuleCalls  int
	instantiateCalls int
	executeCalls     int
	queryCalls       int
	migrateCalls     int
	sudoCalls        int
	replyCalls       int
	ibcCalls         int
}

type ContractState struct {
	CodeID   string
	Data     []byte
	Metadata map[string]interface{}
}

func NewMockWasmVMServer() *MockWasmVMServer {
	return &MockWasmVMServer{
		storedModules: make(map[string][]byte),
		pinnedModules: make(map[string]bool),
		contracts:     make(map[string]*ContractState),
	}
}

func (m *MockWasmVMServer) SetOperationDelay(delay time.Duration) {
	m.operationDelay = delay
}

func (m *MockWasmVMServer) simulateDelay() {
	if m.operationDelay > 0 {
		time.Sleep(m.operationDelay)
	}
}

func (m *MockWasmVMServer) LoadModule(ctx context.Context, req *wasmgrpc.LoadModuleRequest) (*wasmgrpc.LoadModuleResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadModuleCalls++

	if m.shouldFailLoadModule {
		return &wasmgrpc.LoadModuleResponse{Error: "mock load module error"}, nil
	}

	if req.ModuleBytes == nil {
		return &wasmgrpc.LoadModuleResponse{Error: "empty module bytes"}, nil
	}

	// Generate a simple checksum from module bytes for testing
	checksum := fmt.Sprintf("checksum_%x", len(req.ModuleBytes))
	m.storedModules[checksum] = req.ModuleBytes

	return &wasmgrpc.LoadModuleResponse{Checksum: []byte(checksum)}, nil
}

func (m *MockWasmVMServer) GetCode(ctx context.Context, req *wasmgrpc.GetCodeRequest) (*wasmgrpc.GetCodeResponse, error) {
	m.simulateDelay()
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFailGetCode {
		return &wasmgrpc.GetCodeResponse{Error: "mock get code error"}, nil
	}

	checksumStr, err := hex.DecodeString(req.Checksum)
	if err != nil {
		return &wasmgrpc.GetCodeResponse{Error: "invalid checksum"}, nil
	}

	moduleBytes, exists := m.storedModules[string(checksumStr)]
	if !exists {
		return &wasmgrpc.GetCodeResponse{Error: "module not found"}, nil
	}

	return &wasmgrpc.GetCodeResponse{ModuleBytes: moduleBytes}, nil
}

func (m *MockWasmVMServer) PinModule(ctx context.Context, req *wasmgrpc.PinModuleRequest) (*wasmgrpc.PinModuleResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailPin {
		return &wasmgrpc.PinModuleResponse{Error: "mock pin error"}, nil
	}

	checksumStr, err := hex.DecodeString(req.Checksum)
	if err != nil {
		return &wasmgrpc.PinModuleResponse{Error: "invalid checksum"}, nil
	}

	if _, exists := m.storedModules[string(checksumStr)]; !exists {
		return &wasmgrpc.PinModuleResponse{Error: "module not found"}, nil
	}

	m.pinnedModules[string(checksumStr)] = true
	return &wasmgrpc.PinModuleResponse{}, nil
}

func (m *MockWasmVMServer) UnpinModule(ctx context.Context, req *wasmgrpc.UnpinModuleRequest) (*wasmgrpc.UnpinModuleResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailUnpin {
		return &wasmgrpc.UnpinModuleResponse{Error: "mock unpin error"}, nil
	}

	checksumStr, err := hex.DecodeString(req.Checksum)
	if err != nil {
		return &wasmgrpc.UnpinModuleResponse{Error: "invalid checksum"}, nil
	}

	delete(m.pinnedModules, string(checksumStr))
	return &wasmgrpc.UnpinModuleResponse{}, nil
}

func (m *MockWasmVMServer) AnalyzeCode(ctx context.Context, req *wasmgrpc.AnalyzeCodeRequest) (*wasmgrpc.AnalyzeCodeResponse, error) {
	m.simulateDelay()
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFailAnalyze {
		return &wasmgrpc.AnalyzeCodeResponse{Error: "mock analyze error"}, nil
	}

	checksumStr, err := hex.DecodeString(req.Checksum)
	if err != nil {
		return &wasmgrpc.AnalyzeCodeResponse{Error: "invalid checksum"}, nil
	}

	if _, exists := m.storedModules[string(checksumStr)]; !exists {
		return &wasmgrpc.AnalyzeCodeResponse{Error: "module not found"}, nil
	}

	return &wasmgrpc.AnalyzeCodeResponse{
		RequiredCapabilities: []string{"staking"},
		HasIbcEntryPoints:    true,
	}, nil
}

func (m *MockWasmVMServer) GetMetrics(ctx context.Context, req *wasmgrpc.GetMetricsRequest) (*wasmgrpc.GetMetricsResponse, error) {
	m.simulateDelay()
	if m.shouldFailGetMetrics {
		return &wasmgrpc.GetMetricsResponse{Error: "mock metrics error"}, nil
	}

	return &wasmgrpc.GetMetricsResponse{
		Metrics: &wasmgrpc.Metrics{
			HitsPinnedMemoryCache:     10,
			HitsMemoryCache:           20,
			HitsFsCache:               30,
			Misses:                    5,
			ElementsPinnedMemoryCache: 2,
			ElementsMemoryCache:       5,
			SizePinnedMemoryCache:     1024,
			SizeMemoryCache:           2048,
		},
	}, nil
}

func (m *MockWasmVMServer) GetPinnedMetrics(ctx context.Context, req *wasmgrpc.GetPinnedMetricsRequest) (*wasmgrpc.GetPinnedMetricsResponse, error) {
	m.simulateDelay()
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFailGetPinnedMetrics {
		return &wasmgrpc.GetPinnedMetricsResponse{Error: "mock pinned metrics error"}, nil
	}

	perModule := make(map[string]*wasmgrpc.PerModuleMetrics)
	for checksum := range m.pinnedModules {
		perModule[checksum] = &wasmgrpc.PerModuleMetrics{
			Hits: 5,
			Size: 1024,
		}
	}

	return &wasmgrpc.GetPinnedMetricsResponse{
		PinnedMetrics: &wasmgrpc.PinnedMetrics{
			PerModule: perModule,
		},
	}, nil
}

func (m *MockWasmVMServer) Instantiate(ctx context.Context, req *wasmgrpc.InstantiateRequest) (*wasmgrpc.InstantiateResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instantiateCalls++

	if m.shouldFailInstantiate {
		return &wasmgrpc.InstantiateResponse{Error: "mock instantiate error", GasUsed: 500}, nil
	}

	checksumStr, err := hex.DecodeString(req.Checksum)
	if err != nil {
		return &wasmgrpc.InstantiateResponse{Error: "invalid checksum"}, nil
	}

	if _, exists := m.storedModules[string(checksumStr)]; !exists {
		return &wasmgrpc.InstantiateResponse{Error: "module not found"}, nil
	}

	// Use the checksum directly as the contract ID for simplicity in tests
	m.contracts[req.Checksum] = &ContractState{
		CodeID: req.Checksum,
	}

	return &wasmgrpc.InstantiateResponse{
		Data:    []byte("init_result"),
		GasUsed: 100000,
	}, nil
}

func (m *MockWasmVMServer) Execute(ctx context.Context, req *wasmgrpc.ExecuteRequest) (*wasmgrpc.ExecuteResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeCalls++

	if m.shouldFailExecute {
		return &wasmgrpc.ExecuteResponse{Error: "mock execute error", GasUsed: 300}, nil
	}

	if _, exists := m.contracts[req.ContractId]; !exists {
		return &wasmgrpc.ExecuteResponse{Error: "contract not found"}, nil
	}

	return &wasmgrpc.ExecuteResponse{
		Data:    []byte("execute_result"),
		GasUsed: 50000,
	}, nil
}

func (m *MockWasmVMServer) Query(ctx context.Context, req *wasmgrpc.QueryRequest) (*wasmgrpc.QueryResponse, error) {
	m.simulateDelay()
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.queryCalls++

	if m.shouldFailQuery {
		return &wasmgrpc.QueryResponse{Error: "mock query error"}, nil
	}

	if _, exists := m.contracts[req.ContractId]; !exists {
		return &wasmgrpc.QueryResponse{Error: "contract not found"}, nil
	}

	return &wasmgrpc.QueryResponse{
		Result: []byte(`{"data":"query_result"}`),
	}, nil
}

func (m *MockWasmVMServer) Migrate(ctx context.Context, req *wasmgrpc.MigrateRequest) (*wasmgrpc.MigrateResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.migrateCalls++

	if m.shouldFailMigrate {
		return &wasmgrpc.MigrateResponse{Error: "mock migrate error", GasUsed: 600}, nil
	}

	if _, exists := m.contracts[req.ContractId]; !exists {
		return &wasmgrpc.MigrateResponse{Error: "contract not found"}, nil
	}

	return &wasmgrpc.MigrateResponse{
		Data:    []byte("migrate_result"),
		GasUsed: 75000,
	}, nil
}

func (m *MockWasmVMServer) Sudo(ctx context.Context, req *wasmgrpc.SudoRequest) (*wasmgrpc.SudoResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sudoCalls++

	if m.shouldFailSudo {
		return &wasmgrpc.SudoResponse{Error: "mock sudo error", GasUsed: 200}, nil
	}

	if _, exists := m.contracts[req.ContractId]; !exists {
		return &wasmgrpc.SudoResponse{Error: "contract not found"}, nil
	}

	return &wasmgrpc.SudoResponse{
		Data:    []byte("sudo_result"),
		GasUsed: 60000,
	}, nil
}

func (m *MockWasmVMServer) Reply(ctx context.Context, req *wasmgrpc.ReplyRequest) (*wasmgrpc.ReplyResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.replyCalls++

	if m.shouldFailReply {
		return &wasmgrpc.ReplyResponse{Error: "mock reply error", GasUsed: 150}, nil
	}

	if _, exists := m.contracts[req.ContractId]; !exists {
		return &wasmgrpc.ReplyResponse{Error: "contract not found"}, nil
	}

	return &wasmgrpc.ReplyResponse{
		Data:    []byte("reply_result"),
		GasUsed: 40000,
	}, nil
}

// IBC methods - return basic responses
func (m *MockWasmVMServer) IbcChannelOpen(ctx context.Context, req *wasmgrpc.IbcMsgRequest) (*wasmgrpc.IbcMsgResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ibcCalls++

	if m.shouldFailIBCOperations {
		return &wasmgrpc.IbcMsgResponse{Error: "mock ibc channel open error", GasUsed: 50}, nil
	}
	return &wasmgrpc.IbcMsgResponse{Data: []byte(`{"ibc":"channel_open"}`), GasUsed: 100}, nil
}

func (m *MockWasmVMServer) IbcChannelConnect(ctx context.Context, req *wasmgrpc.IbcMsgRequest) (*wasmgrpc.IbcMsgResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ibcCalls++

	if m.shouldFailIBCOperations {
		return &wasmgrpc.IbcMsgResponse{Error: "mock ibc channel connect error", GasUsed: 50}, nil
	}
	return &wasmgrpc.IbcMsgResponse{Data: []byte(`{"ibc":"channel_connect"}`), GasUsed: 100}, nil
}

func (m *MockWasmVMServer) IbcChannelClose(ctx context.Context, req *wasmgrpc.IbcMsgRequest) (*wasmgrpc.IbcMsgResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ibcCalls++

	if m.shouldFailIBCOperations {
		return &wasmgrpc.IbcMsgResponse{Error: "mock ibc channel close error", GasUsed: 50}, nil
	}
	return &wasmgrpc.IbcMsgResponse{Data: []byte(`{"ibc":"channel_close"}`), GasUsed: 100}, nil
}

func (m *MockWasmVMServer) IbcPacketReceive(ctx context.Context, req *wasmgrpc.IbcMsgRequest) (*wasmgrpc.IbcMsgResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ibcCalls++

	if m.shouldFailIBCOperations {
		return &wasmgrpc.IbcMsgResponse{Error: "mock ibc packet receive error", GasUsed: 50}, nil
	}
	return &wasmgrpc.IbcMsgResponse{Data: []byte(`{"ibc":"packet_receive"}`), GasUsed: 100}, nil
}

func (m *MockWasmVMServer) IbcPacketAck(ctx context.Context, req *wasmgrpc.IbcMsgRequest) (*wasmgrpc.IbcMsgResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ibcCalls++

	if m.shouldFailIBCOperations {
		return &wasmgrpc.IbcMsgResponse{Error: "mock ibc packet ack error", GasUsed: 50}, nil
	}
	return &wasmgrpc.IbcMsgResponse{Data: []byte(`{"ibc":"packet_ack"}`), GasUsed: 100}, nil
}

func (m *MockWasmVMServer) IbcPacketTimeout(ctx context.Context, req *wasmgrpc.IbcMsgRequest) (*wasmgrpc.IbcMsgResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ibcCalls++

	if m.shouldFailIBCOperations {
		return &wasmgrpc.IbcMsgResponse{Error: "mock ibc packet timeout error", GasUsed: 50}, nil
	}
	return &wasmgrpc.IbcMsgResponse{Data: []byte(`{"ibc":"packet_timeout"}`), GasUsed: 100}, nil
}

func (m *MockWasmVMServer) IbcSourceCallback(ctx context.Context, req *wasmgrpc.IbcMsgRequest) (*wasmgrpc.IbcMsgResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ibcCalls++

	if m.shouldFailIBCOperations {
		return &wasmgrpc.IbcMsgResponse{Error: "mock ibc source callback error", GasUsed: 50}, nil
	}
	return &wasmgrpc.IbcMsgResponse{Data: []byte(`{"ibc":"source_callback"}`), GasUsed: 100}, nil
}

func (m *MockWasmVMServer) IbcDestinationCallback(ctx context.Context, req *wasmgrpc.IbcMsgRequest) (*wasmgrpc.IbcMsgResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ibcCalls++

	if m.shouldFailIBCOperations {
		return &wasmgrpc.IbcMsgResponse{Error: "mock ibc destination callback error", GasUsed: 50}, nil
	}
	return &wasmgrpc.IbcMsgResponse{Data: []byte(`{"ibc":"destination_callback"}`), GasUsed: 100}, nil
}

func (m *MockWasmVMServer) Ibc2PacketReceive(ctx context.Context, req *wasmgrpc.IbcMsgRequest) (*wasmgrpc.IbcMsgResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ibcCalls++

	if m.shouldFailIBCOperations {
		return &wasmgrpc.IbcMsgResponse{Error: "mock ibc2 packet receive error", GasUsed: 50}, nil
	}
	return &wasmgrpc.IbcMsgResponse{Data: []byte(`{"ibc":"ibc2_packet_receive"}`), GasUsed: 100}, nil
}

func (m *MockWasmVMServer) Ibc2PacketAck(ctx context.Context, req *wasmgrpc.IbcMsgRequest) (*wasmgrpc.IbcMsgResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ibcCalls++

	if m.shouldFailIBCOperations {
		return &wasmgrpc.IbcMsgResponse{Error: "mock ibc2 packet ack error", GasUsed: 50}, nil
	}
	return &wasmgrpc.IbcMsgResponse{Data: []byte(`{"ibc":"ibc2_packet_ack"}`), GasUsed: 100}, nil
}

func (m *MockWasmVMServer) Ibc2PacketTimeout(ctx context.Context, req *wasmgrpc.IbcMsgRequest) (*wasmgrpc.IbcMsgResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ibcCalls++

	if m.shouldFailIBCOperations {
		return &wasmgrpc.IbcMsgResponse{Error: "mock ibc2 packet timeout error", GasUsed: 50}, nil
	}
	return &wasmgrpc.IbcMsgResponse{Data: []byte(`{"ibc":"ibc2_packet_timeout"}`), GasUsed: 100}, nil
}

func (m *MockWasmVMServer) Ibc2PacketSend(ctx context.Context, req *wasmgrpc.IbcMsgRequest) (*wasmgrpc.IbcMsgResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ibcCalls++

	if m.shouldFailIBCOperations {
		return &wasmgrpc.IbcMsgResponse{Error: "mock ibc2 packet send error", GasUsed: 50}, nil
	}
	return &wasmgrpc.IbcMsgResponse{Data: []byte(`{"ibc":"ibc2_packet_send"}`), GasUsed: 100}, nil
}

func (m *MockWasmVMServer) RemoveModule(ctx context.Context, req *wasmgrpc.RemoveModuleRequest) (*wasmgrpc.RemoveModuleResponse, error) {
	m.simulateDelay()
	m.mu.Lock()
	defer m.mu.Unlock()

	checksumStr, err := hex.DecodeString(req.Checksum)
	if err != nil {
		return &wasmgrpc.RemoveModuleResponse{Error: "invalid checksum"}, nil
	}

	delete(m.storedModules, string(checksumStr))
	delete(m.pinnedModules, string(checksumStr))
	return &wasmgrpc.RemoveModuleResponse{}, nil
}

// Test helper functions
func startMockServer(t testing.TB, port string) (*grpc.Server, *MockWasmVMServer) {
	lis, err := net.Listen("tcp", ":"+port)
	require.NoError(t, err)

	server := grpc.NewServer()
	mockService := NewMockWasmVMServer()
	wasmgrpc.RegisterWasmVMServiceServer(server, mockService)

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("Server failed to serve: %v", err)
		}
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	return server, mockService
}

func createTestEnvironment() (wasmvmtypes.Env, wasmvmtypes.MessageInfo, *MockKVStore, wasmvmtypes.GoAPI, *MockQuerier, *MockGasMeter) {
	env := wasmvmtypes.Env{
		Block: wasmvmtypes.BlockInfo{
			Height:  100,
			Time:    1234567890,
			ChainID: "test-chain-1",
		},
		Transaction: &wasmvmtypes.TransactionInfo{
			Index: 0,
		},
		Contract: wasmvmtypes.ContractInfo{
			Address: "cosmos1test",
		},
	}

	info := wasmvmtypes.MessageInfo{
		Sender: "cosmos1sender",
		Funds:  []wasmvmtypes.Coin{},
	}

	store := NewMockKVStore()
	goAPI := wasmvmtypes.GoAPI{
		HumanizeAddress:     mockHumanizeAddress,
		CanonicalizeAddress: mockCanonicalizeAddress,
		ValidateAddress:     mockValidateAddress,
	}
	querier := &MockQuerier{}
	gasMeter := &MockGasMeter{gasLimit: 1000000}

	return env, info, store, goAPI, querier, gasMeter
}

// Connection and initialization tests
func TestNewGRPCEngine_DefaultPort(t *testing.T) {
	// Test connection to default port (should fail if no server running)
	engine, err := NewGRPCEngine("")
	if err != nil {
		// This is expected if no server is running on 50051
		t.Logf("Expected error when no server running: %v", err)
		return
	}

	// If we get here, there might be a server running
	engine.Cleanup()
}

func TestNewGRPCEngine_CustomPort(t *testing.T) {
	// Start mock server on a different port
	server, _ := startMockServer(t, "50052")
	defer server.Stop()

	// Test connection to custom port
	engine, err := NewGRPCEngine("localhost:50052")
	require.NoError(t, err)
	require.NotNil(t, engine)

	engine.Cleanup()
}

func TestNewGRPCEngine_InvalidAddress(t *testing.T) {
	// Test connection to invalid address
	_, err := NewGRPCEngine("invalid-address:99999")
	assert.Error(t, err)
	// The error could be "connection refused" or "context deadline exceeded" depending on the system
}

func TestNewGRPCEngine_ConnectionTimeout(t *testing.T) {
	// Test connection timeout to non-existent port
	start := time.Now()
	_, err := NewGRPCEngine("localhost:99999")
	elapsed := time.Since(start)

	assert.Error(t, err)
	// Should timeout around 5 seconds (with some tolerance)
	assert.True(t, elapsed >= 4*time.Second, "Should timeout after approximately 5 seconds, got %v", elapsed)
}

// Code storage and retrieval tests
func TestGRPCEngine_StoreCode_Success(t *testing.T) {
	server, mockService := startMockServer(t, "50053")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50053")
	require.NoError(t, err)
	defer engine.Cleanup()

	// Test successful store code
	wasmCode := []byte("fake wasm code")
	checksum, gasUsed, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)
	assert.NotEmpty(t, checksum)
	assert.Equal(t, uint64(0), gasUsed) // Mock returns 0

	// Verify code was stored
	assert.Contains(t, mockService.storedModules, string(checksum))
	assert.Equal(t, 1, mockService.loadModuleCalls)
}

func TestGRPCEngine_StoreCode_Failure(t *testing.T) {
	server, mockService := startMockServer(t, "50054")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50054")
	require.NoError(t, err)
	defer engine.Cleanup()

	// Test store code failure
	mockService.shouldFailLoadModule = true
	wasmCode := []byte("fake wasm code")
	_, _, err = engine.StoreCode(wasmCode, 1000000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock load module error")
}

func TestGRPCEngine_StoreCode_NilCode(t *testing.T) {
	server, _ := startMockServer(t, "50055")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50055")
	require.NoError(t, err)
	defer engine.Cleanup()

	// Test store nil code
	_, _, err = engine.StoreCode(nil, 1000000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty module bytes")
}

func TestGRPCEngine_StoreCodeUnchecked(t *testing.T) {
	server, _ := startMockServer(t, "50056")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50056")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, err := engine.StoreCodeUnchecked(wasmCode)
	require.NoError(t, err)
	assert.NotEmpty(t, checksum)
}

func TestGRPCEngine_SimulateStoreCode(t *testing.T) {
	server, _ := startMockServer(t, "50057")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50057")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, gasUsed, err := engine.SimulateStoreCode(wasmCode, 1000000)
	require.NoError(t, err)
	assert.NotEmpty(t, checksum)
	assert.Equal(t, uint64(0), gasUsed)
}

func TestGRPCEngine_GetCode_Success(t *testing.T) {
	server, _ := startMockServer(t, "50058")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50058")
	require.NoError(t, err)
	defer engine.Cleanup()

	// First store some code
	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	// Test get code
	retrievedCode, err := engine.GetCode(checksum)
	require.NoError(t, err)
	assert.Equal(t, wasmCode, []byte(retrievedCode))
}

func TestGRPCEngine_GetCode_NotFound(t *testing.T) {
	server, _ := startMockServer(t, "50059")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50059")
	require.NoError(t, err)
	defer engine.Cleanup()

	// Test get non-existent code
	fakeChecksum := wasmvmtypes.Checksum("nonexistent")
	_, err = engine.GetCode(fakeChecksum)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module not found")
}

func TestGRPCEngine_GetCode_ServerError(t *testing.T) {
	server, mockService := startMockServer(t, "50060")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50060")
	require.NoError(t, err)
	defer engine.Cleanup()

	mockService.shouldFailGetCode = true
	fakeChecksum := wasmvmtypes.Checksum("test")
	_, err = engine.GetCode(fakeChecksum)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock get code error")
}

// Code analysis tests
func TestGRPCEngine_AnalyzeCode_Success(t *testing.T) {
	server, _ := startMockServer(t, "50061")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50061")
	require.NoError(t, err)
	defer engine.Cleanup()

	// First store some code
	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	// Test analyze code
	report, err := engine.AnalyzeCode(checksum)
	require.NoError(t, err)
	assert.Equal(t, "staking", report.RequiredCapabilities)
	assert.True(t, report.HasIBCEntryPoints)
}

func TestGRPCEngine_AnalyzeCode_NotFound(t *testing.T) {
	server, _ := startMockServer(t, "50062")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50062")
	require.NoError(t, err)
	defer engine.Cleanup()

	// Test analyze non-existent code
	fakeChecksum := wasmvmtypes.Checksum("nonexistent")
	_, err = engine.AnalyzeCode(fakeChecksum)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module not found")
}

func TestGRPCEngine_AnalyzeCode_Failure(t *testing.T) {
	server, mockService := startMockServer(t, "50063")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50063")
	require.NoError(t, err)
	defer engine.Cleanup()

	// First store some code
	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	// Test analyze code failure
	mockService.shouldFailAnalyze = true
	_, err = engine.AnalyzeCode(checksum)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock analyze error")
}

// Pin/Unpin tests
func TestGRPCEngine_PinUnpin_Success(t *testing.T) {
	server, mockService := startMockServer(t, "50064")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50064")
	require.NoError(t, err)
	defer engine.Cleanup()

	// First store some code
	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	// Test pin
	err = engine.Pin(checksum)
	require.NoError(t, err)
	assert.True(t, mockService.pinnedModules[string(checksum)])

	// Test unpin
	err = engine.Unpin(checksum)
	require.NoError(t, err)
	assert.False(t, mockService.pinnedModules[string(checksum)])
}

func TestGRPCEngine_Pin_NotFound(t *testing.T) {
	server, _ := startMockServer(t, "50065")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50065")
	require.NoError(t, err)
	defer engine.Cleanup()

	// Test pin non-existent code
	fakeChecksum := wasmvmtypes.Checksum("nonexistent")
	err = engine.Pin(fakeChecksum)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module not found")
}

func TestGRPCEngine_Pin_Failure(t *testing.T) {
	server, mockService := startMockServer(t, "50066")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50066")
	require.NoError(t, err)
	defer engine.Cleanup()

	mockService.shouldFailPin = true
	fakeChecksum := wasmvmtypes.Checksum("test")
	err = engine.Pin(fakeChecksum)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock pin error")
}

func TestGRPCEngine_Unpin_Failure(t *testing.T) {
	server, mockService := startMockServer(t, "50067")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50067")
	require.NoError(t, err)
	defer engine.Cleanup()

	mockService.shouldFailUnpin = true
	fakeChecksum := wasmvmtypes.Checksum("test")
	err = engine.Unpin(fakeChecksum)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock unpin error")
}

// Metrics tests
func TestGRPCEngine_GetMetrics_Success(t *testing.T) {
	server, _ := startMockServer(t, "50068")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50068")
	require.NoError(t, err)
	defer engine.Cleanup()

	// Test get metrics
	metrics, err := engine.GetMetrics()
	require.NoError(t, err)
	assert.Equal(t, uint32(10), metrics.HitsPinnedMemoryCache)
	assert.Equal(t, uint32(20), metrics.HitsMemoryCache)
	assert.Equal(t, uint32(30), metrics.HitsFsCache)
	assert.Equal(t, uint32(5), metrics.Misses)
	assert.Equal(t, uint64(2), metrics.ElementsPinnedMemoryCache)
	assert.Equal(t, uint64(5), metrics.ElementsMemoryCache)
	assert.Equal(t, uint64(1024), metrics.SizePinnedMemoryCache)
	assert.Equal(t, uint64(2048), metrics.SizeMemoryCache)
}

func TestGRPCEngine_GetMetrics_Failure(t *testing.T) {
	server, mockService := startMockServer(t, "50069")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50069")
	require.NoError(t, err)
	defer engine.Cleanup()

	mockService.shouldFailGetMetrics = true
	_, err = engine.GetMetrics()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock metrics error")
}

func TestGRPCEngine_GetPinnedMetrics_Success(t *testing.T) {
	server, _ := startMockServer(t, "50070")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50070")
	require.NoError(t, err)
	defer engine.Cleanup()

	// First store and pin some code
	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)
	err = engine.Pin(checksum)
	require.NoError(t, err)

	// Test get pinned metrics
	pinnedMetrics, err := engine.GetPinnedMetrics()
	require.NoError(t, err)
	assert.NotNil(t, pinnedMetrics)
	assert.Len(t, pinnedMetrics.PerModule, 1)

	// Verify the pinned module metrics
	found := false
	for _, entry := range pinnedMetrics.PerModule {
		if string(entry.Checksum) == string(checksum) {
			assert.Equal(t, uint32(5), entry.Metrics.Hits)
			assert.Equal(t, uint64(1024), entry.Metrics.Size)
			found = true
			break
		}
	}
	assert.True(t, found, "Should find metrics for pinned module")
}

func TestGRPCEngine_GetPinnedMetrics_Failure(t *testing.T) {
	server, mockService := startMockServer(t, "50071")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50071")
	require.NoError(t, err)
	defer engine.Cleanup()

	mockService.shouldFailGetPinnedMetrics = true
	_, err = engine.GetPinnedMetrics()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock pinned metrics error")
}

// Contract operation tests
func TestGRPCEngine_Instantiate_Success(t *testing.T) {
	server, mockService := startMockServer(t, "50072")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50072")
	require.NoError(t, err)
	defer engine.Cleanup()

	// First store some code
	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	// Create test environment
	env, info, store, goAPI, querier, gasMeter := createTestEnvironment()

	// Test instantiate
	result, gasUsed, err := engine.Instantiate(checksum, env, info, []byte(`{"init":"data"}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Ok)
	assert.Equal(t, []byte("init_result"), result.Ok.Data)
	assert.Equal(t, uint64(100000), gasUsed)
	assert.Equal(t, 1, mockService.instantiateCalls)
}

func TestGRPCEngine_Instantiate_Failure(t *testing.T) {
	server, mockService := startMockServer(t, "50073")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50073")
	require.NoError(t, err)
	defer engine.Cleanup()

	// First store some code
	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, info, store, goAPI, querier, gasMeter := createTestEnvironment()

	mockService.shouldFailInstantiate = true
	_, gasUsed, err := engine.Instantiate(checksum, env, info, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock instantiate error")
	assert.Equal(t, uint64(500), gasUsed) // Should return gas used even on error
}

func TestGRPCEngine_Execute_Success(t *testing.T) {
	server, mockService := startMockServer(t, "50074")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50074")
	require.NoError(t, err)
	defer engine.Cleanup()

	// First store some code
	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, info, store, goAPI, querier, gasMeter := createTestEnvironment()

	// Instantiate the contract first
	_, _, err = engine.Instantiate(checksum, env, info, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)

	// Test execute
	msg := []byte(`{"action":"test"}`)
	result, gasUsed, err := engine.Execute(checksum, env, info, msg, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Ok)
	assert.Equal(t, []byte("execute_result"), result.Ok.Data)
	assert.Equal(t, uint64(50000), gasUsed)
	assert.Equal(t, 1, mockService.executeCalls)
}

func TestGRPCEngine_Execute_Failure(t *testing.T) {
	server, mockService := startMockServer(t, "50075")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50075")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, info, store, goAPI, querier, gasMeter := createTestEnvironment()

	mockService.shouldFailExecute = true
	_, _, err = engine.Execute(checksum, env, info, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock execute error")
}

func TestGRPCEngine_Query_Success(t *testing.T) {
	server, mockService := startMockServer(t, "50076")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50076")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	// Instantiate the contract first
	_, _, err = engine.Instantiate(checksum, env, wasmvmtypes.MessageInfo{Sender: "creator"}, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)

	// Test query
	queryMsg := []byte(`{"query":"test"}`)
	queryResult, gasUsed, err := engine.Query(checksum, env, queryMsg, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
	assert.NotNil(t, queryResult)
	assert.NotNil(t, queryResult.Ok)
	assert.Equal(t, []byte(`{"data":"query_result"}`), queryResult.Ok)
	assert.Equal(t, uint64(0), gasUsed) // Mock returns 0 for query
	assert.Equal(t, 1, mockService.queryCalls)
}

func TestGRPCEngine_Query_Failure(t *testing.T) {
	server, mockService := startMockServer(t, "50077")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50077")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	mockService.shouldFailQuery = true
	_, _, err = engine.Query(checksum, env, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock query error")
}

func TestGRPCEngine_Migrate_Success(t *testing.T) {
	server, mockService := startMockServer(t, "50078")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50078")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	// Instantiate the contract first
	_, _, err = engine.Instantiate(checksum, env, wasmvmtypes.MessageInfo{Sender: "creator"}, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)

	result, gasUsed, err := engine.Migrate(checksum, env, []byte(`{"migrate":"data"}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Ok)
	assert.Equal(t, []byte("migrate_result"), result.Ok.Data)
	assert.Equal(t, uint64(75000), gasUsed)
	assert.Equal(t, 1, mockService.migrateCalls)
}

func TestGRPCEngine_Migrate_Failure(t *testing.T) {
	server, mockService := startMockServer(t, "50079")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50079")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	mockService.shouldFailMigrate = true
	_, gasUsed, err := engine.Migrate(checksum, env, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock migrate error")
	assert.Equal(t, uint64(600), gasUsed)
}

func TestGRPCEngine_MigrateWithInfo(t *testing.T) {
	server, _ := startMockServer(t, "50080")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50080")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	// Instantiate the contract first
	_, _, err = engine.Instantiate(checksum, env, wasmvmtypes.MessageInfo{Sender: "creator"}, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)

	migrateInfo := wasmvmtypes.MigrateInfo{
		Sender: "cosmos1test",
	}

	// MigrateWithInfo should delegate to Migrate for now
	result, gasUsed, err := engine.MigrateWithInfo(checksum, env, []byte(`{"migrate":"data"}`), migrateInfo, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint64(75000), gasUsed)
}

func TestGRPCEngine_Sudo_Success(t *testing.T) {
	server, mockService := startMockServer(t, "50081")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50081")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	// Instantiate the contract first
	_, _, err = engine.Instantiate(checksum, env, wasmvmtypes.MessageInfo{Sender: "creator"}, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)

	result, gasUsed, err := engine.Sudo(checksum, env, []byte(`{"sudo":"command"}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Ok)
	assert.Equal(t, []byte("sudo_result"), result.Ok.Data)
	assert.Equal(t, uint64(60000), gasUsed)
	assert.Equal(t, 1, mockService.sudoCalls)
}

func TestGRPCEngine_Sudo_Failure(t *testing.T) {
	server, mockService := startMockServer(t, "50082")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50082")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	mockService.shouldFailSudo = true
	_, gasUsed, err := engine.Sudo(checksum, env, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock sudo error")
	assert.Equal(t, uint64(200), gasUsed)
}

func TestGRPCEngine_Reply_Success(t *testing.T) {
	server, mockService := startMockServer(t, "50083")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50083")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	// Instantiate the contract first
	_, _, err = engine.Instantiate(checksum, env, wasmvmtypes.MessageInfo{Sender: "creator"}, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)

	reply := wasmvmtypes.Reply{
		ID:      1,
		GasUsed: 100,
		Result: wasmvmtypes.SubMsgResult{
			Ok: &wasmvmtypes.SubMsgResponse{
				Events: []wasmvmtypes.Event{},
				Data:   []byte(`{"success":"true"}`),
			},
		},
	}

	result, gasUsed, err := engine.Reply(checksum, env, reply, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Ok)
	assert.Equal(t, []byte("reply_result"), result.Ok.Data)
	assert.Equal(t, uint64(40000), gasUsed)
	assert.Equal(t, 1, mockService.replyCalls)
}

func TestGRPCEngine_Reply_Failure(t *testing.T) {
	server, mockService := startMockServer(t, "50084")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50084")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	reply := wasmvmtypes.Reply{
		ID:      1,
		GasUsed: 100,
		Result:  wasmvmtypes.SubMsgResult{},
	}

	mockService.shouldFailReply = true
	_, gasUsed, err := engine.Reply(checksum, env, reply, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock reply error")
	assert.Equal(t, uint64(150), gasUsed)
}

// IBC operation tests
func TestGRPCEngine_IBCChannelOpen_Success(t *testing.T) {
	server, mockService := startMockServer(t, "50085")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50085")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	channelOpenMsg := wasmvmtypes.IBCChannelOpenMsg{
		OpenInit: &wasmvmtypes.IBCOpenInit{
			Channel: wasmvmtypes.IBCChannel{
				Endpoint: wasmvmtypes.IBCEndpoint{
					PortID:    "wasm.cosmos1test",
					ChannelID: "channel-0",
				},
				CounterpartyEndpoint: wasmvmtypes.IBCEndpoint{
					PortID:    "transfer",
					ChannelID: "channel-1",
				},
				Order:   wasmvmtypes.Unordered,
				Version: "ics20-1",
			},
		},
	}

	result, gasUsed, err := engine.IBCChannelOpen(checksum, env, channelOpenMsg, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint64(100), gasUsed)
	assert.Equal(t, 1, mockService.ibcCalls)
}

func TestGRPCEngine_IBCChannelConnect_Success(t *testing.T) {
	server, _ := startMockServer(t, "50086")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50086")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	channelConnectMsg := wasmvmtypes.IBCChannelConnectMsg{
		OpenAck: &wasmvmtypes.IBCOpenAck{
			Channel: wasmvmtypes.IBCChannel{
				Endpoint: wasmvmtypes.IBCEndpoint{
					PortID:    "wasm.cosmos1test",
					ChannelID: "channel-0",
				},
				CounterpartyEndpoint: wasmvmtypes.IBCEndpoint{
					PortID:    "transfer",
					ChannelID: "channel-1",
				},
				Order:   wasmvmtypes.Unordered,
				Version: "ics20-1",
			},
			CounterpartyVersion: "ics20-1",
		},
	}

	basicResult, gasUsed, err := engine.IBCChannelConnect(checksum, env, channelConnectMsg, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
	assert.NotNil(t, basicResult)
	assert.Equal(t, uint64(100), gasUsed)
}

func TestGRPCEngine_IBCChannelClose_Success(t *testing.T) {
	server, _ := startMockServer(t, "50087")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50087")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	channelCloseMsg := wasmvmtypes.IBCChannelCloseMsg{
		CloseInit: &wasmvmtypes.IBCCloseInit{
			Channel: wasmvmtypes.IBCChannel{
				Endpoint: wasmvmtypes.IBCEndpoint{
					PortID:    "wasm.cosmos1test",
					ChannelID: "channel-0",
				},
				CounterpartyEndpoint: wasmvmtypes.IBCEndpoint{
					PortID:    "transfer",
					ChannelID: "channel-1",
				},
				Order:   wasmvmtypes.Unordered,
				Version: "ics20-1",
			},
		},
	}

	basicResult, gasUsed, err := engine.IBCChannelClose(checksum, env, channelCloseMsg, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
	assert.NotNil(t, basicResult)
	assert.Equal(t, uint64(100), gasUsed)
}

func TestGRPCEngine_IBCPacketReceive_Success(t *testing.T) {
	server, _ := startMockServer(t, "50088")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50088")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	packetReceiveMsg := wasmvmtypes.IBCPacketReceiveMsg{
		Packet: wasmvmtypes.IBCPacket{
			Data: []byte(`{"amount":"100","denom":"token"}`),
			Src: wasmvmtypes.IBCEndpoint{
				PortID:    "transfer",
				ChannelID: "channel-1",
			},
			Dest: wasmvmtypes.IBCEndpoint{
				PortID:    "wasm.cosmos1test",
				ChannelID: "channel-0",
			},
			Sequence: 1,
			Timeout: wasmvmtypes.IBCTimeout{
				Timestamp: 0,
			},
		},
		Relayer: "cosmos1relayer",
	}

	receiveResult, gasUsed, err := engine.IBCPacketReceive(checksum, env, packetReceiveMsg, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
	assert.NotNil(t, receiveResult)
	assert.Equal(t, uint64(100), gasUsed)
}

func TestGRPCEngine_IBCPacketAck_Success(t *testing.T) {
	server, _ := startMockServer(t, "50089")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50089")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	packetAckMsg := wasmvmtypes.IBCPacketAckMsg{
		OriginalPacket: wasmvmtypes.IBCPacket{
			Data: []byte(`{"amount":"100","denom":"token"}`),
			Src: wasmvmtypes.IBCEndpoint{
				PortID:    "wasm.cosmos1test",
				ChannelID: "channel-0",
			},
			Dest: wasmvmtypes.IBCEndpoint{
				PortID:    "transfer",
				ChannelID: "channel-1",
			},
			Sequence: 1,
			Timeout: wasmvmtypes.IBCTimeout{
				Timestamp: 0,
			},
		},
		Acknowledgement: wasmvmtypes.IBCAcknowledgement{
			Data: []byte(`{"result":"success"}`),
		},
		Relayer: "cosmos1relayer",
	}

	basicResult, gasUsed, err := engine.IBCPacketAck(checksum, env, packetAckMsg, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
	assert.NotNil(t, basicResult)
	assert.Equal(t, uint64(100), gasUsed)
}

func TestGRPCEngine_IBCPacketTimeout_Success(t *testing.T) {
	server, _ := startMockServer(t, "50090")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50090")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	packetTimeoutMsg := wasmvmtypes.IBCPacketTimeoutMsg{
		Packet: wasmvmtypes.IBCPacket{
			Data: []byte(`{"amount":"100","denom":"token"}`),
			Src: wasmvmtypes.IBCEndpoint{
				PortID:    "wasm.cosmos1test",
				ChannelID: "channel-0",
			},
			Dest: wasmvmtypes.IBCEndpoint{
				PortID:    "transfer",
				ChannelID: "channel-1",
			},
			Sequence: 1,
			Timeout: wasmvmtypes.IBCTimeout{
				Timestamp: 0,
			},
		},
		Relayer: "cosmos1relayer",
	}

	basicResult, gasUsed, err := engine.IBCPacketTimeout(checksum, env, packetTimeoutMsg, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
	assert.NotNil(t, basicResult)
	assert.Equal(t, uint64(100), gasUsed)
}

func TestGRPCEngine_IBCSourceCallback_Success(t *testing.T) {
	server, _ := startMockServer(t, "50091")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50091")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	sourceCallbackMsg := wasmvmtypes.IBCSourceCallbackMsg{
		Acknowledgement: &wasmvmtypes.IBCAckCallbackMsg{
			Acknowledgement: wasmvmtypes.IBCAcknowledgement{
				Data: []byte(`{"success":"true"}`),
			},
			OriginalPacket: wasmvmtypes.IBCPacket{
				Data: []byte(`{"amount":"100","denom":"token"}`),
				Src: wasmvmtypes.IBCEndpoint{
					PortID:    "wasm.cosmos1test",
					ChannelID: "channel-0",
				},
				Dest: wasmvmtypes.IBCEndpoint{
					PortID:    "transfer",
					ChannelID: "channel-1",
				},
				Sequence: 1,
				Timeout: wasmvmtypes.IBCTimeout{
					Timestamp: 0,
				},
			},
			Relayer: "cosmos1relayer",
		},
	}

	basicResult, gasUsed, err := engine.IBCSourceCallback(checksum, env, sourceCallbackMsg, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
	assert.NotNil(t, basicResult)
	assert.Equal(t, uint64(100), gasUsed)
}

func TestGRPCEngine_IBCDestinationCallback_Success(t *testing.T) {
	server, _ := startMockServer(t, "50092")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50092")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	destCallbackMsg := wasmvmtypes.IBCDestinationCallbackMsg{
		Ack: wasmvmtypes.IBCAcknowledgement{
			Data: []byte(`{"success":"true"}`),
		},
		Packet: wasmvmtypes.IBCPacket{
			Data: []byte(`{"amount":"100","denom":"token"}`),
			Src: wasmvmtypes.IBCEndpoint{
				PortID:    "transfer",
				ChannelID: "channel-1",
			},
			Dest: wasmvmtypes.IBCEndpoint{
				PortID:    "wasm.cosmos1test",
				ChannelID: "channel-0",
			},
			Sequence: 1,
			Timeout: wasmvmtypes.IBCTimeout{
				Timestamp: 0,
			},
		},
	}

	basicResult, gasUsed, err := engine.IBCDestinationCallback(checksum, env, destCallbackMsg, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
	assert.NotNil(t, basicResult)
	assert.Equal(t, uint64(100), gasUsed)
}

func TestGRPCEngine_IBCOperations_Failure(t *testing.T) {
	server, mockService := startMockServer(t, "50093")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50093")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	mockService.shouldFailIBCOperations = true

	// Test all IBC operations fail
	channelOpenMsg := wasmvmtypes.IBCChannelOpenMsg{}
	_, gasUsed, err := engine.IBCChannelOpen(checksum, env, channelOpenMsg, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock ibc channel open error")
	assert.Equal(t, uint64(50), gasUsed)

	channelConnectMsg := wasmvmtypes.IBCChannelConnectMsg{}
	_, gasUsed, err = engine.IBCChannelConnect(checksum, env, channelConnectMsg, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock ibc channel connect error")
	assert.Equal(t, uint64(50), gasUsed)
}

// Concurrent access tests
func TestGRPCEngine_ConcurrentAccess(t *testing.T) {
	server, _ := startMockServer(t, "50094")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50094")
	require.NoError(t, err)
	defer engine.Cleanup()

	// Store some code first
	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, info, store, goAPI, querier, gasMeter := createTestEnvironment()

	// Instantiate the contract first
	_, _, err = engine.Instantiate(checksum, env, info, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)

	// Run multiple operations concurrently
	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*3)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(3)

		// Concurrent executions
		go func() {
			defer wg.Done()
			_, _, err := engine.Execute(checksum, env, info, []byte(`{"action":"test"}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
			if err != nil {
				errors <- err
			}
		}()

		// Concurrent queries
		go func() {
			defer wg.Done()
			_, _, err := engine.Query(checksum, env, []byte(`{"query":"test"}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
			if err != nil {
				errors <- err
			}
		}()

		// Concurrent metrics calls
		go func() {
			defer wg.Done()
			_, err := engine.GetMetrics()
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}
}

// Performance and timeout tests
func TestGRPCEngine_OperationTimeout(t *testing.T) {
	server, mockService := startMockServer(t, "50095")
	defer server.Stop()

	// Set a delay to simulate slow operations
	mockService.SetOperationDelay(100 * time.Millisecond)

	engine, err := NewGRPCEngine("localhost:50095")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")

	// Test that operations still complete with reasonable delays
	start := time.Now()
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.True(t, elapsed >= 100*time.Millisecond, "Should take at least the simulated delay")
	assert.NotEmpty(t, checksum)
}

// Edge case tests
func TestGRPCEngine_LargePayloads(t *testing.T) {
	server, _ := startMockServer(t, "50096")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50096")
	require.NoError(t, err)
	defer engine.Cleanup()

	// Test with large wasm code
	largeWasmCode := make([]byte, 1024*1024) // 1MB
	for i := range largeWasmCode {
		largeWasmCode[i] = byte(i % 256)
	}

	checksum, _, err := engine.StoreCode(largeWasmCode, 10000000)
	require.NoError(t, err)
	assert.NotEmpty(t, checksum)

	// Verify we can retrieve the large code
	retrievedCode, err := engine.GetCode(checksum)
	require.NoError(t, err)
	assert.Equal(t, largeWasmCode, []byte(retrievedCode))
}

func TestGRPCEngine_EmptyAndNilInputs(t *testing.T) {
	server, _ := startMockServer(t, "50097")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50097")
	require.NoError(t, err)
	defer engine.Cleanup()

	// Test with minimal code (empty code would be rejected)
	minimalCode := []byte("minimal")
	checksum, _, err := engine.StoreCode(minimalCode, 1000000)
	require.NoError(t, err)
	assert.NotEmpty(t, checksum)

	env, info, store, goAPI, querier, gasMeter := createTestEnvironment()

	// Instantiate the contract first
	_, _, err = engine.Instantiate(checksum, env, info, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)

	// Test empty messages
	_, _, err = engine.Execute(checksum, env, info, []byte{}, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)

	_, _, err = engine.Query(checksum, env, []byte{}, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)
}

func TestGRPCEngine_InvalidJSON(t *testing.T) {
	server, _ := startMockServer(t, "50098")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50098")
	require.NoError(t, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(t, err)

	env, info, store, goAPI, querier, gasMeter := createTestEnvironment()

	// Instantiate the contract first
	_, _, err = engine.Instantiate(checksum, env, info, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err)

	// Test invalid JSON - should still work as the mock doesn't validate JSON
	invalidJSON := []byte(`{"invalid": json}`)
	_, _, err = engine.Execute(checksum, env, info, invalidJSON, store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(t, err) // Mock server handles this gracefully
}

// Cleanup tests
func TestGRPCEngine_Cleanup(t *testing.T) {
	server, _ := startMockServer(t, "50099")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50099")
	require.NoError(t, err)

	// Test that cleanup doesn't panic
	engine.Cleanup()

	// Test that cleanup is idempotent
	engine.Cleanup()
}

// Integration test that tries to connect to actual port 50051
func TestGRPCEngine_RealPort50051(t *testing.T) {
	// This test will only pass if there's actually a WasmVM service running on port 50051
	t.Run("ConnectionTest", func(t *testing.T) {
		engine, err := NewGRPCEngine("localhost:50051")
		if err != nil {
			t.Skipf("No WasmVM service running on port 50051: %v", err)
			return
		}
		defer engine.Cleanup()

		t.Log("Successfully connected to WasmVM service on port 50051")

		// Try to get metrics (this should work even without stored code)
		metrics, err := engine.GetMetrics()
		if err != nil {
			t.Logf("GetMetrics failed (expected if service doesn't implement it): %v", err)
		} else {
			t.Logf("Got metrics: %+v", metrics)
		}
	})
}

// Benchmark tests
func BenchmarkGRPCEngine_StoreCode(b *testing.B) {
	server, _ := startMockServer(b, "50100")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50100")
	require.NoError(b, err)
	defer engine.Cleanup()

	wasmCode := []byte("fake wasm code for benchmarking")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := engine.StoreCode(wasmCode, 1000000)
		require.NoError(b, err)
	}
}

func BenchmarkGRPCEngine_Execute(b *testing.B) {
	server, _ := startMockServer(b, "50101")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50101")
	require.NoError(b, err)
	defer engine.Cleanup()

	// Store code first
	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(b, err)

	env, info, store, goAPI, querier, gasMeter := createTestEnvironment()

	// Instantiate the contract
	_, _, err = engine.Instantiate(checksum, env, info, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := engine.Execute(checksum, env, info, []byte(`{"action":"benchmark"}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
		require.NoError(b, err)
	}
}

func BenchmarkGRPCEngine_Query(b *testing.B) {
	server, _ := startMockServer(b, "50102")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50102")
	require.NoError(b, err)
	defer engine.Cleanup()

	// Store code first
	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(b, err)

	env, _, store, goAPI, querier, gasMeter := createTestEnvironment()

	// Instantiate the contract
	_, _, err = engine.Instantiate(checksum, env, wasmvmtypes.MessageInfo{Sender: "creator"}, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := engine.Query(checksum, env, []byte(`{"query":"benchmark"}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
		require.NoError(b, err)
	}
}

func BenchmarkGRPCEngine_ConcurrentOperations(b *testing.B) {
	server, _ := startMockServer(b, "50103")
	defer server.Stop()

	engine, err := NewGRPCEngine("localhost:50103")
	require.NoError(b, err)
	defer engine.Cleanup()

	// Store code first
	wasmCode := []byte("fake wasm code")
	checksum, _, err := engine.StoreCode(wasmCode, 1000000)
	require.NoError(b, err)

	env, info, store, goAPI, querier, gasMeter := createTestEnvironment()

	// Instantiate the contract
	_, _, err = engine.Instantiate(checksum, env, info, []byte(`{}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Alternate between execute and query
			if b.N%2 == 0 {
				_, _, err := engine.Execute(checksum, env, info, []byte(`{"action":"benchmark"}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
				require.NoError(b, err)
			} else {
				_, _, err := engine.Query(checksum, env, []byte(`{"query":"benchmark"}`), store, goAPI, querier, gasMeter, 1000000, wasmvmtypes.UFraction{})
				require.NoError(b, err)
			}
		}
	})
}
