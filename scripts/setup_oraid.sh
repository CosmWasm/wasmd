#!/bin/bash
set -ux

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
MONIKER=${MONIKER:-node001}
HIDE_LOGS="/dev/null"
# PASSWORD=${PASSWORD:-$1}
rm -rf .oraid/

oraid init --chain-id "$CHAIN_ID" "$MONIKER" >$HIDE_LOGS

oraid keys add $USER --keyring-backend test 2>&1 | tee account.txt
oraid keys add $USER-eth --keyring-backend test --eth 2>&1 | tee account-eth.txt
oraid keys unsafe-export-eth-key $USER-eth --keyring-backend test 2>&1 | tee priv-eth.txt

# hardcode the validator account for this instance
oraid genesis add-genesis-account $USER "100000000000000orai" --keyring-backend test
oraid genesis add-genesis-account $USER-eth "100000000000000orai" --keyring-backend test
oraid genesis add-genesis-account orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 "100000000000000orai" --keyring-backend test

# submit a genesis validator tx
# Workraround for https://github.com/cosmos/cosmos-sdk/issues/8251
oraid genesis gentx $USER "250000000orai" --chain-id="$CHAIN_ID" -y --keyring-backend test

oraid genesis collect-gentxs

oraid start --json-rpc.address="0.0.0.0:8545" --json-rpc.ws-address="0.0.0.0:8546" --json-rpc.api="eth,web3,net,txpool,debug" --json-rpc.enable
