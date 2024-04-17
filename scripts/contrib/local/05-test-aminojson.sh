#!/bin/bash
set -o errexit -o nounset -o pipefail -x

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

echo "-----------------------"
echo "## Add new CosmWasm contract"
RESP=$(wasmd tx wasm store "$DIR/../../../x/wasm/keeper/testdata/hackatom.wasm" \
  --from validator --gas 1500000 -y --chain-id=testing --node=http://localhost:26657 -b sync -o json --keyring-backend=test)
sleep 6
RESP=$(wasmd q tx $(echo "$RESP"| jq -r '.txhash') -o json)
CODE_ID=$(echo "$RESP" | jq -r '.events[]| select(.type=="store_code").attributes[]| select(.key=="code_id").value')
CODE_HASH=$(echo "$RESP" | jq -r '.events[]| select(.type=="store_code").attributes[]| select(.key=="code_checksum").value')
echo "* Code id: $CODE_ID"
echo "* Code checksum: $CODE_HASH"

echo "-----------------------"
echo "## Create new contract instance"
INIT="{\"verifier\":\"$(wasmd keys show validator -a --keyring-backend=test)\", \"beneficiary\":\"$(wasmd keys show fred -a --keyring-backend=test)\"}"

wasmd tx wasm instantiate "$CODE_ID" "$INIT" --admin="$(wasmd keys show validator -a --keyring-backend=test)" \
  --from validator --amount="100ustake" --label "local0.1.0" \
  --gas 1000000 -y --chain-id=testing -b sync -o json --keyring-backend=test --sign-mode amino-json