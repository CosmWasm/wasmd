#!/bin/bash

set -eu

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
NODE_HOME=${NODE_HOME:-"$PWD/.oraid"}
ARGS="--from $USER --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b block --home $NODE_HOME"
HIDE_LOGS="/dev/null"

user_address=$(oraid keys show $USER --home $NODE_HOME --keyring-backend test -a)
user_pubkey=$(oraid keys show $USER --home $NODE_HOME --keyring-backend test -p | jq '.key' | tr -d '"')
oraid tx evm set-mapping-evm $user_pubkey $ARGS > $HIDE_LOGS

expected_evm_address=$(oraid debug pubkey-simple $user_pubkey)
actual_evm_address=$(oraid query evm mappedevm $user_address --output json | jq '.evm_address' | tr -d '"')
if ! [[ $actual_evm_address =~ $expected_evm_address ]] ; then
   echo "The evm addresses dont match"; exit 1
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
cosmos_balance=$(oraid query bank balances $user_address --denom orai --output json | jq '.amount | tonumber')
if [[ $evm_balance_int -ne $cosmos_balance ]] ; then
   echo "The evm addresses dont match"; exit 1
fi

# test balance change when cosmos address sends some coins to another address
oraid tx send $USER orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 1orai $ARGS > $HIDE_LOGS
# evm balance should be 0
balance_hex=$(curl --no-progress-meter http://localhost:8545/ -X POST -H "Content-Type: application/json" --data '{"method":"eth_getBalance","params":["'"$actual_evm_address"'", "latest"],"id":1,"jsonrpc":"2.0"}' | jq '.result' | bc)
balance_hex_no_prefix=${balance_hex#0x}
balance_hex_no_prefix_upper=$(echo "$balance_hex_no_prefix" | tr '[:lower:]' '[:upper:]')
balance_decimal_after_change=$(echo "ibase=16; ${balance_hex_no_prefix_upper}" | bc)
if [[ $balance_decimal_after_change -eq $balance_decimal ]] ; then
   echo "The evm balance does not get updated after the cosmos address sends coin."; exit 1
fi

echo "EVM cosmos mapping tests passed!"