#!/bin/bash

set -eu

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
NODE_HOME=${NODE_HOME:-"$PWD/.oraid"}
ARGS="--from $USER --chain-id $CHAIN_ID -y --keyring-backend test --gas 20000000 -b block --home $NODE_HOME"
docker_command="docker-compose -f $PWD/docker-compose-e2e-upgrade.yml exec"
validator1_command="$docker_command validator1 bash -c"
HIDE_LOGS="/dev/null"

# add a signer wallet
signer="signer"
$validator1_command "echo y | oraid keys add $signer --home $NODE_HOME --keyring-backend test"
signer_address_result=`$validator1_command "oraid keys show $signer --home $NODE_HOME --keyring-backend test --output json"`
signer_address=$(echo $signer_address_result | jq '.address' | tr -d '"')

# add a multisig wallet
multisig="multisig"
$validator1_command "echo y | oraid keys add $multisig --multisig $USER,$signer --multisig-threshold 1 --home $NODE_HOME --keyring-backend test"
multisig_address_result=`$validator1_command "oraid keys show $multisig --home $NODE_HOME --keyring-backend test --output json"`
multisig_address=$(echo $multisig_address_result | jq '.address' | tr -d '"')

user_address_result=`$validator1_command "oraid keys show $USER --home $NODE_HOME --keyring-backend test --output json"`
user_address=$(echo $user_address_result | jq '.address' | tr -d '"')

# send some tokens to the multisign address
$validator1_command "oraid tx send $user_address $multisig_address 100000000orai $ARGS > $HIDE_LOGS"
$validator1_command "oraid tx send $user_address $signer_address 100000000orai $ARGS > $HIDE_LOGS"

# now we test multi-sign
# generate dry message
$validator1_command "oraid tx send $multisig_address $user_address 1orai --generate-only $ARGS 2>&1 | tee tx.json"

# sign message
$validator1_command "oraid tx sign --from $user_address --multisig=$multisig_address tx.json $ARGS 2>&1 | tee tx-signed-data.json"

# multisign
$validator1_command "oraid tx multisign tx.json multisig tx-signed-data.json $ARGS 2>&1 | tee tx-signed.json"

# broadcast
result=`$validator1_command "oraid tx broadcast tx-signed.json $ARGS --output json"`
code=$(echo $result | jq '.code | tonumber')
# clean up tx files
$validator1_command "rm tx*.json"

if [[ $code -gt 0 ]] ; then
   echo "Multi-sig test failed"; exit 1
fi

echo "Multi-sign test passed!"