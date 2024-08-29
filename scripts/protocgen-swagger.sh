#!/usr/bin/env bash

set -eo pipefail

# go get ./...
# go install github.com/cosmos/gogoproto/protoc-gen-gocosmos
# go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger@v1.16.0
# apk add nodejs-current npm
# npm install -g swagger-combine

BASEDIR=$(dirname $0)
PROJECTDIR=$BASEDIR/..
# default is tmp folder
SOURCEDIR=$(realpath ${1:-$PROJECTDIR/tmp})
DOC_DIR=$(realpath $PROJECTDIR/doc)

COSMOS_SDK_DIR=${COSMOS_SDK_DIR:-$(go list -f "{{ .Dir }}" -m github.com/cosmos/cosmos-sdk)}
COSMOS_WASM_DIR=${COSMOS_WASM_DIR:-$(go list -f "{{ .Dir }}" -m github.com/CosmWasm/wasmd)}
IBC_DIR=${IBC_DIR:-$(go list -f "{{ .Dir }}" -m github.com/cosmos/ibc-go/v4)}
INTERCHAIN_DIR=${INTERCHAIN_DIR:-$(go list -f "{{ .Dir }}" -m github.com/cosmos/interchain-accounts)}

# scan all folders that contain proto file
proto_dirs=$(find $PROJECTDIR/proto $COSMOS_SDK_DIR/proto $COSMOS_SDK_DIR/third_party/proto $INTERCHAIN_DIR/proto $IBC_DIR/proto $COSMOS_WASM_DIR -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)

GEN_DIR=$SOURCEDIR/swagger-gen
# clean swagger files
rm -rf $GEN_DIR
mkdir -p $GEN_DIR

for dir in $proto_dirs; do
  # generate swagger files (filter query files)
  query_file=$(find "${dir}" -maxdepth 1 \( -name 'query.proto' -o -name 'service.proto' \))
  if [[ ! -z "$query_file" ]]; then   
    buf alpha protoc  \
    -I="$PROJECTDIR/proto" \
    -I="$COSMOS_WASM_DIR/proto" \
    -I="$IBC_DIR/proto" \
    -I="$INTERCHAIN_DIR/proto" \
    -I="$COSMOS_SDK_DIR/third_party/proto" \
    -I="$COSMOS_SDK_DIR/proto" \
    --gocosmos_out=Mgoogle/protobuf/any.proto=github.com/cosmos/cosmos-sdk/codec/types,Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,plugins=interfacetype+grpc,paths=source_relative:$COSMOS_SDK_DIR \
    --swagger_out=$GEN_DIR \
    --swagger_opt=logtostderr=true,fqn_for_swagger_name=true,simple_operation_ids=true \
    $query_file
        
  fi
done

sed -i 's/UpgradedConsensusState/UpgradedIBCConsensusState/' $GEN_DIR/ibc/core/client/v1/query.swagger.json
sed -i 's/InterchainAccount/IBCInterchainAccount/' $GEN_DIR/ibc/applications/interchain_accounts/controller/v1/query.swagger.json

swagger_files=$(find $GEN_DIR/ibc $GEN_DIR/cosmwasm $GEN_DIR/intertx -name 'query.swagger.json' | xargs)

node -e "var fs = require('fs'),file='$COSMOS_SDK_DIR/client/docs/config.json',result = fs.readFileSync(file).toString().replace('./client','$COSMOS_SDK_DIR/client').replace(/.\/tmp-swagger-gen/g, '$GEN_DIR');
var swaggerFiles = '$swagger_files'.split(' '), obj = JSON.parse(result);
swaggerFiles.forEach(swaggerFile => {

  obj.apis.push({
      url: swaggerFile,
      operationIds: {
          rename: {
              Params: swaggerFile.split('/').slice(3,-2).map(dir=>dir[0].toUpperCase() + dir.slice(1)).join('') + 'Params'
          }    
      }
  });
});

fs.writeFileSync('$GEN_DIR/config.json', JSON.stringify(obj, null, 2));
"


# combine swagger files
# all the individual swagger files need to be configured in `config.json` for merging
swagger-combine $GEN_DIR/config.json -o $DOC_DIR/swagger-ui/swagger.yaml -f yaml --continueOnConflictingPaths true --includeDefinitions true
