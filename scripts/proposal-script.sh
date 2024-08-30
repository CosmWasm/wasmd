#!/bin/sh
set -ux

VERSION=${VERSION:-"v0.41.3"}
HEIGHT=${HEIGHT:-100}
VALIDATOR1_HOME=${VALIDATOR1_HOME:-"$HOME/.oraid/validator1"}
VALIDATOR1_ARGS="--from validator1 --chain-id testing -y --keyring-backend test --home $VALIDATOR1_HOME --fees 2orai -b sync"
VALIDATOR2_ARGS="--from validator2 --chain-id testing -y --keyring-backend test --home $HOME/.oraid/validator2 --fees 2orai -b sync"

update_proposal() {
    cat $PWD/scripts/json/proposal.json | jq "$1" >$PWD/scripts/json/temp_proposal.json && mv $PWD/scripts/json/temp_proposal.json $PWD/scripts/json/proposal.json
}

# update authority proposal.json
MODULE_ACCOUNT=$(oraid query auth module-account gov)
update_proposal ".messages[0][\"authority\"]=\"$MODULE_ACCOUNT\""
# update plan -> name proposal.json
update_proposal ".messages[0][\"plan\"][\"name\"]=\"$VERSION\""
# update plan -> height proposal.json
update_proposal ".messages[0][\"plan\"][\"height\"]=\"$HEIGHT\""
# update metadata proposal.json
# METADATA_BASE64=$(base64 -i $PWD/scripts/json/proposal_metadata.json)
# update_proposal ".metadata=\"$METADATA_BASE64\""

oraid tx gov submit-proposal $PWD/scripts/json/proposal.json $VALIDATOR1_ARGS

# sleep 2s before vote
sleep 2
oraid tx gov vote 1 yes $VALIDATOR1_ARGS && oraid tx gov vote 1 yes $VALIDATOR2_ARGS
