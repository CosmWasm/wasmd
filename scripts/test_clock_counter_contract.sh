#!/bin/bash
# Before running this script, you must setup local network:
# sh $PWD/scripts/multinode-local-testnet.sh
# cw-clock-example.wasm source code: https://github.com/oraichain/cw-plus.git

set -ux

WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/cw-clock-example.wasm"}
ARGS="--chain-id testing -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync"
VALIDATOR1_ARGS=${VALIDATOR1_ARGS:-"--from validator1 --home $HOME/.oraid/validator1"}
VALIDATOR2_ARGS=${VALIDATOR2_ARGS:-"--from validator2 --home $HOME/.oraid/validator2"}
QUERY_MSG=${QUERY_MSG:-'{"get_config":{}}'}

CONTRACT_GAS_LIMIT=${CONTRACT_GAS_LIMIT:-"123000000"}
TITLE=${TITLE:-"add contract to clock module"}
INITIAL_DEPOSIT=${INITIAL_DEPOSIT:-"10000000orai"}
DESCRIPTION=${DESCRIPTION:-"add cw-clock contract to clock module"}
HIDE_LOGS="/dev/null"
CLOCK_PROPOSAL_FILE=${CLOCK_PROPOSAL_FILE:-"$PWD/scripts/json/clock-proposal.json"}

store_ret=$(oraid tx wasm store $WASM_PATH $VALIDATOR1_ARGS $ARGS --output json)
store_txhash=$(echo $store_ret | jq -r '.txhash')
# need to sleep 1s for tx already in block
sleep 2
# need to use temp.json since there's a weird error: jq: parse error: Invalid string: control characters from U+0000 through U+001F must be escaped at line 1, column 72291
# probably because of weird characters from the raw code bytes
oraid query tx $store_txhash --output json > temp.json
code_id=$(cat temp.json | jq -r '.events[4].attributes[] | select(.key | contains("code_id")).value')
rm temp.json
oraid tx wasm instantiate $code_id '{}' --label 'cw clock contract' $VALIDATOR1_ARGS --admin $(oraid keys show validator1 --keyring-backend test --home $HOME/.oraid/validator1 -a) $ARGS > $HIDE_LOGS
# need to sleep 1s for tx already in block
sleep 1
contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts | last')
echo "cw-clock contract address: $contract_address, $CONTRACT_GAS_LIMIT, $TITLE, $INITIAL_DEPOSIT, $DESCRIPTION"

# create clock proposal to whitelist contract
update_proposal() {
    cat $CLOCK_PROPOSAL_FILE | jq "$1" >$PWD/scripts/json/temp_proposal.json && mv $PWD/scripts/json/temp_proposal.json $CLOCK_PROPOSAL_FILE
}

# update authority proposal.json
MODULE_ACCOUNT=$(oraid query auth module-account gov --output json | jq '.account.value.address')
update_proposal ".messages[0][\"authority\"]=$MODULE_ACCOUNT"
# update clock contract in the proposal
update_proposal ".messages[0].params.contract_addresses=[\"$contract_address\""]

store_ret=$(oraid tx gov submit-proposal $CLOCK_PROPOSAL_FILE $VALIDATOR1_ARGS $ARGS --output json)
store_txhash=$(echo $store_ret | jq -r '.txhash')
# sleep 2s before vote to wait for tx confirm
sleep 2
proposal_id=$(oraid query tx $store_txhash --output json | jq -r '.events[4].attributes[] | select(.key | contains("proposal_id")).value')

oraid tx gov vote $proposal_id yes $VALIDATOR1_ARGS $ARGS > $HIDE_LOGS && oraid tx gov vote $proposal_id yes $VALIDATOR2_ARGS $ARGS > $HIDE_LOGS

# sleep to wait til the proposal passes
echo "Sleep til the proposal passes..."
sleep 5

# Query the counter
counter_before=$(oraid query wasm contract-state smart $contract_address $QUERY_MSG --node "tcp://localhost:26657" --output json | jq -r '.data.val | tonumber')
sleep 2
echo "cw-clock counter_before: $counter_before"

counter_after=$(oraid query wasm contract-state smart $contract_address $QUERY_MSG --node "tcp://localhost:26657" --output json | jq -r '.data.val | tonumber')
sleep 2
echo "cw-clock counter_after: $counter_after"

if [[ $counter_after == $counter_before ]]; then
  echo "Clock Counter Test Failed"; exit 1
fi

QUERY_MSG='{"get_after_sudo":{}}'
after_sudo_before=$(oraid query wasm contract-state smart $contract_address $QUERY_MSG --node "tcp://localhost:26657" --output json | jq -r '.data | tonumber')
sleep 2
echo "cw-clock after sudo before: $after_sudo_before"

after_sudo_after=$(oraid query wasm contract-state smart $contract_address $QUERY_MSG --node "tcp://localhost:26657" --output json | jq -r '.data | tonumber')
sleep 2
echo "cw-clock after sudo after: $after_sudo_after"

if [[ $after_sudo_before == $after_sudo_after ]]; then
  echo "Clock Counter After Sudo Test Failed"; exit 1
fi

echo "Clock Counter Test Passed"
