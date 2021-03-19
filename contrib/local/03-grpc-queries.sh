#!/bin/bash
set -o errexit -o nounset -o pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

echo "-----------------------"
COSMOS_SDK_DIR=${COSMOS_SDK_DIR:-$(go list -f "{{ .Dir }}" -m github.com/cosmos/cosmos-sdk)}

echo "### List all codes"
RESP=$(grpcurl -plaintext -import-path $COSMOS_SDK_DIR/third_party/proto -import-path $COSMOS_SDK_DIR/proto -import-path . -proto ./x/wasm/types/query.proto \
  localhost:9090  cosmwasm.wasm.v1beta1.Query/Codes)
echo "$RESP" | jq

CODE_ID=$(echo "$RESP" | jq -r '.codeInfos[-1].codeId')
echo "### List contract by code"
RESP=$(grpcurl -plaintext -import-path $COSMOS_SDK_DIR/third_party/proto -import-path $COSMOS_SDK_DIR/proto -import-path . -proto ./x/wasm/types/query.proto \
  -d "{\"codeId\": $CODE_ID}" localhost:9090  cosmwasm.wasm.v1beta1.Query/ContractsByCode )
echo $RESP | jq

echo "### Show history for contract"
CONTRACT=$(echo $RESP | jq -r ".contractInfos[-1].address")
grpcurl -plaintext -import-path $COSMOS_SDK_DIR/third_party/proto -import-path $COSMOS_SDK_DIR/proto -import-path . -proto ./x/wasm/types/query.proto \
  -d "{\"address\": \"$CONTRACT\"}" localhost:9090  cosmwasm.wasm.v1beta1.Query/ContractHistory | jq

echo "### Show contract state"
grpcurl -plaintext -import-path $COSMOS_SDK_DIR/third_party/proto -import-path $COSMOS_SDK_DIR/proto -import-path . -proto ./x/wasm/types/query.proto \
  -d "{\"address\": \"$CONTRACT\"}" localhost:9090  cosmwasm.wasm.v1beta1.Query/AllContractState | jq

echo "Empty state due to 'burner' contract cleanup"
