#!/bin/bash

source ./localnet_vars.sh

# Resets wasmd configuration files
wasmd unsafe-reset-all localnet --home ${APP_HOME}
