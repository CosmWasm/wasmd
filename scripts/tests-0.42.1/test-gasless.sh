#!/bin/bash

set -eu

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/swapmap.wasm"}
EXECUTE_MSG=${EXECUTE_MSG:-'{"ping":{}}'}
NODE_HOME=${NODE_HOME:-"$PWD/.oraid"}
ARGS="--from $USER --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync --home $NODE_HOME"
HIDE_LOGS="/dev/null"
PROPOSAL_TEMPLATE_PATH="$PWD/scripts/json/set-gasless-proposal.json"

# prepare a new contract for gasless
store_tx=$(oraid tx wasm store $WASM_PATH $ARGS --output json)
upload_wasm_txhash=$(echo $store_tx | jq -r '.txhash')
# wait for tx included in a block
sleep 2

code_id=$(curl --no-progress-meter http://localhost:1317/cosmos/tx/v1beta1/txs/$upload_wasm_txhash | jq -r '.tx_response.events[] | select (.type == "store_code").attributes[] | select (.key == "code_id").value')
oraid tx wasm instantiate $code_id '{}' --label 'testing' --admin $(oraid keys show $USER --keyring-backend test --home $NODE_HOME -a) $ARGS > $HIDE_LOGS
# wait for tx included in a block
sleep 2
contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts[0]')
echo "contract_address: $contract_address"

# try executing something, gas should equal 0
exec_before=$(oraid tx wasm execute $contract_address $EXECUTE_MSG $ARGS --output json)
exec_before_txhash=$(echo $exec_before | jq -r '.txhash')
# wait for tx included in a block
sleep 2
gas_used_before=$(curl --no-progress-meter http://localhost:1317/cosmos/tx/v1beta1/txs/$exec_before_txhash | jq -r '.tx_response.gas_used | tonumber')
echo "gas used before gasless: $gas_used_before"

PROPOSAL_PATH="$PWD/scripts/json/temp-set-gasless-proposal.json"
echo $(jq --arg address "$contract_address" '.messages[0].contracts = [$address]' $PROPOSAL_TEMPLATE_PATH) > $PROPOSAL_PATH
# set gasless proposal
oraid tx gov submit-proposal $PROPOSAL_PATH $ARGS > $HIDE_LOGS
rm $PROPOSAL_PATH
# wait for tx included in a block
sleep 2
proposal_id=$(oraid query gov proposals --page-reverse --output json | jq '.proposals[0].id | tonumber')
oraid tx gov vote $proposal_id yes $ARGS > $HIDE_LOGS

# wait til proposal passes
sleep 6
proposal_status=$(oraid query gov proposal $proposal_id --output json | jq .proposal.status)
if [ $proposal_status -ne 3 ] ; then
   echo "The proposal has not passed yet"; exit 1
fi

result=$(oraid tx wasm execute $contract_address $EXECUTE_MSG $ARGS --output json)
# wait for tx included in a block
sleep 2
result_txhash=$(echo $result | jq -r '.txhash')
result_tx=$(curl --no-progress-meter http://localhost:1317/cosmos/tx/v1beta1/txs/$result_txhash)
gas_used_after=$(echo $result_tx | jq -r '.tx_response.gas_used | tonumber')
code=$(echo $result_tx | jq '.tx_response.code | tonumber')
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
oraid tx wasm instantiate $code_id '{}' --label 'testing2' --admin $(oraid keys show $USER --keyring-backend test --home $NODE_HOME -a) $ARGS > $HIDE_LOGS
# wait for tx included in a block
sleep 2
non_gasless_contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts[2]')
echo 'non_gasless_contract_address:' $non_gasless_contract_address
result2=$(oraid tx wasm execute $non_gasless_contract_address $EXECUTE_MSG $ARGS --output json)
echo "result2: $result2"
# wait for tx included in a block
sleep 2
result_txhash=$(echo $result2 | jq -r '.txhash')
result_tx=$(curl --no-progress-meter http://localhost:1317/cosmos/tx/v1beta1/txs/$result_txhash)
gas_used_non_gasless=$(echo $result_tx | jq '.tx_response.gas_used | tonumber')
code=$(echo $result_tx | jq '.tx_response.code | tonumber')
echo "gas used of non gasless: $gas_used_non_gasless"
if ! [[ $code == 0 ]] ; then
   echo "Contract gasless execution failed"; exit 1
fi

if [[ $gas_used_non_gasless -le $gas_used_after ]] ; then
   echo "Gas used non gas less is not large enough!"; exit 1
fi

echo "Gasless tests passed!"
