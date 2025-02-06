package main

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// -----------------------------------------------------------------------------
// Dummy Cosmos-SDK-like types and helpers (minimal implementation)
// -----------------------------------------------------------------------------

// Tx is a dummy transaction interface.
type Tx interface{}

// AnteHandler defines the function signature for an ante handler.
type AnteHandler func(ctx Context, tx Tx, simulate bool) (Context, error)

// Context is a minimal imitation of the Cosmos SDK Context.
type Context struct {
	blockHeight int64
	values      map[string]interface{}
}

// NewContext creates a new Context with a given block height.
func NewContext(blockHeight int64) Context {
	return Context{
		blockHeight: blockHeight,
		values:      make(map[string]interface{}),
	}
}

// WithBlockHeight returns a copy of the context with the specified block height.
func (ctx Context) WithBlockHeight(height int64) Context {
	newCtx := ctx.copy()
	newCtx.blockHeight = height
	return newCtx
}

// BlockHeight returns the block height.
func (ctx Context) BlockHeight() int64 {
	return ctx.blockHeight
}

// WithValue returns a copy of the context with the given key/value pair.
func (ctx Context) WithValue(key string, value interface{}) Context {
	newCtx := ctx.copy()
	newCtx.values[key] = value
	return newCtx
}

// Value retrieves a value from the context by key.
func (ctx Context) Value(key string) interface{} {
	return ctx.values[key]
}

// copy creates a shallow copy of the context.
func (ctx Context) copy() Context {
	newValues := make(map[string]interface{})
	for k, v := range ctx.values {
		newValues[k] = v
	}
	return Context{
		blockHeight: ctx.blockHeight,
		values:      newValues,
	}
}

// Uint64ToBigEndian converts a uint64 to an 8-byte big-endian slice.
func Uint64ToBigEndian(n uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, n)
	return b
}

// BigEndianToUint64 converts an 8-byte big-endian slice to a uint64.
func BigEndianToUint64(bz []byte) uint64 {
	if len(bz) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// -----------------------------------------------------------------------------
// Dummy KVStore interfaces and implementations (for testing)
// -----------------------------------------------------------------------------

// KVStore defines a minimal key/value store interface.
type KVStore interface {
	Get(key []byte) ([]byte, error)
	Set(key []byte, value []byte) error
	NewTransaction() KVStoreTransaction
}

// KVStoreTransaction defines a minimal transactional interface.
type KVStoreTransaction interface {
	Get(key []byte) ([]byte, error)
	Set(key []byte, value []byte) error
	Commit() error
	Discard()
}

// dummyKVStore is an in-memory key/value store.
type dummyKVStore struct {
	data map[string][]byte
}

func newDummyKVStore() *dummyKVStore {
	return &dummyKVStore{data: make(map[string][]byte)}
}

func (d *dummyKVStore) Get(key []byte) ([]byte, error) {
	return d.data[string(key)], nil
}

func (d *dummyKVStore) Set(key []byte, value []byte) error {
	d.data[string(key)] = value
	return nil
}

// dummyTransaction implements KVStoreTransaction.
type dummyTransaction struct {
	store     *dummyKVStore
	pending   map[string][]byte
	committed bool
}

func (t *dummyTransaction) Get(key []byte) ([]byte, error) {
	// Check pending changes first.
	if val, ok := t.pending[string(key)]; ok {
		return val, nil
	}
	return t.store.Get(key)
}

func (t *dummyTransaction) Set(key []byte, value []byte) error {
	t.pending[string(key)] = value
	return nil
}

func (t *dummyTransaction) Commit() error {
	for k, v := range t.pending {
		t.store.data[k] = v
	}
	t.committed = true
	return nil
}

func (t *dummyTransaction) Discard() {
	// Discard pending changes if not committed.
	if !t.committed {
		t.pending = nil
	}
}

// dummyKVStoreService is a dummy service that returns our dummy store.
type dummyKVStoreService struct {
	store *dummyKVStore
}

func newDummyKVStoreService() *dummyKVStoreService {
	return &dummyKVStoreService{store: newDummyKVStore()}
}

// OpenKVStore returns a KVStore.
func (d *dummyKVStoreService) OpenKVStore(ctx Context) KVStore {
	return &dummyStoreWrapper{store: d.store}
}

// dummyStoreWrapper wraps dummyKVStore to implement KVStore.
type dummyStoreWrapper struct {
	store *dummyKVStore
}

func (d *dummyStoreWrapper) Get(key []byte) ([]byte, error) {
	return d.store.Get(key)
}

func (d *dummyStoreWrapper) Set(key []byte, value []byte) error {
	return d.store.Set(key, value)
}

func (d *dummyStoreWrapper) NewTransaction() KVStoreTransaction {
	return &dummyTransaction{
		store:   d.store,
		pending: make(map[string][]byte),
	}
}

// -----------------------------------------------------------------------------
// Production Code: CountTXDecorator and Helpers
// -----------------------------------------------------------------------------

// --- Begin added definitions to fix missing references ---
// These definitions provide the missing TX counter key and context helpers.
// In your production code, you may already have these in your types package.
const txCounterKey = "tx_counter"

// TXCounterPrefix is the key under which the tx counter and height are stored.
var TXCounterPrefix = []byte(txCounterKey)

// WithTXCounter returns a new context with the given transaction counter injected.
func WithTXCounter(ctx Context, counter uint32) Context {
	return ctx.WithValue(txCounterKey, counter)
}

// TXCounter retrieves the transaction counter from the context.
// If no value is found, it returns 0.
func TXCounter(ctx Context) uint32 {
	if counter, ok := ctx.Value(txCounterKey).(uint32); ok {
		return counter
	}
	return 0
}
// --- End added definitions ---

// CountTXDecorator is an ante handler that counts the transaction position within a block.
type CountTXDecorator struct {
	storeService *dummyKVStoreService
}

// NewCountTXDecorator creates a new instance of CountTXDecorator.
func NewCountTXDecorator(s *dummyKVStoreService) *CountTXDecorator {
	return &CountTXDecorator{storeService: s}
}

// AnteHandle stores a transaction counter with the current height encoded in the store.
// It lets the application handle global rollback behavior instead of keeping state in the handler itself.
// The counter is passed upstream via the Context (retrieved via TXCounter).
// Simulations do not get a transaction counter assigned.
func (a CountTXDecorator) AnteHandle(ctx Context, tx Tx, simulate bool, next AnteHandler) (Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	store := a.storeService.OpenKVStore(ctx)
	currentHeight := ctx.BlockHeight()

	// Begin a new transaction.
	txn := store.NewTransaction()
	// Ensure the transaction is discarded if not committed.
	defer txn.Discard()

	var txCounter uint32 = 0
	// Load the existing counter (if any).
	bz, err := txn.Get(TXCounterPrefix)
	if err != nil {
		return ctx, errors.New("read tx counter: " + err.Error())
	}
	if bz != nil {
		lastHeight, val := decodeHeightCounter(bz)
		if currentHeight == lastHeight {
			txCounter = val
		}
	}

	// Increment the counter for the current transaction.
	newTxCounter := txCounter + 1
	if err := txn.Set(TXCounterPrefix, encodeHeightCounter(currentHeight, newTxCounter)); err != nil {
		return ctx, errors.New("store tx counter: " + err.Error())
	}

	// Commit the transaction.
	if err := txn.Commit(); err != nil {
		return ctx, errors.New("commit transaction: " + err.Error())
	}

	// Pass the updated counter value via context.
	return next(WithTXCounter(ctx, newTxCounter), tx, simulate)
}

// encodeHeightCounter encodes the block height and transaction counter into a byte slice.
// The first 8 bytes represent the block height (big-endian),
// followed by 4 bytes for the counter.
func encodeHeightCounter(height int64, counter uint32) []byte {
	counterBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(counterBytes, counter)
	return append(Uint64ToBigEndian(uint64(height)), counterBytes...)
}

// decodeHeightCounter decodes a byte slice into the block height and transaction counter.
func decodeHeightCounter(bz []byte) (int64, uint32) {
	if len(bz) < 12 {
		// Return zero values if the slice is too short.
		return 0, 0
	}
	return int64(BigEndianToUint64(bz[0:8])), binary.BigEndian.Uint32(bz[8:])
}

// -----------------------------------------------------------------------------
// Test Functions
// -----------------------------------------------------------------------------

// testSameBlock simulates two transactions in the same block.
func testSameBlock() error {
	storeService := newDummyKVStoreService()
	ctx := NewContext(100)
	decorator := NewCountTXDecorator(storeService)

	// Dummy next handler that returns the context unmodified.
	nextHandler := func(ctx Context, tx Tx, simulate bool) (Context, error) {
		return ctx, nil
	}

	// Execute the handler for the first transaction.
	var err error
	ctx, err = decorator.AnteHandle(ctx, nil, false, nextHandler)
	if err != nil {
		return fmt.Errorf("first transaction failed: %v", err)
	}
	counter1 := TXCounter(ctx)

	// Execute the handler for a second transaction in the same block.
	ctx, err = decorator.AnteHandle(ctx, nil, false, nextHandler)
	if err != nil {
		return fmt.Errorf("second transaction failed: %v", err)
	}
	counter2 := TXCounter(ctx)

	// The second counter should be exactly one more than the first.
	if counter2 != counter1+1 {
		return fmt.Errorf("expected counter %d, got %d", counter1+1, counter2)
	}

	fmt.Printf("testSameBlock passed: counter1=%d, counter2=%d\n", counter1, counter2)
	return nil
}

// testNewBlock simulates a transaction in one block and then in a new block.
func testNewBlock() error {
	storeService := newDummyKVStoreService()
	ctx := NewContext(100)
	decorator := NewCountTXDecorator(storeService)

	nextHandler := func(ctx Context, tx Tx, simulate bool) (Context, error) {
		return ctx, nil
	}

	// First transaction in block 100.
	var err error
	ctx, err = decorator.AnteHandle(ctx, nil, false, nextHandler)
	if err != nil {
		return fmt.Errorf("transaction in block 100 failed: %v", err)
	}
	counter := TXCounter(ctx)
	if counter != 1 {
		return fmt.Errorf("expected counter 1 in block 100, got %d", counter)
	}

	// Simulate a new block.
	ctx = NewContext(101)
	ctx, err = decorator.AnteHandle(ctx, nil, false, nextHandler)
	if err != nil {
		return fmt.Errorf("transaction in block 101 failed: %v", err)
	}
	newCounter := TXCounter(ctx)
	if newCounter != 1 {
		return fmt.Errorf("expected counter 1 in block 101, got %d", newCounter)
	}

	fmt.Printf("testNewBlock passed: block 100 counter=1, block 101 counter=%d\n", newCounter)
	return nil
}

// -----------------------------------------------------------------------------
// Main Function: Run Tests
// -----------------------------------------------------------------------------

func main() {
	if err := testSameBlock(); err != nil {
		fmt.Println("testSameBlock FAILED:", err)
		return
	}
	if err := testNewBlock(); err != nil {
		fmt.Println("testNewBlock FAILED:", err)
		return
	}
	fmt.Println("All tests passed!")
}
