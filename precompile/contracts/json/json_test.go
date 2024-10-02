package json_test

import (
	"math/big"
	"testing"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/precompile/contracts/json"
	"github.com/CosmWasm/wasmd/precompile/registry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/stretchr/testify/require"
)

func TestExtractAsBytes(t *testing.T) {
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)

	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}
	p, err := json.NewContract()
	require.Nil(t, err)
	method := json.ABI.Methods[json.ExtractAsBytesMethod]
	suppliedGas := uint64(10_000_000)

	for _, test := range []struct {
		body           []byte
		expectedOutput []byte
	}{
		{
			[]byte("{\"key\":1}"),
			[]byte("1"),
		}, {
			[]byte("{\"key\":\"1\"}"),
			[]byte("1"),
		}, {
			[]byte("{\"key\":[1,2,3]}"),
			[]byte("[1,2,3]"),
		}, {
			[]byte("{\"key\":{\"nested\":1}}"),
			[]byte("{\"nested\":1}"),
		},
	} {
		args, err := method.Inputs.Pack(test.body, "key")
		require.Nil(t, err)
		res, _, err := p.Run(&evm, registry.JsonContractAddress, registry.JsonContractAddress,
			append(method.ID, args...),
			suppliedGas,
			false,
			nil,
		)
		require.Nil(t, err)
		output, err := method.Outputs.Unpack(res)
		require.Nil(t, err)
		require.Equal(t, 1, len(output))
		require.Equal(t, output[0].([]byte), test.expectedOutput)
	}
}

func TestExtractAsBytesList(t *testing.T) {
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)

	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}
	p, err := json.NewContract()
	require.Nil(t, err)
	method := json.ABI.Methods[json.ExtractAsBytesListMethod]
	suppliedGas := uint64(10_000_000)

	for _, test := range []struct {
		body           []byte
		expectedOutput [][]byte
	}{
		{
			[]byte("{\"key\":[],\"key2\":1}"),
			[][]byte{},
		}, {
			[]byte("{\"key\":[1,2,3],\"key2\":1}"),
			[][]byte{[]byte("1"), []byte("2"), []byte("3")},
		}, {
			[]byte("{\"key\":[\"1\", \"2\"],\"key2\":1}"),
			[][]byte{[]byte("\"1\""), []byte("\"2\"")},
		}, {
			[]byte("{\"key\":[{\"nested\":1}, {\"nested\":2}],\"key2\":1}"),
			[][]byte{[]byte("{\"nested\":1}"), []byte("{\"nested\":2}")},
		},
	} {
		args, err := method.Inputs.Pack(test.body, "key")
		require.Nil(t, err)
		res, _, err := p.Run(&evm, registry.JsonContractAddress, registry.JsonContractAddress,
			append(method.ID, args...),
			suppliedGas,
			false,
			nil,
		)
		require.Nil(t, err)
		output, err := method.Outputs.Unpack(res)
		require.Nil(t, err)
		require.Equal(t, 1, len(output))
		require.Equal(t, output[0].([][]byte), test.expectedOutput)
	}
}

func TestExtractAsUint256(t *testing.T) {
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)

	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}
	p, err := json.NewContract()
	require.Nil(t, err)
	method := json.ABI.Methods[json.ExtractAsUint256Method]
	suppliedGas := uint64(10_000_000)
	n := new(big.Int)

	n.SetString("12345678901234567890", 10)
	for _, test := range []struct {
		body           []byte
		expectedOutput *big.Int
	}{
		{
			[]byte("{\"key\":\"12345678901234567890\"}"),
			n,
		}, {
			[]byte("{\"key\":\"0\"}"),
			big.NewInt(0),
		},
	} {
		args, err := method.Inputs.Pack(test.body, "key")
		require.Nil(t, err)
		res, _, err := p.Run(&evm, registry.JsonContractAddress, registry.JsonContractAddress,
			append(method.ID, args...),
			suppliedGas,
			false,
			nil,
		)
		require.Nil(t, err)
		output, err := method.Outputs.Unpack(res)
		require.Nil(t, err)
		require.Equal(t, 1, len(output))
		require.Equal(t, 0, output[0].(*big.Int).Cmp(test.expectedOutput))
	}
}
