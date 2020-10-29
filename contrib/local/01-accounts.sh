#!/bin/bash
set -o errexit -o nounset -o pipefail

BASE_ACCOUNT=$(wasmcli keys show validator -a)
wasmcli q account "$BASE_ACCOUNT" -o json | jq

echo "## Add new account"
wasmcli keys add fred

echo "## Check balance"
NEW_ACCOUNT=$(wasmcli keys show fred -a)
wasmcli q bank balances "$NEW_ACCOUNT" -o json || true

echo "## Transfer tokens"
wasmcli tx send validator "$NEW_ACCOUNT" 1ustake --gas 1000000  -y --chain-id=testing --node=http://localhost:26657 -b block | jq

echo "## Check balance again"
wasmcli q bank balances "$NEW_ACCOUNT" -o json | jq
