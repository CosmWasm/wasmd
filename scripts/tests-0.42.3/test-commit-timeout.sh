#!/bin/bash

set -eu

NODE_HOME=${NODE_HOME:-"$PWD/.oraid"}
TEST_NODE_HOME="$NODE_HOME/.oraid"

# remove old test default config files
rm -rf $TEST_NODE_HOME
# populate config files in the test directory
oraid version --home $TEST_NODE_HOME

# Extract timeout commit from the test commit file. Should be 500ms
timeout_commit=$(sed -n 's/^[[:space:]]*timeout_commit[[:space:]]*=[[:space:]]*"\(.*\)".*/\1/p' $TEST_NODE_HOME/config/config.toml)
echo "timeout commit: $timeout_commit"

if ! [[ $timeout_commit == "500ms"  ]] ; then
   echo "Commit timeout test failed"; exit 1
fi

echo "Commit timeout test passed!"