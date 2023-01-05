#!/bin/bash
set -o errexit -o nounset -o pipefail

contract_path=$1
label=$2
init=${3:-{\}}
code_id=${4:-}
CHAIN_ID=${CHAIN_ID:-testing}
ADMIN=${ADMIN:-validator}

echo "Enter passphrase:"
read -s passphrase

if [ -z $code_id ]
then 
    store_ret=$(echo $passphrase | wasmd tx wasm store $contract_path --from $ADMIN --gas="auto" --gas-adjustment 1.2 --chain-id=$CHAIN_ID -o json -b block -y)
    echo $store_ret
    if [ ! `command -v jq` ]; then  
        echo "Installing jq ..."
        [ `uname -s | grep Darwin` ] && brew install jq || apk add jq    
    fi  
    code_id=$(echo $store_ret | jq -r '.logs[0].events[-1].attributes[0].value')
fi 

# echo "wasmd tx wasm instantiate $code_id '$init' --from $ADMIN --label '$label' --gas auto --gas-adjustment 1.2 --chain-id=$CHAIN_ID -y"
# quote string with "" with escape content inside which contains " characters

admin=$(echo $passphrase | wasmd keys show $ADMIN --output json | jq -r '.address')

(echo $passphrase;echo $passphrase) | wasmd tx wasm instantiate $code_id "$init" --from $ADMIN --label "$label" --gas auto --gas-adjustment 1.2 --admin $admin --chain-id=$CHAIN_ID -b block -y -o json
contract_address=$(wasmd query wasm list-contract-by-code $code_id --output json | jq -r '.contracts[-1]')

echo "contract address: $contract_address"