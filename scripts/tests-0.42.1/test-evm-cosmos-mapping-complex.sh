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

private_key=$(oraid keys unsafe-export-cosmos-key $USER --keyring-backend test --home $NODE_HOME)
user_address=$(oraid keys show $USER --home $NODE_HOME --keyring-backend test -a)

current_cosmos_sequence_before=$(oraid query auth account $user_address --output json | jq '.sequence | tonumber')
PRIVATE_KEY_ETH=$private_key sh $PWD/scripts/test-erc20-deploy.sh
current_cosmos_sequence_after=$(oraid query auth account $user_address --output json | jq '.sequence | tonumber')
expected_cosmos_sequence=$((current_cosmos_sequence_before + 1))

if [[ $current_cosmos_sequence_after -ne $expected_cosmos_sequence ]] ; then
   echo "Cosmos sequences don't match"; exit 1
fi

echo "EVM cosmos mapping complex tests passed!"