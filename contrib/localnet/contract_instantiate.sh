#!/bin/bash

source ./localnet_vars.sh

[ -z "$CODE_ID" ] && echo "Please set contract CODE_ID" && exit 1

echo "Contract code id: $CODE_ID"

# Instantiate contract
# Depending on your contract, you may need to modify this
INIT=$(jq -n '{}')
wasmd tx wasm instantiate $CODE_ID "$INIT" --from $USER --amount=1000ucosm --label "experiment 1" $TXFLAG -y $KEYRING

# Check the contract state (and account balance)
wasmd query wasm list-contract-by-code $CODE_ID $NODE
CONTRACT_ADDR=$(wasmd query wasm list-contract-by-code $CODE_ID $NODE |  grep address | tail -1 | cut -d\  -f3)

echo
echo "Contract address: $CONTRACT_ADDR"
