package types

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	wasmvm "github.com/CosmWasm/wasmvm/v3"
	wasmgrpc "github.com/CosmWasm/wasmvm/v3/rpc"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
)

// grpcEngineEnhanced is an enhanced WasmEngine implementation that supports storage and query operations
type grpcEngineEnhanced struct {
	client          wasmgrpc.WasmVMServiceClient
	conn            *grpc.ClientConn
	hostService     *HostServiceHandler
	hostServiceAddr string
	idCounter       uint64
	idMutex         sync.Mutex
}

// NewGRPCEngineEnhanced creates an enhanced gRPC engine with storage support
func NewGRPCEngineEnhanced(vmAddr string, hostServiceAddr string) (WasmEngine, error) {
	if vmAddr == "" {
		vmAddr = "localhost:50051"
	}
	if hostServiceAddr == "" {
		hostServiceAddr = "localhost:50052"
	}

	conn, err := grpc.NewClient(vmAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	client := wasmgrpc.NewWasmVMServiceClient(conn)
	hostService := NewHostServiceHandler(hostServiceAddr)

	return &grpcEngineEnhanced{
		client:          client,
		conn:            conn,
		hostService:     hostService,
		hostServiceAddr: hostServiceAddr,
	}, nil
}

// generateRequestID creates a unique request ID for tracking resources
func (g *grpcEngineEnhanced) generateRequestID() string {
	g.idMutex.Lock()
	defer g.idMutex.Unlock()
	g.idCounter++
	return fmt.Sprintf("req_%d_%d", time.Now().Unix(), g.idCounter)
}

// prepareRequest registers resources and returns the request ID
func (g *grpcEngineEnhanced) prepareRequest(store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter) string {
	requestID := g.generateRequestID()
	g.hostService.RegisterRequest(requestID, store, querier, goapi, gasMeter)
	return requestID
}

// cleanupRequest unregisters resources after request completion
func (g *grpcEngineEnhanced) cleanupRequest(requestID string) {
	g.hostService.UnregisterRequest(requestID)
}

// StoreCode implementation
func (g *grpcEngineEnhanced) StoreCode(code wasmvm.WasmCode, gasLimit uint64) (wasmvm.Checksum, uint64, error) {
	req := &wasmgrpc.LoadModuleRequest{ModuleBytes: code}
	resp, err := g.client.LoadModule(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, 0, errors.New(resp.Error)
	}
	return wasmvmtypes.Checksum(resp.Checksum), 0, nil
}

func (g *grpcEngineEnhanced) StoreCodeUnchecked(code wasmvm.WasmCode) (wasmvm.Checksum, error) {
	req := &wasmgrpc.LoadModuleRequest{ModuleBytes: code}
	resp, err := g.client.LoadModule(context.Background(), req)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return wasmvmtypes.Checksum(resp.Checksum), nil
}

func (g *grpcEngineEnhanced) SimulateStoreCode(code wasmvm.WasmCode, gasLimit uint64) (wasmvm.Checksum, uint64, error) {
	return g.StoreCode(code, gasLimit)
}

func (g *grpcEngineEnhanced) AnalyzeCode(checksum wasmvmtypes.Checksum) (*wasmvmtypes.AnalysisReport, error) {
	req := &wasmgrpc.AnalyzeCodeRequest{Checksum: string(checksum)}
	resp, err := g.client.AnalyzeCode(context.Background(), req)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return &wasmvmtypes.AnalysisReport{
		RequiredCapabilities: strings.Join(resp.RequiredCapabilities, ","),
		HasIBCEntryPoints:    resp.HasIbcEntryPoints,
	}, nil
}

// Instantiate with storage support
func (g *grpcEngineEnhanced) Instantiate(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	info wasmvmtypes.MessageInfo,
	initMsg []byte,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.ContractResult, uint64, error) {
	// Register resources for this request
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	// Create context with callback service information
	ctx := &wasmgrpc.Context{
		BlockHeight: env.Block.Height,
		Sender:      info.Sender,
		ChainId:     env.Block.ChainID,
	}

	// For now, use the standard request until extended proto is generated
	// In the future, this would include the callback service address
	req := &wasmgrpc.InstantiateRequest{
		Checksum:  string(checksum),
		Context:   ctx,
		InitMsg:   initMsg,
		GasLimit:  gasLimit,
		RequestId: requestID,
	}

	resp, err := g.client.Instantiate(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}

	return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp.Data}}, resp.GasUsed, nil
}

// Execute with storage support
func (g *grpcEngineEnhanced) Execute(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	info wasmvmtypes.MessageInfo,
	executeMsg []byte,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.ContractResult, uint64, error) {
	// Register resources for this request
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{
		BlockHeight: env.Block.Height,
		Sender:      info.Sender,
		ChainId:     env.Block.ChainID,
	}

	req := &wasmgrpc.ExecuteRequest{
		ContractId: string(checksum),
		Context:    ctx,
		Msg:        executeMsg,
		GasLimit:   gasLimit,
		RequestId:  requestID,
	}

	resp, err := g.client.Execute(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}

	return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp.Data}}, resp.GasUsed, nil
}

// Query with storage support
func (g *grpcEngineEnhanced) Query(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	queryMsg []byte,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.QueryResult, uint64, error) {
	// Register resources for this request
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{
		BlockHeight: env.Block.Height,
		Sender:      "",
		ChainId:     env.Block.ChainID,
	}

	req := &wasmgrpc.QueryRequest{
		ContractId: string(checksum),
		Context:    ctx,
		QueryMsg:   queryMsg,
		RequestId:  requestID,
	}

	resp, err := g.client.Query(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, 0, errors.New(resp.Error)
	}

	return &wasmvmtypes.QueryResult{Ok: resp.Result}, 0, nil
}

// Migrate with storage support
func (g *grpcEngineEnhanced) Migrate(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	migrateMsg []byte,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.ContractResult, uint64, error) {
	// Register resources for this request
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{
		BlockHeight: env.Block.Height,
		Sender:      "",
		ChainId:     env.Block.ChainID,
	}

	req := &wasmgrpc.MigrateRequest{
		ContractId: string(checksum),
		Checksum:   string(checksum),
		Context:    ctx,
		MigrateMsg: migrateMsg,
		GasLimit:   gasLimit,
		RequestId:  requestID,
	}

	resp, err := g.client.Migrate(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}

	return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp.Data}}, resp.GasUsed, nil
}

func (g *grpcEngineEnhanced) MigrateWithInfo(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	migrateMsg []byte,
	migrateInfo wasmvmtypes.MigrateInfo,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.ContractResult, uint64, error) {
	// For now, use the regular Migrate method
	return g.Migrate(checksum, env, migrateMsg, store, goapi, querier, gasMeter, gasLimit, deserCost)
}

// Sudo with storage support
func (g *grpcEngineEnhanced) Sudo(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	sudoMsg []byte,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.ContractResult, uint64, error) {
	// Register resources for this request
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{
		BlockHeight: env.Block.Height,
		Sender:      "",
		ChainId:     env.Block.ChainID,
	}

	req := &wasmgrpc.SudoRequest{
		ContractId: string(checksum),
		Context:    ctx,
		Msg:        sudoMsg,
		GasLimit:   gasLimit,
		RequestId:  requestID,
	}

	resp, err := g.client.Sudo(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}

	return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp.Data}}, resp.GasUsed, nil
}

// Reply with storage support
func (g *grpcEngineEnhanced) Reply(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	reply wasmvmtypes.Reply,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.ContractResult, uint64, error) {
	// Register resources for this request
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{
		BlockHeight: env.Block.Height,
		Sender:      "",
		ChainId:     env.Block.ChainID,
	}

	replyMsg, err := json.Marshal(reply)
	if err != nil {
		return nil, 0, err
	}

	req := &wasmgrpc.ReplyRequest{
		ContractId: string(checksum),
		Context:    ctx,
		ReplyMsg:   replyMsg,
		GasLimit:   gasLimit,
		RequestId:  requestID,
	}

	resp, err := g.client.Reply(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}

	return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp.Data}}, resp.GasUsed, nil
}

// GetCode retrieves the code for a given checksum
func (g *grpcEngineEnhanced) GetCode(checksum wasmvmtypes.Checksum) (wasmvm.WasmCode, error) {
	req := &wasmgrpc.GetCodeRequest{Checksum: string(checksum)}
	resp, err := g.client.GetCode(context.Background(), req)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return resp.ModuleBytes, nil
}

// Cleanup closes the connection
func (g *grpcEngineEnhanced) Cleanup() {
	if g.conn != nil {
		_ = g.conn.Close()
	}
}

// Pin pins a code to memory
func (g *grpcEngineEnhanced) Pin(checksum wasmvmtypes.Checksum) error {
	req := &wasmgrpc.PinModuleRequest{Checksum: string(checksum)}
	resp, err := g.client.PinModule(context.Background(), req)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

// Unpin unpins a code from memory
func (g *grpcEngineEnhanced) Unpin(checksum wasmvmtypes.Checksum) error {
	req := &wasmgrpc.UnpinModuleRequest{Checksum: string(checksum)}
	resp, err := g.client.UnpinModule(context.Background(), req)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

// GetMetrics returns VM metrics
func (g *grpcEngineEnhanced) GetMetrics() (*wasmvmtypes.Metrics, error) {
	req := &wasmgrpc.GetMetricsRequest{}
	resp, err := g.client.GetMetrics(context.Background(), req)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	if resp.Metrics == nil {
		return nil, errors.New("no metrics returned")
	}
	return &wasmvmtypes.Metrics{
		HitsPinnedMemoryCache:     resp.Metrics.HitsPinnedMemoryCache,
		HitsMemoryCache:           resp.Metrics.HitsMemoryCache,
		HitsFsCache:               resp.Metrics.HitsFsCache,
		Misses:                    resp.Metrics.Misses,
		ElementsPinnedMemoryCache: resp.Metrics.ElementsPinnedMemoryCache,
		ElementsMemoryCache:       resp.Metrics.ElementsMemoryCache,
		SizePinnedMemoryCache:     resp.Metrics.SizePinnedMemoryCache,
		SizeMemoryCache:           resp.Metrics.SizeMemoryCache,
	}, nil
}

// GetPinnedMetrics returns pinned code metrics
func (g *grpcEngineEnhanced) GetPinnedMetrics() (*wasmvmtypes.PinnedMetrics, error) {
	req := &wasmgrpc.GetPinnedMetricsRequest{}
	resp, err := g.client.GetPinnedMetrics(context.Background(), req)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	if resp.PinnedMetrics == nil {
		return nil, errors.New("no pinned metrics returned")
	}

	var perModule []wasmvmtypes.PerModuleEntry
	for checksum, metrics := range resp.PinnedMetrics.PerModule {
		perModule = append(perModule, wasmvmtypes.PerModuleEntry{
			Checksum: wasmvmtypes.Checksum(checksum),
			Metrics: wasmvmtypes.PerModuleMetrics{
				Hits: metrics.Hits,
				Size: metrics.Size,
			},
		})
	}

	return &wasmvmtypes.PinnedMetrics{
		PerModule: perModule,
	}, nil
}

// IBC methods with storage support (simplified implementations for now)
func (g *grpcEngineEnhanced) IBCChannelOpen(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	channel wasmvmtypes.IBCChannelOpenMsg,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.IBCChannelOpenResult, uint64, error) {
	// Register resources for this request
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{BlockHeight: env.Block.Height, Sender: "", ChainId: env.Block.ChainID}
	msgBytes := []byte{} // TODO: serialize channel properly
	req := &wasmgrpc.IbcMsgRequest{Checksum: string(checksum), Context: ctx, Msg: msgBytes, GasLimit: gasLimit, RequestId: requestID}
	resp, err := g.client.IbcChannelOpen(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}
	return &wasmvmtypes.IBCChannelOpenResult{}, resp.GasUsed, nil
}

// Continue with other IBC methods...
// (Similar pattern for IBCChannelConnect, IBCChannelClose, IBCPacketReceive, etc.)
// For brevity, I'll include just the pattern - the full implementation would follow the same approach

func (g *grpcEngineEnhanced) IBCChannelConnect(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	channel wasmvmtypes.IBCChannelConnectMsg,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{BlockHeight: env.Block.Height, Sender: "", ChainId: env.Block.ChainID}
	msgBytes := []byte{} // TODO: serialize channel properly
	req := &wasmgrpc.IbcMsgRequest{Checksum: string(checksum), Context: ctx, Msg: msgBytes, GasLimit: gasLimit, RequestId: requestID}
	resp, err := g.client.IbcChannelConnect(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}
	return &wasmvmtypes.IBCBasicResult{}, resp.GasUsed, nil
}

// Implement remaining IBC methods following the same pattern...

func (g *grpcEngineEnhanced) IBCChannelClose(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	channel wasmvmtypes.IBCChannelCloseMsg,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{BlockHeight: env.Block.Height, Sender: "", ChainId: env.Block.ChainID}
	msgBytes := []byte{} // TODO: serialize channel properly
	req := &wasmgrpc.IbcMsgRequest{Checksum: string(checksum), Context: ctx, Msg: msgBytes, GasLimit: gasLimit, RequestId: requestID}
	resp, err := g.client.IbcChannelClose(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}
	return &wasmvmtypes.IBCBasicResult{}, resp.GasUsed, nil
}

func (g *grpcEngineEnhanced) IBCPacketReceive(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	packet wasmvmtypes.IBCPacketReceiveMsg,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.IBCReceiveResult, uint64, error) {
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{BlockHeight: env.Block.Height, Sender: "", ChainId: env.Block.ChainID}
	msgBytes := []byte{} // TODO: serialize packet properly
	req := &wasmgrpc.IbcMsgRequest{Checksum: string(checksum), Context: ctx, Msg: msgBytes, GasLimit: gasLimit, RequestId: requestID}
	resp, err := g.client.IbcPacketReceive(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}
	return &wasmvmtypes.IBCReceiveResult{}, resp.GasUsed, nil
}

func (g *grpcEngineEnhanced) IBCPacketAck(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	ack wasmvmtypes.IBCPacketAckMsg,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{BlockHeight: env.Block.Height, Sender: "", ChainId: env.Block.ChainID}
	msgBytes := []byte{} // TODO: serialize ack properly
	req := &wasmgrpc.IbcMsgRequest{Checksum: string(checksum), Context: ctx, Msg: msgBytes, GasLimit: gasLimit, RequestId: requestID}
	resp, err := g.client.IbcPacketAck(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}
	return &wasmvmtypes.IBCBasicResult{}, resp.GasUsed, nil
}

func (g *grpcEngineEnhanced) IBCPacketTimeout(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	packet wasmvmtypes.IBCPacketTimeoutMsg,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{BlockHeight: env.Block.Height, Sender: "", ChainId: env.Block.ChainID}
	msgBytes := []byte{} // TODO: serialize packet properly
	req := &wasmgrpc.IbcMsgRequest{Checksum: string(checksum), Context: ctx, Msg: msgBytes, GasLimit: gasLimit, RequestId: requestID}
	resp, err := g.client.IbcPacketTimeout(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}
	return &wasmvmtypes.IBCBasicResult{}, resp.GasUsed, nil
}

func (g *grpcEngineEnhanced) IBCSourceCallback(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	msg wasmvmtypes.IBCSourceCallbackMsg,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{BlockHeight: env.Block.Height, Sender: "", ChainId: env.Block.ChainID}
	msgBytes := []byte{} // TODO: serialize msg properly
	req := &wasmgrpc.IbcMsgRequest{Checksum: string(checksum), Context: ctx, Msg: msgBytes, GasLimit: gasLimit, RequestId: requestID}
	resp, err := g.client.IbcSourceCallback(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}
	return &wasmvmtypes.IBCBasicResult{}, resp.GasUsed, nil
}

func (g *grpcEngineEnhanced) IBCDestinationCallback(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	msg wasmvmtypes.IBCDestinationCallbackMsg,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{BlockHeight: env.Block.Height, Sender: "", ChainId: env.Block.ChainID}
	msgBytes := []byte{} // TODO: serialize msg properly
	req := &wasmgrpc.IbcMsgRequest{Checksum: string(checksum), Context: ctx, Msg: msgBytes, GasLimit: gasLimit, RequestId: requestID}
	resp, err := g.client.IbcDestinationCallback(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}
	return &wasmvmtypes.IBCBasicResult{}, resp.GasUsed, nil
}

// IBC2 methods
func (g *grpcEngineEnhanced) IBC2PacketAck(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	payload wasmvmtypes.IBC2AcknowledgeMsg,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{BlockHeight: env.Block.Height, Sender: "", ChainId: env.Block.ChainID}
	msgBytes := []byte{} // TODO: serialize payload properly
	req := &wasmgrpc.IbcMsgRequest{Checksum: string(checksum), Context: ctx, Msg: msgBytes, GasLimit: gasLimit, RequestId: requestID}
	resp, err := g.client.Ibc2PacketAck(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}
	return &wasmvmtypes.IBCBasicResult{}, resp.GasUsed, nil
}

func (g *grpcEngineEnhanced) IBC2PacketReceive(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	payload wasmvmtypes.IBC2PacketReceiveMsg,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.IBCReceiveResult, uint64, error) {
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{BlockHeight: env.Block.Height, Sender: "", ChainId: env.Block.ChainID}
	msgBytes := []byte{} // TODO: serialize payload properly
	req := &wasmgrpc.IbcMsgRequest{Checksum: string(checksum), Context: ctx, Msg: msgBytes, GasLimit: gasLimit, RequestId: requestID}
	resp, err := g.client.Ibc2PacketReceive(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}
	return &wasmvmtypes.IBCReceiveResult{}, resp.GasUsed, nil
}

func (g *grpcEngineEnhanced) IBC2PacketTimeout(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	packet wasmvmtypes.IBC2PacketTimeoutMsg,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{BlockHeight: env.Block.Height, Sender: "", ChainId: env.Block.ChainID}
	msgBytes := []byte{} // TODO: serialize packet properly
	req := &wasmgrpc.IbcMsgRequest{Checksum: string(checksum), Context: ctx, Msg: msgBytes, GasLimit: gasLimit, RequestId: requestID}
	resp, err := g.client.Ibc2PacketTimeout(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}
	return &wasmvmtypes.IBCBasicResult{}, resp.GasUsed, nil
}

func (g *grpcEngineEnhanced) IBC2PacketSend(
	checksum wasmvmtypes.Checksum,
	env wasmvmtypes.Env,
	packet wasmvmtypes.IBC2PacketSendMsg,
	store wasmvm.KVStore,
	goapi wasmvm.GoAPI,
	querier wasmvm.Querier,
	gasMeter wasmvm.GasMeter,
	gasLimit uint64,
	deserCost wasmvmtypes.UFraction,
) (*wasmvmtypes.IBCBasicResult, uint64, error) {
	requestID := g.prepareRequest(store, goapi, querier, gasMeter)
	defer g.cleanupRequest(requestID)

	ctx := &wasmgrpc.Context{BlockHeight: env.Block.Height, Sender: "", ChainId: env.Block.ChainID}
	msgBytes := []byte{} // TODO: serialize packet properly
	req := &wasmgrpc.IbcMsgRequest{Checksum: string(checksum), Context: ctx, Msg: msgBytes, GasLimit: gasLimit, RequestId: requestID}
	resp, err := g.client.Ibc2PacketSend(context.Background(), req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Error != "" {
		return nil, resp.GasUsed, errors.New(resp.Error)
	}
	return &wasmvmtypes.IBCBasicResult{}, resp.GasUsed, nil
}
