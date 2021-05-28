#!/bin/bash

source ./localnet_vars.sh

[ -z "$CONTRACT" ] && echo "Please set CONTRACT with the path to the contract wasm file" && exit 1

RES=$(wasmd tx wasm store $CONTRACT --from $USER $TXFLAG -y $KEYRING)
export CODE_ID=$(echo $RES | jq -r '.logs[0].events[0].attributes[-1].value')

# List current contracts
wasmd query wasm list-code $NODE

echo
echo "Contract id: $CODE_ID"
echo "export CODE_ID=$CODE_ID"
