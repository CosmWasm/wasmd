# Wasm Zone

[![CircleCI](https://circleci.com/gh/cosmwasm/wasmd/tree/master.svg?style=shield)](https://circleci.com/gh/cosmwasm/wasmd/tree/master)
[![codecov](https://codecov.io/gh/cosmwasm/wasmd/branch/master/graph/badge.svg)](https://codecov.io/gh/cosmwasm/wasmd)
[![Go Report Card](https://goreportcard.com/badge/github.com/cosmwasm/wasmd)](https://goreportcard.com/report/github.com/cosmwasm/wasmd)
[![license](https://img.shields.io/github/license/cosmwasm/wasmd.svg)](https://github.com/cosmwasm/wasmd/blob/master/LICENSE)
[![LoC](https://tokei.rs/b1/github/cosmwasm/wasmd)](https://github.com/cosmwasm/wasmd)
<!-- [![GolangCI](https://golangci.com/badges/github.com/cosmwasm/wasmd.svg)](https://golangci.com/r/github.com/cosmwasm/wasmd) -->

This repository hosts `Wasmd`, the first implementation of a cosmos zone with wasm smart contracts enabled.

This code was forked from the `cosmos/gaia` repository and the majority of the codebase is the same as `gaia`.

**Note**: Requires [Go 1.13+](https://golang.org/dl/)

**Compatibility**: Last merge from `cosmos/gaia` was `d6dfa141e2ae38a1ff9f53fca8078c0822671b95`

## Quick Start

```
make install
make test
```

To set up a single node testnet, [look at the deployment documentation](./docs/deploy-testnet.md).

If you want to deploy a whole cluster, [look at the network scripts](./networks/README.md).

## Dockerized

This is just designed for local testing/CI - DO NOT USE IN PRODUCTION

Build: `docker build  -t wasmd:manual .`

Run:

```sh
docker volume rm -f wasmd_data

# pass password (one time) as env variable for setup, so we don't need to keep typing it
docker run --rm -it \
    -e PASSWORD=my-secret-password \
    --mount type=volume,source=wasmd_data,target=/root \
    wasmd:manual ./setup.sh

# This will start both wasmd and wasmcli rest-server, only wasmcli output is shown on the screen
docker run --rm -it -p 26657:26657 -p 26656:26656 -p 1317:1317 \
    --mount type=volume,source=wasmd_data,target=/root \
    wasmd:manual ./run.sh

# view wasmd logs in another shell
docker run --rm -it \
    --mount type=volume,source=wasmd_data,target=/root,readonly \
    wasmd:manual ./logs.sh
```
