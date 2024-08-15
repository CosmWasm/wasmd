#!/bin/bash
# Before running this script, you must setup local network:
# sh $PWD/scripts/multinode-local-testnet.sh

WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/cw-clock-example.wasm"}
ARGS="--chain-id testing -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b block"
VALIDATOR_HOME=${VALIDATOR_HOME:-"$HOME/.oraid/validator1"}
QUERY_MSG=${QUERY_MSG:-'{"get_config":{}}'}

CONTRACT_GAS_LIMIT=${CONTRACT_GAS_LIMIT:-"123000000"}
TITLE=${TITLE:-"add contract to clock module"}
INITIAL_DEPOSIT=${INITIAL_DEPOSIT:-"10000000orai"}
DESCRIPTION=${DESCRIPTION:-"add cw-clock contract to clock module"}
HIDE_LOGS="/dev/null"

store_ret=$(oraid tx wasm store $WASM_PATH --from validator1 --home $VALIDATOR_HOME $ARGS --output json)
code_id=$(echo $store_ret | jq -r '.logs[0].events[1].attributes[] | select(.key | contains("code_id")).value')
oraid tx wasm instantiate $code_id '{}' --label 'cw clock contract' --from validator1 --home $VALIDATOR_HOME -b block --admin $(oraid keys show validator1 --keyring-backend test --home $VALIDATOR_HOME -a) $ARGS > $HIDE_LOGS
contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts | last')
echo "cw-clock contract address: $contract_address, $CONTRACT_GAS_LIMIT, $TITLE, $INITIAL_DEPOSIT, $DESCRIPTION"

add_contract_result=$(oraid tx clock add-contract $contract_address $CONTRACT_GAS_LIMIT "$TITLE" "$INITIAL_DEPOSIT" "$DESCRIPTION" --from validator1 --home $VALIDATOR_HOME $ARGS --output json)
proposal_id=$(echo $add_contract_result | jq -r '.logs[0].events[4].attributes[] | select(.key | contains("proposal_id")).value')
echo "proposal id: $proposal_id"

oraid tx gov vote $proposal_id yes --from validator1 --home "$HOME/.oraid/validator1" $ARGS > $HIDE_LOGS && oraid tx gov vote $proposal_id yes --from validator2 --home "$HOME/.oraid/validator2" $ARGS > $HIDE_LOGS

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

if [ $counter_after -gt $counter_before ]
then
echo "Clock Counter Test Passed"
else
echo "Clock Counter Test Failed"
fi
