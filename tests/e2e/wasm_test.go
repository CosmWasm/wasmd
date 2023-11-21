package e2e

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/CosmWasm/wasmd/x/wasm/ibctesting"
)

func TestNFTSubmessages(t *testing.T) {
	// Given a cw721_base contract and a compatible contract as receiver
	// and a nft minted
	// when a send_nft is executed on the base contract, a submessage is emitted
	// and handled by the receiver contract
	specs := map[string]struct {
		submsgPayload string
		expErr        bool
	}{
		"succeed": {
			submsgPayload: `"succeed"`,
		},
		"fail": {
			submsgPayload: `"fail"`,
			expErr:        true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			coord := ibctesting.NewCoordinator(t, 1)
			chain := coord.GetChain(ibctesting.GetChainID(1))
			minterAddress := chain.SenderAccount.GetAddress()

			codeID := chain.StoreCodeFile("testdata/cw721_base.wasm.gz").CodeID
			senderContractAddr := chain.InstantiateContract(codeID, []byte(fmt.Sprintf(`{"name":"Reece #00001", "symbol":"juno-reece-test-#00001", "minter":"%s"}`, minterAddress.String())))

			codeID = chain.StoreCodeFile("testdata/cw721_receiver.wasm.gz").CodeID
			receiverContractAddr := chain.InstantiateContract(codeID, []byte(`{}`))

			// and token minted
			execMsg := types.MsgExecuteContract{
				Sender:   minterAddress.String(),
				Contract: senderContractAddr.String(),
				Msg:      []byte(fmt.Sprintf(`{"mint":{"token_id":"00000", "owner":"%s"}}`, minterAddress.String())),
			}
			_, err := chain.SendMsgs(&execMsg)
			require.NoError(t, err)

			// when submessages is emitted
			execMsg = types.MsgExecuteContract{
				Sender:   minterAddress.String(),
				Contract: senderContractAddr.String(),
				Msg:      []byte(fmt.Sprintf(`{"send_nft": { "contract": "%s", "token_id": "00000", "msg": "%s" }}`, receiverContractAddr.String(), base64.RawStdEncoding.EncodeToString([]byte(spec.submsgPayload)))),
			}
			_, gotErr := chain.SendMsgs(&execMsg)
			// then
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
