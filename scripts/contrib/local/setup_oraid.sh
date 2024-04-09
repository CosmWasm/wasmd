#!/bin/bash
set -o errexit -o nounset -o pipefail

PASSWORD=${PASSWORD:-1234567890}
STAKE=${STAKE_TOKEN:-ustake}
FEE=${FEE_TOKEN:-ucosm}
CHAIN_ID=${CHAIN_ID:-testing}
MONIKER=${MONIKER:-node001}

oraid init --chain-id "$CHAIN_ID" "$MONIKER"
# staking/governance token is hardcoded in config, change this
## OSX requires: -i.
sed -i. "s/\"stake\"/\"$STAKE\"/" "$HOME"/.oraid/config/genesis.json
if ! oraid keys show validator --keyring-backend=test; then
  (
    echo "$PASSWORD"
    echo "$PASSWORD"
  ) | oraid keys add validator --keyring-backend=test
fi
# hardcode the validator account for this instance
echo "$PASSWORD" | oraid genesis add-genesis-account validator "1000000000000$STAKE,1000000000000$FEE" --keyring-backend=test
# (optionally) add a few more genesis accounts
for addr in "$@"; do
  echo "$addr"
  oraid genesis add-genesis-account "$addr" "1000000000$STAKE,1000000000$FEE" --keyring-backend=test
done
# submit a genesis validator tx
## Workraround for https://github.com/cosmos/cosmos-sdk/issues/8251
(
  echo "$PASSWORD"
  echo "$PASSWORD"
  echo "$PASSWORD"
) | oraid genesis gentx validator "250000000$STAKE" --chain-id="$CHAIN_ID" --amount="250000000$STAKE" --keyring-backend=test
## should be:
# (echo "$PASSWORD"; echo "$PASSWORD"; echo "$PASSWORD") | oraid gentx validator "250000000$STAKE" --chain-id="$CHAIN_ID"
oraid genesis collect-gentxs
