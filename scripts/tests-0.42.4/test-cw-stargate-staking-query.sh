#!/bin/bash
# Before running this script, you must setup local network:
# sh $PWD/scripts/multinode-local-testnet.sh
# oraiswap-token.wasm source code: https://github.com/oraichain/oraiswap.git

set -eu

WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/cw-stargate-staking-query.wasm"}
ARGS="--chain-id testing -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync"
VALIDATOR1_ARGS=${VALIDATOR1_ARGS:-"--from validator1 --home $NODE_HOME"}

HIDE_LOGS="/dev/null"

store_ret=$(oraid tx wasm store $WASM_PATH $VALIDATOR1_ARGS $ARGS --output json)
store_txhash=$(echo $store_ret | jq -r '.txhash')
# need to sleep 1s for tx already in block
sleep 2
code_id=$(oraid query tx $store_txhash --output json | jq -r '.events[4].attributes[1].value | tonumber')
oraid tx wasm instantiate $code_id '{}' --label 'cw stargate staking query' $VALIDATOR1_ARGS --admin $(oraid keys show validator1 --keyring-backend test --home $HOME/.oraid/validator1 -a) $ARGS > $HIDE_LOGS
# need to sleep 1s for tx already in block
sleep 2
contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts | last')
echo "cw-stargate-staking-query contract address: $contract_address"

# query first validator info
validator_addr=$(oraid query staking validators --output json | jq '.validators[0].operator_address')

# Query validator via cosmwasm contract
QUERY_MSG="{\"staking_validator\":{\"val_addr\":$validator_addr}}"
validator_info=$(oraid query wasm contract-state smart $contract_address $QUERY_MSG --node "tcp://localhost:26657" --output json | jq '.data')

if ! [[ $validator_info == $validator_addr  ]] ; then
   echo "CW Stargate staking query failed"; exit 1
fi

echo "CW Stargate staking query passed"
