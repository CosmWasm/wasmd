package wasmtesting

import (
	storetypes "cosmossdk.io/store/types"
)

// MockCommitMultiStore mock with a CacheMultiStore to capture commits
type MockCommitMultiStore struct {
	storetypes.CommitMultiStore
	Committed []bool
}

func (m *MockCommitMultiStore) CacheMultiStore() storetypes.CacheMultiStore {
	m.Committed = append(m.Committed, false)
	return &mockCMS{m, &m.Committed[len(m.Committed)-1]}
}

type mockCMS struct {
	*MockCommitMultiStore
	committed *bool
}

func (m *mockCMS) CacheMultiStore() storetypes.CacheMultiStore {
	return m
}

func (m *mockCMS) Write() {
	*m.committed = true
}
