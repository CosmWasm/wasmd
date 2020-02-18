#!/bin/bash
set -euo pipefail

mkdir -p /root/log
touch /root/log/wasmd.log
./run_wasmd.sh >> /root/log/wasmd.log &

sleep 4
echo Starting Rest Server...

./run_wasmcli.sh
