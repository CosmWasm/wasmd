#!/bin/bash
set -euo pipefail

mkdir -p /root/log
touch /root/log/wasmd.log
./run_wasmd.sh >> /root/log/wasmd.log &

sleep 4
echo Starting Rest Server...

wasmcli rest-server --laddr tcp://0.0.0.0:1317 --trust-node --cors
