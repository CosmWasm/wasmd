#!/usr/bin/env bash

set -eo pipefail


protoc \
-I=. \
-I="$(go list -f "{{ .Dir }}" -m github.com/cosmos/cosmos-sdk)/third_party/proto" \
-I="$(go list -f "{{ .Dir }}" -m github.com/cosmos/cosmos-sdk)/proto" \
--gocosmos_out=\
Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types,\
plugins=interfacetype+grpc,paths=source_relative:. \
./x/wasm/internal/types/types.proto ./x/wasm/internal/types/query.proto ./x/wasm/internal/types/msg.proto \
./x/wasm/internal/types/proposal.proto ./x/wasm/internal/types/genesis.proto \
./x/wasm/internal/types/ibc.proto