#!/bin/sh

PASSWORD=${PASSWORD:-1234567890}

wasmd init --chain-id=testing testing
# staking/governance token is hardcoded in config, change this
sed -i 's/"stake"/"ustake"/' "$HOME"/.wasmd/config/genesis.json
(echo $PASSWORD; echo $PASSWORD) | wasmcli keys add validator
# hardcode the validator account for this instance
echo $PASSWORD | wasmd add-genesis-account validator 1000000000ustake,1000000000ucosm
# (optionally) add a few more genesis accounts
for addr in "$@"; do
  wasmd add-genesis-account $addr 1000000000ustake,1000000000ucosm
done
# submit a genesis validator tx
(echo $PASSWORD; echo $PASSWORD; echo $PASSWORD) | wasmd gentx --name validator --amount 250000000ustake
wasmd collect-gentxs