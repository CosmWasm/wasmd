package keeper

import (
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	fuzz "github.com/google/gofuzz"
	tmBytes "github.com/tendermint/tendermint/libs/bytes"
)

func FuzzAddr(m *sdk.AccAddress, c fuzz.Continue) {
	*m = make([]byte, 20)
	c.Read(*m)
}
func FuzzAbsoluteTxPosition(m *types.AbsoluteTxPosition, c fuzz.Continue) {
	m.BlockHeight = int64(c.RandUint64()) // can't be negative
	m.TxIndex = c.RandUint64()
}

func FuzzContractInfo(m *types.ContractInfo, c fuzz.Continue) {
	const maxSize = 1024
	m.CodeID = c.RandUint64()
	FuzzAddr(&m.Creator, c)
	FuzzAddr(&m.Admin, c)
	m.Label = c.RandString()
	m.InitMsg = make([]byte, c.RandUint64()%maxSize)
	c.Read(m.InitMsg)
	c.Fuzz(&m.Created)
	c.Fuzz(&m.LastUpdated)
	m.PreviousCodeID = c.RandUint64()
}
func FuzzStateModel(m *types.Model, c fuzz.Continue) {
	m.Key = tmBytes.HexBytes(c.RandString())
	c.Fuzz(&m.Value)
}
