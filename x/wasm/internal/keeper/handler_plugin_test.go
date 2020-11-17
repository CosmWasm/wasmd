package keeper

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncoding(t *testing.T) {
	_, _, addr1 := keyPubAddr()
	_, _, addr2 := keyPubAddr()
	invalidAddr := "xrnd1d02kd90n38qvr3qb9qof83fn2d2"
	valAddr := make(sdk.ValAddress, sdk.AddrLen)
	valAddr[0] = 12
	valAddr2 := make(sdk.ValAddress, sdk.AddrLen)
	valAddr2[1] = 123

	jsonMsg := json.RawMessage(`{"foo": 123}`)

	cases := map[string]struct {
		sender sdk.AccAddress
		input  wasmvmtypes.CosmosMsg
		// set if valid
		output []sdk.Msg
		// set if invalid
		isError bool
	}{
		"simple send": {
			sender: addr1,
			input: wasmvmtypes.CosmosMsg{
				Bank: &wasmvmtypes.BankMsg{
					Send: &wasmvmtypes.SendMsg{
						FromAddress: addr1.String(),
						ToAddress:   addr2.String(),
						Amount: []wasmvmtypes.Coin{
							{
								Denom:  "uatom",
								Amount: "12345",
							},
							{
								Denom:  "usdt",
								Amount: "54321",
							},
						},
					},
				},
			},
			output: []sdk.Msg{
				&banktypes.MsgSend{
					FromAddress: addr1.String(),
					ToAddress:   addr2.String(),
					Amount: sdk.Coins{
						sdk.NewInt64Coin("uatom", 12345),
						sdk.NewInt64Coin("usdt", 54321),
					},
				},
			},
		},
		"invalid send amount": {
			sender: addr1,
			input: wasmvmtypes.CosmosMsg{
				Bank: &wasmvmtypes.BankMsg{
					Send: &wasmvmtypes.SendMsg{
						FromAddress: addr1.String(),
						ToAddress:   addr2.String(),
						Amount: []wasmvmtypes.Coin{
							{
								Denom:  "uatom",
								Amount: "123.456",
							},
						},
					},
				},
			},
			isError: true,
		},
		"invalid address": {
			sender: addr1,
			input: wasmvmtypes.CosmosMsg{
				Bank: &wasmvmtypes.BankMsg{
					Send: &wasmvmtypes.SendMsg{
						FromAddress: addr1.String(),
						ToAddress:   invalidAddr,
						Amount: []wasmvmtypes.Coin{
							{
								Denom:  "uatom",
								Amount: "7890",
							},
						},
					},
				},
			},
			isError: false, // addresses are checked in the handler
			output: []sdk.Msg{
				&banktypes.MsgSend{
					FromAddress: addr1.String(),
					ToAddress:   invalidAddr,
					Amount: sdk.Coins{
						sdk.NewInt64Coin("uatom", 7890),
					},
				},
			},
		},
		"wasm execute": {
			sender: addr1,
			input: wasmvmtypes.CosmosMsg{
				Wasm: &wasmvmtypes.WasmMsg{
					Execute: &wasmvmtypes.ExecuteMsg{
						ContractAddr: addr2.String(),
						Msg:          jsonMsg,
						Send: []wasmvmtypes.Coin{
							wasmvmtypes.NewCoin(12, "eth"),
						},
					},
				},
			},
			output: []sdk.Msg{
				&types.MsgExecuteContract{
					Sender:    addr1.String(),
					Contract:  addr2.String(),
					Msg:       jsonMsg,
					SentFunds: sdk.NewCoins(sdk.NewInt64Coin("eth", 12)),
				},
			},
		},
		"wasm instantiate": {
			sender: addr1,
			input: wasmvmtypes.CosmosMsg{
				Wasm: &wasmvmtypes.WasmMsg{
					Instantiate: &wasmvmtypes.InstantiateMsg{
						CodeID: 7,
						Msg:    jsonMsg,
						Send: []wasmvmtypes.Coin{
							wasmvmtypes.NewCoin(123, "eth"),
						},
					},
				},
			},
			output: []sdk.Msg{
				&types.MsgInstantiateContract{
					Sender: addr1.String(),
					CodeID: 7,
					// TODO: fix this
					Label:     fmt.Sprintf("Auto-created by %s", addr1),
					InitMsg:   jsonMsg,
					InitFunds: sdk.NewCoins(sdk.NewInt64Coin("eth", 123)),
				},
			},
		},
		"staking delegate": {
			sender: addr1,
			input: wasmvmtypes.CosmosMsg{
				Staking: &wasmvmtypes.StakingMsg{
					Delegate: &wasmvmtypes.DelegateMsg{
						Validator: valAddr.String(),
						Amount:    wasmvmtypes.NewCoin(777, "stake"),
					},
				},
			},
			output: []sdk.Msg{
				&stakingtypes.MsgDelegate{
					DelegatorAddress: addr1.String(),
					ValidatorAddress: valAddr.String(),
					Amount:           sdk.NewInt64Coin("stake", 777),
				},
			},
		},
		"staking delegate to non-validator": {
			sender: addr1,
			input: wasmvmtypes.CosmosMsg{
				Staking: &wasmvmtypes.StakingMsg{
					Delegate: &wasmvmtypes.DelegateMsg{
						Validator: addr2.String(),
						Amount:    wasmvmtypes.NewCoin(777, "stake"),
					},
				},
			},
			isError: false, // fails in the handler
			output: []sdk.Msg{
				&stakingtypes.MsgDelegate{
					DelegatorAddress: addr1.String(),
					ValidatorAddress: addr2.String(),
					Amount:           sdk.NewInt64Coin("stake", 777),
				},
			},
		},
		"staking undelegate": {
			sender: addr1,
			input: wasmvmtypes.CosmosMsg{
				Staking: &wasmvmtypes.StakingMsg{
					Undelegate: &wasmvmtypes.UndelegateMsg{
						Validator: valAddr.String(),
						Amount:    wasmvmtypes.NewCoin(555, "stake"),
					},
				},
			},
			output: []sdk.Msg{
				&stakingtypes.MsgUndelegate{
					DelegatorAddress: addr1.String(),
					ValidatorAddress: valAddr.String(),
					Amount:           sdk.NewInt64Coin("stake", 555),
				},
			},
		},
		"staking redelegate": {
			sender: addr1,
			input: wasmvmtypes.CosmosMsg{
				Staking: &wasmvmtypes.StakingMsg{
					Redelegate: &wasmvmtypes.RedelegateMsg{
						SrcValidator: valAddr.String(),
						DstValidator: valAddr2.String(),
						Amount:       wasmvmtypes.NewCoin(222, "stake"),
					},
				},
			},
			output: []sdk.Msg{
				&stakingtypes.MsgBeginRedelegate{
					DelegatorAddress:    addr1.String(),
					ValidatorSrcAddress: valAddr.String(),
					ValidatorDstAddress: valAddr2.String(),
					Amount:              sdk.NewInt64Coin("stake", 222),
				},
			},
		},
		"staking withdraw (implicit recipient)": {
			sender: addr1,
			input: wasmvmtypes.CosmosMsg{
				Staking: &wasmvmtypes.StakingMsg{
					Withdraw: &wasmvmtypes.WithdrawMsg{
						Validator: valAddr2.String(),
					},
				},
			},
			output: []sdk.Msg{
				&distributiontypes.MsgSetWithdrawAddress{
					DelegatorAddress: addr1.String(),
					WithdrawAddress:  addr1.String(),
				},
				&distributiontypes.MsgWithdrawDelegatorReward{
					DelegatorAddress: addr1.String(),
					ValidatorAddress: valAddr2.String(),
				},
			},
		},
		"staking withdraw (explicit recipient)": {
			sender: addr1,
			input: wasmvmtypes.CosmosMsg{
				Staking: &wasmvmtypes.StakingMsg{
					Withdraw: &wasmvmtypes.WithdrawMsg{
						Validator: valAddr2.String(),
						Recipient: addr2.String(),
					},
				},
			},
			output: []sdk.Msg{
				&distributiontypes.MsgSetWithdrawAddress{
					DelegatorAddress: addr1.String(),
					WithdrawAddress:  addr2.String(),
				},
				&distributiontypes.MsgWithdrawDelegatorReward{
					DelegatorAddress: addr1.String(),
					ValidatorAddress: valAddr2.String(),
				},
			},
		},
	}

	encoder := DefaultEncoders()
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			res, err := encoder.Encode(tc.sender, tc.input)
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.output, res)
			}
		})
	}

}
