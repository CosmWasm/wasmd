#!/bin/bash

set -eu

source "$(dirname $0)/../utils.sh"

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
USER2=${USER2:-''}
WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/swapmap.wasm"}
EXECUTE_MSG=${EXECUTE_MSG:-'{"ping":{}}'}
NODE_HOME=${NODE_HOME:-"$HOME/.oraid"}
VALIDATOR1_HOME=${VALIDATOR1_HOME:-"$NODE_HOME/$USER"}
VALIDATOR2_HOME="$NODE_HOME/$USER2"
VALIDATOR1_ARG="--from $USER --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync --home $VALIDATOR1_HOME"
VALIDATOR2_ARG="--from $USER2 --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync --home $VALIDATOR2_HOME"
HIDE_LOGS="/dev/null"
PROPOSAL_TEMPLATE_PATH="$PWD/scripts/json/set-gasless-proposal.json"

get_tx_response () {
   local command="curl --no-progress-meter http://localhost:1317/cosmos/tx/v1beta1/txs/$1 | jq .tx_response"
   retry_exec "$command" "null" | tr -dc '[:alnum:] \[\]\{\}"_=:,'
}

# prepare a new contract for gasless
upload_wasm_txhash=$(oraid tx wasm store $WASM_PATH $VALIDATOR1_ARG --output json | jq -r '.txhash')
echo "upload_wasm_txhash: $upload_wasm_txhash"
get_tx_response $upload_wasm_txhash > $HIDE_LOGS
code_id=$(oraid q wasm list-code --reverse --limit 1 --output json | jq -r .code_infos[0].code_id)

oraid tx wasm instantiate $code_id '{}' --label 'testing' --admin $(oraid keys show $USER --keyring-backend test --home $VALIDATOR1_HOME -a) $VALIDATOR1_ARG > $HIDE_LOGS
# wait for tx included in a block
command="oraid query wasm list-contract-by-code $code_id --output json | jq '.contracts[0]'"
contract_address=$(retry_exec "$command" "null" | jq -r '.')

# try executing something, gas should equal 0
exec_before_txhash=$(oraid tx wasm execute $contract_address $EXECUTE_MSG $VALIDATOR1_ARG --output json | jq -r '.txhash')
# wait for tx included in a block
echo "exec_before_txhash: $exec_before_txhash"
gas_used_before=$(get_tx_response $exec_before_txhash | jq -r '.gas_used | tonumber')
echo "gas used before gasless: $gas_used_before"

PROPOSAL_PATH="$PWD/scripts/json/temp-set-gasless-proposal.json"
echo $(jq --arg address "$contract_address" '.messages[0].contracts = [$address]' $PROPOSAL_TEMPLATE_PATH) > $PROPOSAL_PATH
# set gasless proposal
create_proposal_txhash=$(oraid tx gov submit-proposal $PROPOSAL_PATH $VALIDATOR1_ARG --output json | jq -r .txhash)
echo "create_proposal_txhash: $create_proposal_txhash"
rm $PROPOSAL_PATH
sleep 2
proposal_id=$(oraid query gov proposals --page-reverse --output json | jq '.proposals[0].id | tonumber')

oraid tx gov vote $proposal_id yes $VALIDATOR1_ARG > $HIDE_LOGS
if [ "$USER2" != '' ]; then
   oraid tx gov vote $proposal_id yes $VALIDATOR2_ARG > $HIDE_LOGS
fi

# wait til proposal passes
command="oraid query gov proposal $proposal_id --output json | jq .proposal.status"
proposal_status=$(retry_exec "$command" "2")
# proposal_status=$(oraid query gov proposal $proposal_id --output json | jq .proposal.status)
if [ $proposal_status -eq "4" ] ; then
   echo "The proposal has failed"; exit 1
fi
if [ $proposal_status -ne "3" ] ; then
   echo "The proposal has not passed yet"; exit 1
fi
result_txhash=$(oraid tx wasm execute $contract_address $EXECUTE_MSG $VALIDATOR1_ARG --output json | jq -r '.txhash')
# wait for tx included in a block
# command="curl --no-progress-meter http://localhost:1317/cosmos/tx/v1beta1/txs/$result_txhash | jq '.tx_response'"
result_tx=$(get_tx_response $result_txhash)
# result_tx=$(curl --no-progress-meter http://localhost:1317/cosmos/tx/v1beta1/txs/$result_txhash)
gas_used_after=$(echo $result_tx | jq -r '.gas_used | tonumber')
code=$(echo $result_tx | jq '.code | tonumber')
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
oraid tx wasm instantiate $code_id '{}' --label 'testing2' --admin $(oraid keys show $USER --keyring-backend test --home $VALIDATOR1_HOME -a) $VALIDATOR1_ARG > $HIDE_LOGS
# wait for tx included in a block
command="oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts[1]'"
non_gasless_contract_address=$(retry_exec "$command" "null")
# non_gasless_contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts[1]')
echo 'non_gasless_contract_address:' $non_gasless_contract_address
result_txhash=$(oraid tx wasm execute $non_gasless_contract_address $EXECUTE_MSG $VALIDATOR1_ARG --output json | jq -r '.txhash')
# wait for tx included in a block
# command="curl --no-progress-meter http://localhost:1317/cosmos/tx/v1beta1/txs/$result_txhash | jq .tx_response"
result_tx=$(get_tx_response $result_txhash)
# result_tx=$(curl --no-progress-meter http://localhost:1317/cosmos/tx/v1beta1/txs/$result_txhash)
gas_used_non_gasless=$(echo $result_tx | jq '.gas_used | tonumber')
code=$(echo $result_tx | jq '.code | tonumber')
echo "gas used of non gasless: $gas_used_non_gasless"
if ! [[ $code == 0 ]] ; then
   echo "Contract gasless execution failed"; exit 1
fi

if [[ $gas_used_non_gasless -le $gas_used_after ]] ; then
   echo "Gas used non gas less is not large enough! Contract gasless test failed"; exit 1
fi

echo "Gasless tests passed!"
