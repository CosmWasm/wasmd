#!/bin/sh
#set -euo pipefail

wasmcli rest-server --laddr tcp://0.0.0.0:1317 --trust-node --cors
