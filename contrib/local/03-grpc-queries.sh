#!/bin/bash
set -o errexit -o nounset -o pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

echo "-----------------------"
COSMOS_SDK_DIR=${COSMOS_SDK_DIR:-$(go list -f "{{ .Dir }}" -m github.com/cosmos/cosmos-sdk)}

echo "### List all codes"
grpcurl -plaintext -import-path $COSMOS_SDK_DIR/third_party/proto -import-path $COSMOS_SDK_DIR/proto -import-path . -proto ./x/wasm/internal/types/query.proto \
  localhost:9090  wasmd.x.wasmd.v1beta1.Query/Codes | jq

echo "### List contract by code"
RESP=$(grpcurl -plaintext -import-path $COSMOS_SDK_DIR/third_party/proto -import-path $COSMOS_SDK_DIR/proto -import-path . -proto ./x/wasm/internal/types/query.proto \
  -d '{"codeId":2}' localhost:9090  wasmd.x.wasmd.v1beta1.Query/ContractsByCode )
echo $RESP | jq

echo "### Show history for contract"
CONTRACT=$(echo $RESP | jq -r ".contractInfos[-1].address")
grpcurl -plaintext -import-path $COSMOS_SDK_DIR/third_party/proto -import-path $COSMOS_SDK_DIR/proto -import-path . -proto ./x/wasm/internal/types/query.proto \
  -d "{\"address\": \"$CONTRACT\"}" localhost:9090  wasmd.x.wasmd.v1beta1.Query/ContractHistory | jq

echo "### Show contract state"
CONTRACT=$(echo $RESP | jq -r ".contractInfos[-1].address")
grpcurl -plaintext -import-path $COSMOS_SDK_DIR/third_party/proto -import-path $COSMOS_SDK_DIR/proto -import-path . -proto ./x/wasm/internal/types/query.proto \
  -d "{\"address\": \"$CONTRACT\"}" localhost:9090  wasmd.x.wasmd.v1beta1.Query/AllContractState | jq

echo "Empty state due to 'burner' contract cleanup"
