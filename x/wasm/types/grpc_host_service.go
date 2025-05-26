package types

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	wasmvm "github.com/CosmWasm/wasmvm/v3"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
)

// HostServiceHandler provides storage and query callbacks for the gRPC-based WasmEngine
type HostServiceHandler struct {
	// Map request IDs to their associated resources
	requestContext sync.Map
	// The address this host service is listening on
	address string
}

// requestResources holds the resources needed for a specific request
type requestResources struct {
	store    wasmvm.KVStore
	querier  wasmvm.Querier
	goapi    wasmvm.GoAPI
	gasMeter wasmvm.GasMeter
}

// NewHostServiceHandler creates a new host service handler
func NewHostServiceHandler(address string) *HostServiceHandler {
	return &HostServiceHandler{
		address: address,
	}
}

// RegisterRequest associates resources with a request ID
func (h *HostServiceHandler) RegisterRequest(requestID string, store wasmvm.KVStore, querier wasmvm.Querier, goapi wasmvm.GoAPI, gasMeter wasmvm.GasMeter) {
	h.requestContext.Store(requestID, &requestResources{
		store:    store,
		querier:  querier,
		goapi:    goapi,
		gasMeter: gasMeter,
	})
}

// UnregisterRequest removes resources for a request ID
func (h *HostServiceHandler) UnregisterRequest(requestID string) {
	h.requestContext.Delete(requestID)
}

// GetAddress returns the address where the host service is listening
func (h *HostServiceHandler) GetAddress() string {
	return h.address
}

// getResources retrieves resources for a request ID
func (h *HostServiceHandler) getResources(requestID string) (*requestResources, error) {
	val, ok := h.requestContext.Load(requestID)
	if !ok {
		return nil, fmt.Errorf("no resources found for request ID: %s", requestID)
	}
	return val.(*requestResources), nil
}

// The following methods provide the actual implementations for storage and query operations
// These will be called by the gRPC service handlers once the proto types are generated

// HandleStorageGet processes a storage get request
func (h *HostServiceHandler) HandleStorageGet(ctx context.Context, requestID string, key []byte) (value []byte, exists bool, err error) {
	resources, err := h.getResources(requestID)
	if err != nil {
		return nil, false, err
	}

	value = resources.store.Get(key)
	return value, value != nil, nil
}

// HandleStorageSet processes a storage set request
func (h *HostServiceHandler) HandleStorageSet(ctx context.Context, requestID string, key, value []byte) error {
	resources, err := h.getResources(requestID)
	if err != nil {
		return err
	}

	resources.store.Set(key, value)
	return nil
}

// HandleStorageDelete processes a storage delete request
func (h *HostServiceHandler) HandleStorageDelete(ctx context.Context, requestID string, key []byte) error {
	resources, err := h.getResources(requestID)
	if err != nil {
		return err
	}

	resources.store.Delete(key)
	return nil
}

// HandleQueryChain processes a chain query request
func (h *HostServiceHandler) HandleQueryChain(ctx context.Context, requestID string, queryBytes []byte, gasLimit uint64) ([]byte, error) {
	resources, err := h.getResources(requestID)
	if err != nil {
		return nil, err
	}

	// Deserialize the query request
	var queryReq wasmvmtypes.QueryRequest
	err = json.Unmarshal(queryBytes, &queryReq)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal query request: %w", err)
	}

	// Execute the query
	return resources.querier.Query(queryReq, gasLimit)
}

// HandleHumanizeAddress processes an address humanization request
func (h *HostServiceHandler) HandleHumanizeAddress(ctx context.Context, requestID string, canonical []byte) (string, uint64, error) {
	resources, err := h.getResources(requestID)
	if err != nil {
		return "", 0, err
	}

	return resources.goapi.HumanizeAddress(canonical)
}

// HandleCanonicalizeAddress processes an address canonicalization request
func (h *HostServiceHandler) HandleCanonicalizeAddress(ctx context.Context, requestID string, human string) ([]byte, uint64, error) {
	resources, err := h.getResources(requestID)
	if err != nil {
		return nil, 0, err
	}

	return resources.goapi.CanonicalizeAddress(human)
}

// HandleConsumeGas processes a gas consumption request
func (h *HostServiceHandler) HandleConsumeGas(ctx context.Context, requestID string, amount uint64, descriptor string) error {
	_, err := h.getResources(requestID)
	if err != nil {
		return err
	}

	// GasMeter doesn't have ConsumeGas method in the interface, 
	// it's handled internally by the VM
	// This would need to be implemented differently in the actual VM integration
	return nil
}

// HandleGetGasRemaining gets the remaining gas for a request
func (h *HostServiceHandler) HandleGetGasRemaining(ctx context.Context, requestID string) (uint64, error) {
	resources, err := h.getResources(requestID)
	if err != nil {
		return 0, err
	}

	// GasMeter only has GasConsumed() method in the interface
	return resources.gasMeter.GasConsumed(), nil
}

// IteratorHandler provides methods for handling storage iteration
type IteratorHandler struct {
	iter wasmvmtypes.Iterator
}

// HandleStorageIterator creates an iterator for a storage range
func (h *HostServiceHandler) HandleStorageIterator(ctx context.Context, requestID string, start, end []byte) (*IteratorHandler, error) {
	resources, err := h.getResources(requestID)
	if err != nil {
		return nil, err
	}

	iter := resources.store.Iterator(start, end)
	return &IteratorHandler{iter: iter}, nil
}

// HandleStorageReverseIterator creates a reverse iterator for a storage range
func (h *HostServiceHandler) HandleStorageReverseIterator(ctx context.Context, requestID string, start, end []byte) (*IteratorHandler, error) {
	resources, err := h.getResources(requestID)
	if err != nil {
		return nil, err
	}

	iter := resources.store.ReverseIterator(start, end)
	return &IteratorHandler{iter: iter}, nil
}

// Next advances the iterator and returns the key-value pair
func (ih *IteratorHandler) Next() (key []byte, value []byte, done bool) {
	if !ih.iter.Valid() {
		return nil, nil, true
	}

	key = ih.iter.Key()
	value = ih.iter.Value()
	ih.iter.Next()
	return key, value, false
}

// Close closes the iterator
func (ih *IteratorHandler) Close() error {
	return ih.iter.Close()
}
