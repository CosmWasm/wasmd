#!/bin/sh

set -eo pipefail

PROJECT_PROTO_DIR=x/wasm/internal/types/
COSMOS_SDK_DIR=${COSMOS_SDK_DIR:-$(go list -f "{{ .Dir }}" -m github.com/cosmos/cosmos-sdk)}

# Generate Go types from protobuf
protoc \
  -I=. \
  -I="$COSMOS_SDK_DIR/third_party/proto" \
  -I="$COSMOS_SDK_DIR/proto" \
  --gocosmos_out=Mgoogle/protobuf/any.proto=github.com/cosmos/cosmos-sdk/codec/types,Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,plugins=interfacetype+grpc,paths=source_relative:. \
  --grpc-gateway_out .\
  --grpc-gateway_opt logtostderr=true \
  --grpc-gateway_opt paths=Mgoogle/protobuf/any.proto=github.com/cosmos/cosmos-sdk/codec/types,Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,paths=source_relative \
  --doc_out=./doc \
  --doc_opt=markdown,proto.md \
  $(find "${PROJECT_PROTO_DIR}" -maxdepth 1 -name '*.proto')
