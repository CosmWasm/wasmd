#!/bin/bash

set -eu

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/swapmap.wasm"}
EXECUTE_MSG=${EXECUTE_MSG:-'{"ping":{}}'}
NODE_HOME=${NODE_HOME:-"$PWD/.oraid"}
ARGS="--from $USER --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b block --home $NODE_HOME"
HIDE_LOGS="/dev/null"

# prepare a new contract for gasless
store_ret=$(oraid tx wasm store $WASM_PATH $ARGS --output json)
code_id=$(echo $store_ret | jq -r '.logs[0].events[1].attributes[] | select(.key | contains("code_id")).value')
oraid tx wasm instantiate $code_id '{}' --label 'testing' --admin $(oraid keys show $USER --keyring-backend test --home $NODE_HOME -a) $ARGS > $HIDE_LOGS
contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts[0]')
echo $contract_address

# try executing something, gas should equal 0
gas_used_before=$(oraid tx wasm execute $contract_address $EXECUTE_MSG $ARGS --output json --gas 20000000 | jq '.gas_used | tonumber')
echo "gas used before gasless: $gas_used_before"

# set gasless proposal
oraid tx gov submit-proposal set-gasless $contract_address --title "gasless" --description "gasless" --deposit 10000000orai $ARGS > $HIDE_LOGS
proposal_id=$(oraid query gov proposals --reverse --output json | jq '.proposals[0].proposal_id | tonumber')
oraid tx gov vote $proposal_id yes $ARGS > $HIDE_LOGS

# wait til proposal passes
sleep 6
proposal_status=$(oraid query gov proposal $proposal_id --output json | jq .status)
if ! [[ $proposal_status =~ "PROPOSAL_STATUS_PASSED" ]] ; then
   echo "The proposal has not passed yet"; exit 1
fi

result=$(oraid tx wasm execute $contract_address $EXECUTE_MSG $ARGS --output json)
gas_used_after=$(echo $result | jq '.gas_used | tonumber')
code=$(echo $result | jq '.code | tonumber')
echo "gas used after gasless: $gas_used_after"
if ! [[ $code == 0 ]] ; then
   echo "Contract gasless execution failed"; exit 1
fi

# 1.9 is a magic number chosen to check that if the gas used after gasless has dropped significantly or not
gas_used_compare=$(echo "$gas_used_before / 1.9 / 1" | bc)
echo "gas_used_compare: $gas_used_compare"
if [[ $gas_used_compare -lt $gas_used_after ]] ; then
   echo "Gas used after is not small enough!"; exit 1
fi

# try testing with non-gasless contract with the same logic, should have much higher gas
oraid tx wasm instantiate $code_id '{}' --label 'testing' --admin $(oraid keys show $USER --keyring-backend test --home $NODE_HOME -a) $ARGS > $HIDE_LOGS
non_gasless_contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts[1]')
echo $non_gasless_contract_address
result=$(oraid tx wasm execute $non_gasless_contract_address $EXECUTE_MSG $ARGS --output json)
gas_used_non_gasless=$(echo $result | jq '.gas_used | tonumber')
code=$(echo $result | jq '.code | tonumber')
echo "gas used of non gasless: $gas_used_non_gasless"
if ! [[ $code == 0 ]] ; then
   echo "Contract gasless execution failed"; exit 1
fi

if [[ $gas_used_non_gasless -le $gas_used_after ]] ; then
   echo "Gas used non gas less is not large enough!"; exit 1
fi

echo "Gasless tests passed!"