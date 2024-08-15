#!/bin/bash

set -eu

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
NODE_HOME=${NODE_HOME:-"$PWD/.oraid"}
ARGS="--from $USER --chain-id $CHAIN_ID -y --keyring-backend test --gas 20000000 -b block --home $NODE_HOME"
HIDE_LOGS="/dev/null"

# add a signer wallet
signer="signer"
echo "y" | oraid keys add $signer --home $NODE_HOME --keyring-backend test
signer_address=$(oraid keys show $signer --home $NODE_HOME --keyring-backend test -a)

# add a multisig wallet
multisig="multisig"
echo "y" | oraid keys add $multisig --multisig $USER,$signer --multisig-threshold 1 --home $NODE_HOME --keyring-backend test
multisig_address=$(oraid keys show $multisig --home $NODE_HOME --keyring-backend test -a)
user_address=$(oraid keys show $USER --home $NODE_HOME --keyring-backend test -a)

# send some tokens to the multisign address
oraid tx send $user_address $multisig_address 100000000orai $ARGS > $HIDE_LOGS
oraid tx send $user_address $signer_address 100000000orai $ARGS > $HIDE_LOGS

# now we test multi-sign
# generate dry message
oraid tx send $multisig_address $user_address 1orai --generate-only $ARGS 2>&1 | tee tx.json

# sign message
oraid tx sign --from $user_address --multisig=$multisig_address tx.json $ARGS 2>&1 | tee tx-signed-data.json

# multisign
oraid tx multisign tx.json multisig tx-signed-data.json $ARGS 2>&1 | tee tx-signed.json

# broadcast
result=$(oraid tx broadcast tx-signed.json $ARGS --output json | jq '.code | tonumber')

# clean up tx files
rm tx*.json

if ! [[ $result == 0 ]] ; then
   echo "Multi-sig test failed"; exit 1
fi

echo "Multi-sign test passed!"