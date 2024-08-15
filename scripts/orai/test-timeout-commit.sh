#!/bin/bash

rm -rf .oraid

oraid init --chain-id "foo" "bar" --home .oraid

timeout_commit=$(awk -F ' = ' '/^timeout_commit/ {gsub(/"/, "", $2); print $2}' .oraid/config/config.toml)

echo $timeout_commit

if ! [[ $timeout_commit =~ "500ms" ]] ; then
   echo "Timeout commit is not 500ms. Test Failed"; exit 1
fi

echo "Test Passed"