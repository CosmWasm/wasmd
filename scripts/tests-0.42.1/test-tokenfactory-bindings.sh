#!/bin/bash

set -ux

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
NODE_HOME=${NODE_HOME:-"$PWD/.oraid"}
WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/tokenfactory.wasm"}
ARGS="--from $USER --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync --home $NODE_HOME"
user_address=$(oraid keys show $USER --keyring-backend test --home $NODE_HOME -a)
HIDE_LOGS="/dev/null"

# deploy cw-bindings contract
store_txhash=$(oraid tx wasm store $WASM_PATH $ARGS --output json | jq -r '.txhash')
# need to sleep 1s
sleep 1
code_id=$(oraid query tx $store_txhash --output json | jq -r '.events[4].attributes[] | select(.key | contains("code_id")).value')
oraid tx wasm instantiate $code_id '{}' --label 'tokenfactory cw bindings testing' --admin $user_address $ARGS >$HIDE_LOGS
sleep 1
contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts[0]')
echo $contract_address

subdenom="usdc"
CREATE_DENOM_MSG='{"create_denom":{"subdenom":"'"$subdenom"'"}}'
QUERY_DENOM_MSG='{"get_denom":{"creator_address":"'"$user_address"'","subdenom":"'"$subdenom"'"}}'

echo "create denom msg: $CREATE_DENOM_MSG"
echo "query denom msg: $QUERY_DENOM_MSG"

# send to the contract some funds to create denom
oraid tx bank send $user_address $contract_address 100000000orai $ARGS > $HIDE_LOGS

# create denom
# sleep 1s to not miss match account sequence
sleep 1
oraid tx wasm execute $contract_address $CREATE_DENOM_MSG $ARGS > $HIDE_LOGS

# query created denom
# sleep 1s for create denom tx already in block
sleep 1
created_denom=$(oraid query wasm contract-state smart $contract_address $QUERY_DENOM_MSG --output json | jq '.data.denom' | tr -d '"')

if ! [[ $created_denom =~ "factory/$user_address/$subdenom" ]] ; then
   echo "The created denom does not match with our expected denom"; exit 1
fi

echo "Tokenfactory cw binding tests passed!"
