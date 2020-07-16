#!/bin/sh
#set -o errexit -o nounset -o pipefail

PASSWORD=${PASSWORD:-1234567890}
STAKE=${STAKE_TOKEN:-ustake}
FEE=${FEE_TOKEN:-ucosm}

wasmgovd init --chain-id=testing testing
sed -i 's/permission": "Everybody"/permission": "Nobody"/'  "$HOME"/.wasmgovd/config/genesis.json
# staking/governance token is hardcoded in config, change this
sed -i "s/\"stake\"/\"$STAKE\"/" "$HOME"/.wasmgovd/config/genesis.json
if ! wasmcli keys show validator; then
  (echo "$PASSWORD"; echo "$PASSWORD") | wasmcli keys add validator
fi
# hardcode the validator account for this instance
echo "$PASSWORD" | wasmgovd add-genesis-account validator "1000000000$STAKE,1000000000$FEE"
# (optionally) add a few more genesis accounts
for addr in "$@"; do
  echo $addr
  wasmgovd add-genesis-account "$addr" "1000000000$STAKE,1000000000$FEE"
done
# submit a genesis validator tx
(echo "$PASSWORD"; echo "$PASSWORD"; echo "$PASSWORD") | wasmgovd gentx --name validator --amount "250000000$STAKE"
wasmgovd collect-gentxs
