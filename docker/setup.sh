#!/bin/bash
set -o errexit -o nounset -o pipefail

PASSWORD=${PASSWORD:-1234567890}
STAKE=${STAKE_TOKEN:-ustake}
FEE=${FEE_TOKEN:-ucosm}

wasmd init --chain-id=testing testing
# staking/governance token is hardcoded in config, change this
sed -i "s/\"stake\"/\"$STAKE\"/" "$HOME"/.wasmd/config/genesis.json
if ! wasmcli keys show validator; then
  (echo "$PASSWORD"; echo "$PASSWORD") | wasmcli keys add validator
fi
# hardcode the validator account for this instance
echo "$PASSWORD" | wasmd add-genesis-account validator "1000000000$STAKE,1000000000$FEE"
# (optionally) add a few more genesis accounts
for addr in "$@"; do
  echo $addr
  wasmd add-genesis-account "$addr" "1000000000$STAKE,1000000000$FEE"
done
# submit a genesis validator tx
(echo "$PASSWORD"; echo "$PASSWORD"; echo "$PASSWORD") | wasmd gentx --name validator --amount "250000000$STAKE"
wasmd collect-gentxs
