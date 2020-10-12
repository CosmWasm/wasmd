package types

import (
	"github.com/cosmos/cosmos-sdk/store"
	"github.com/cosmos/cosmos-sdk/store/iavl"
	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
	iavl2 "github.com/tendermint/iavl"
	dbm "github.com/tendermint/tm-db"
	"testing"
)

// This is modeled close to
// https://github.com/CosmWasm/cosmwasm-plus/blob/f97a7de44b6a930fd1d5179ee6f95b786a532f32/packages/storage-plus/src/prefix.rs#L183
// and designed to ensure the IAVL store handles bounds the same way as the mock storage we use in Rust contract tests
func TestIavlRangeBounds(t *testing.T) {
	db := dbm.NewMemDB()
	tree, err := iavl2.NewMutableTree(db, 50)
	require.NoError(t, err)
	store := iavl.UnsafeNewStore(tree)

	expected := []KV{
		{[]byte("bar"), []byte("1")},
		{[]byte("ra"), []byte("2")},
		{[]byte("zi"), []byte("3")},
	}
	reversed := []KV{
		{[]byte("zi"), []byte("3")},
		{[]byte("ra"), []byte("2")},
		{[]byte("bar"), []byte("1")},
	}

	// set up test cases, like `ensure_proper_range_bounds` in `cw-storage-plus`
	for _, kv := range expected {
		store.Set(kv.Key, kv.Value)
	}

	ascAll := store.Iterator(nil, nil)
	assert.Equal(t, expected, consume(ascAll))

	descAll := store.ReverseIterator(nil, nil)
	assert.Equal(t, reversed, consume(descAll))
}

type KV struct {
	Key []byte
	Value []byte
}

func consume(itr store.Iterator) []KV {
	defer itr.Close()

	var res []KV
	for ; itr.Valid(); itr.Next() {
		k, v := itr.Key(), itr.Value()
		res = append(res, KV{k, v})
	}
	return res
}