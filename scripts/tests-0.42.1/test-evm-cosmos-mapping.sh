#!/bin/bash

set -ux

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
NODE_HOME=${NODE_HOME:-"$PWD/.oraid"}
ARGS="--from $USER --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync --home $NODE_HOME"
HIDE_LOGS="/dev/null"

user_address=$(oraid keys show $USER --home $NODE_HOME --keyring-backend test -a)
user_pubkey=$(oraid keys show $USER --home $NODE_HOME --keyring-backend test -p | jq '.key' | tr -d '"')
oraid tx evm set-mapping-evm $user_pubkey $ARGS &>$HIDE_LOGS
# wait for the tx to be completed
sleep 2

expected_evm_address=$(oraid debug pubkey-simple $user_pubkey)
actual_evm_address=$(oraid query evm mappedevm $user_address --output json | jq '.evm_address' | tr -d '"')
if ! [[ $actual_evm_address =~ $expected_evm_address ]]; then
   echo "The evm addresses dont match. EVM cosmos mapping test failed"
   exit 1
fi

# wait for the jsonrpc to start
sleep 14
balance_hex=$(curl --no-progress-meter http://localhost:8545/ -X POST -H "Content-Type: application/json" --data '{"method":"eth_getBalance","params":["'"$actual_evm_address"'", "latest"],"id":1,"jsonrpc":"2.0"}' | jq '.result' | bc)
balance_hex_no_prefix=${balance_hex#0x}
balance_hex_no_prefix_upper=$(echo "$balance_hex_no_prefix" | tr '[:lower:]' '[:upper:]')
balance_decimal=$(echo "ibase=16; ${balance_hex_no_prefix_upper}" | bc)

echo "balance: $balance_decimal"
evm_decimals="10^18"
cosmos_decimals="10^6"
evm_balance=$(echo "scale=10; ($balance_decimal / $evm_decimals) * $cosmos_decimals" | bc)
evm_balance_int=$(echo ${evm_balance%.*})
cosmos_balance=$(oraid query bank balance $user_address orai --output json | jq '.balance.amount | tonumber')
if [[ $evm_balance_int -ne $cosmos_balance ]]; then
   echo "The evm addresses dont match. EVM cosmos mapping test failed"
   exit 1
fi

# test balance change when cosmos address sends some coins to another address
oraid tx bank send $USER orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 1orai $ARGS >$HIDE_LOGS
# need to sleep 2s for tx completed
sleep 2
balance_hex=$(curl --no-progress-meter http://localhost:8545/ -X POST -H "Content-Type: application/json" --data '{"method":"eth_getBalance","params":["'"$actual_evm_address"'", "latest"],"id":1,"jsonrpc":"2.0"}' | jq '.result' | bc)
balance_hex_no_prefix=${balance_hex#0x}
balance_hex_no_prefix_upper=$(echo "$balance_hex_no_prefix" | tr '[:lower:]' '[:upper:]')
balance_decimal_after_change=$(echo "ibase=16; ${balance_hex_no_prefix_upper}" | bc)
if [[ $balance_decimal_after_change -eq $balance_decimal ]]; then
   echo "The evm balance does not get updated after the cosmos address sends coin. EVM cosmos mapping test failed"
   exit 1
fi

# test balance change when evm address sends some coins to another address
cosmos_balance_before_send=$(oraid query bank balance $user_address orai --output json | jq '.balance.amount | tonumber')
private_key=$(oraid keys unsafe-export-cosmos-key $USER --keyring-backend test --home $NODE_HOME)
PRIVATE_KEY_ETH=$private_key sh $PWD/scripts/test-evm-send-token.sh

sleep 2
balance_hex=$(curl --no-progress-meter http://localhost:8545/ -X POST -H "Content-Type: application/json" --data '{"method":"eth_getBalance","params":["'"$actual_evm_address"'", "latest"],"id":1,"jsonrpc":"2.0"}' | jq '.result' | bc)
balance_hex_no_prefix=${balance_hex#0x}
balance_hex_no_prefix_upper=$(echo "$balance_hex_no_prefix" | tr '[:lower:]' '[:upper:]')
balance_decimal_after_send=$(echo "ibase=16; ${balance_hex_no_prefix_upper}" | bc)
if [[ $balance_decimal_after_change -eq $balance_decimal_after_send ]]; then
   echo "The evm balance does not get updated after the evm address sends coin. EVM cosmos mapping test failed"
   exit 1
fi

cosmos_balance_after_send=$(oraid query bank balance $user_address orai --output json | jq '.balance.amount | tonumber')
if [[ $cosmos_balance_before_send -eq $cosmos_balance_after_send ]]; then
   echo "The cosmos balance does not get updated after the evm address sends coin. EVM cosmos mapping test failed"
   exit 1
fi

echo "EVM cosmos mapping tests passed!"
