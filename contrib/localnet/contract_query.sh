#!/bin/bash

source ./localnet_vars.sh

[ -z "$CODE_ID" ] && echo "Please set CODE_ID" && exit 1

# Check the contract state (and account balance)
wasmd query wasm list-contract-by-code $CODE_ID $NODE
CONTRACT_ADDR=$(wasmd query wasm list-contract-by-code $CODE_ID $NODE |  grep address | tail -1 | cut -d\  -f3)

# We should see this contract with 1000ucosm
wasmd query wasm contract $CONTRACT_ADDR $NODE
wasmd query account $CONTRACT_ADDR $NODE

# You can dump entire contract state
wasmd query wasm contract-state all $CONTRACT_ADDR $NODE

# Query 'somedata/datakey<n>' directly
PREFIX="somedata"
PREFIX_KEY=$(echo -n $PREFIX | xxd -ps)
PREFIX_KEY=$(echo -n $PREFIX | wc -c | xargs printf "%04x$PREFIX_KEY")

for I in $(seq 0 1000)
do
  KEY="datakey$I"
  KEY_KEY=$(echo -n $KEY | xxd -ps)
  echo "Querying '$PREFIX/$KEY' directly:"
  wasmd query wasm contract-state raw $CONTRACT_ADDR ${PREFIX_KEY}${KEY_KEY} $NODE --hex
done

# Note that keys are hex encoded, and val is base64 encoded.
# To view the returned data (assuming it is ASCII), try something like:
# (Note that in many cases the binary data returned is non in ascii format, thus the encoding)
#wasmd query wasm contract-state all $CONTRACT_ADDR $NODE --output "json" | jq -r '.models[0].key' | xxd -r -ps
# FIXME:
#wasmd query wasm contract-state all $CONTRACT_ADDR $NODE
#wasmd query wasm contract-state all $CONTRACT_ADDR $NODE --output "json" | jq -r '.models[0].value' | base64 -d
# FIXME:
#wasmd query wasm contract-state all $CONTRACT_ADDR $NODE

# Or try a "smart query", executing against the contract
wasmd query wasm contract-state smart $CONTRACT_ADDR '{"grab_data":{"start_after":null}}' $NODE
