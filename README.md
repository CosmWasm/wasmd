# Wasm Zone

[![CircleCI](https://circleci.com/gh/cosmwasm/wasmd/tree/master.svg?style=shield)](https://circleci.com/gh/cosmwasm/wasmd/tree/master)
[![codecov](https://codecov.io/gh/cosmwasm/wasmd/branch/master/graph/badge.svg)](https://codecov.io/gh/cosmwasm/wasmd)
[![Go Report Card](https://goreportcard.com/badge/github.com/CosmWasm/wasmd)](https://goreportcard.com/report/github.com/CosmWasm/wasmd)
[![license](https://img.shields.io/github/license/cosmwasm/wasmd.svg)](https://github.com/CosmWasm/wasmd/blob/master/LICENSE)
[![LoC](https://tokei.rs/b1/github/cosmwasm/wasmd)](https://github.com/CosmWasm/wasmd)
<!-- [![GolangCI](https://golangci.com/badges/github.com/CosmWasm/wasmd.svg)](https://golangci.com/r/github.com/CosmWasm/wasmd) -->

This repository hosts `Wasmd`, the first implementation of a cosmos zone with wasm smart contracts enabled.

This code was forked from the `cosmos/gaia` repository and the majority of the codebase is the same as `gaia`.

**Note**: Requires [Go 1.14+](https://golang.org/dl/)

**Compatibility**: Last merge from `cosmos/gaia` was `20a071cf84b965d0892c5a216b32c44963a83173` (Apr 27, 2020)

`cosmos-sdk` was updated to `goz-phase-3` (`ba70f4d4d`)

## Supported Systems

The supported systems are limited by the dlls created in [`go-cosmwasm`](https://github.com/CosmWasm/go-cosmwasm). In particular, **we only support MacOS and Linux**. 
For linux, the default is to build for glibc, and we cross-compile with CentOS 7 to provide
backwards compatibility for `glibc 2.12+`. This includes all known supported distributions
using glibc (CentOS 7 uses 2.12, obsolete Debian Jessy uses 2.19). 

As of `0.9.0` we support `muslc` Linux systems, in particular **Alpine linux**,
which is popular in docker distributions. Note that we do **not** store the
static `muslc` build in the repo, so you must compile this yourself, and pass `-tags muslc`.
Please look at the [`Dockerfile`](./Dockerfile) for an example of how we build a static Go
binary for `muslc`. (Or just use this Dockerfile for your production setup).


## Stability

**This is alpha software, do not run on a production system.** Notably, we currently provide **no migration path** not even "dump state and restart" to move to future versions. At **beta** we will begin to offer migrations and better backwards compatibility guarantees.

With the `v0.6.0` tag, we entered semver. That means anything with `v0.6.x` tags is compatible with each other, 
and everything with `v0.7.x` tags is compatible with each other. 
We will have a series of minor version updates prior to `v1.0.0`, 
where we offer strong backwards compatibility guarantees. 

We are currently on `v0.9` and targeting `v0.10` for late July, at which point
we will have feature freeze for the `v1` tag and only perform needed bugfixes
(which may be API breaking).

Thank you to all projects who have run this code in your testnets and
given feedback to improve stability.

## Encoding
The used cosmos-sdk version is in transition migrating from amino encoding to protobuf for state. So are we now.
We use standard cosmos-sdk encoding (amino) for all sdk Messages. However, the message body sent to all contracts, as well as the internal state is encoded using JSON. Cosmwasm allows arbitrary bytes with the contract itself responsible for decodng. For better UX, we often use `json.RawMessage` to contain these bytes, which enforces that it is valid json, but also give a much more readable interface.  If you want to use another encoding in the contracts, that is a relatively minor change to wasmd but would currently require a fork. Please open in issue if this is important for your use case.

## Quick Start

```
make install
make test
```
if you are using a linux without X or headless linux, look at [this article](https://ahelpme.com/linux/dbusexception-could-not-get-owner-of-name-org-freedesktop-secrets-no-such-name) or [#31](https://github.com/CosmWasm/wasmd/issues/31#issuecomment-577058321).

To set up a single node testnet, [look at the deployment documentation](./docs/deploy-testnet.md).

If you want to deploy a whole cluster, [look at the network scripts](./networks/README.md).

## Dockerized

We provide a docker image to help with test setups. There are two modes to use it

Build: `docker build -t cosmwasm/wasmd:latest .`  or pull from dockerhub

### Dev server

Bring up a local node with a test account containing tokens

This is just designed for local testing/CI - do not use these scripts in production.
Very likely you will assign tokens to accounts whose mnemonics are public on github.

```sh
docker volume rm -f wasmd_data

# pass password (one time) as env variable for setup, so we don't need to keep typing it
# add some addresses that you have private keys for (locally) to give them genesis funds
docker run --rm -it \
    -e PASSWORD=xxxxxxxxx \
    --mount type=volume,source=wasmd_data,target=/root \
    cosmwasm/wasmd:latest ./setup.sh cosmos1pkptre7fdkl6gfrzlesjjvhxhlc3r4gmmk8rs6

# This will start both wasmd and wasmcli rest-server, only wasmcli output is shown on the screen
docker run --rm -it -p 26657:26657 -p 26656:26656 -p 1317:1317 \
    --mount type=volume,source=wasmd_data,target=/root \
    cosmwasm/wasmd:latest ./run_all.sh

# view wasmd logs in another shell
docker run --rm -it \
    --mount type=volume,source=wasmd_data,target=/root,readonly \
    cosmwasm/wasmd:latest ./logs.sh
```

### CI

For CI, we want to generate a template one time and save to disk/repo. Then we can start a chain copying the initial state, but not modifying it. This lets us get the same, fresh start every time.

```sh
# Init chain and pass addresses so they are non-empty accounts
rm -rf ./template && mkdir ./template
docker run --rm -it \
    -e PASSWORD=xxxxxxxxx \
    --mount type=bind,source=$(pwd)/template,target=/root \
    cosmwasm/wasmd:latest ./setup.sh cosmos1pkptre7fdkl6gfrzlesjjvhxhlc3r4gmmk8rs6

sudo chown -R $(id -u):$(id -g) ./template

# FIRST TIME
# bind to non-/root and pass an argument to run.sh to copy the template into /root
# we need wasmd_data volume mount not just for restart, but also to view logs
docker volume rm -f wasmd_data
docker run --rm -it -p 26657:26657 -p 26656:26656 -p 1317:1317 \
    --mount type=bind,source=$(pwd)/template,target=/template \
    --mount type=volume,source=wasmd_data,target=/root \
    cosmwasm/wasmd:latest ./run_all.sh /template

# RESTART CHAIN with existing state
docker run --rm -it -p 26657:26657 -p 26656:26656 -p 1317:1317 \
    --mount type=volume,source=wasmd_data,target=/root \
    cosmwasm/wasmd:latest ./run_all.sh

# view wasmd logs in another shell
docker run --rm -it \
    --mount type=volume,source=wasmd_data,target=/root,readonly \
    cosmwasm/wasmd:latest ./logs.sh
```

## Contributors

Much thanks to all who have contributed to this project, from this app, to the `cosmwasm` framework, to example contracts and documentation.
Or even testing the app and bringing up critical issues. The following have helped bring this project to life:

* Ethan Frey [ethanfrey](https://github.com/ethanfrey)
* Simon Warta [webmaster128](https://github.com/webmaster128)
* Alex Peters [alpe](https://github.com/alpe)
* Aaron Craelius [aaronc](https://github.com/aaronc)
* Sunny Aggarwal [sunnya97](https://github.com/sunnya97)
* Cory Levinson [clevinson](https://github.com/clevinson)
* Sahith Narahari [sahith-narahari](https://github.com/sahith-narahari)
* Jehan Tremback [jtremback](https://github.com/jtremback)
* Shane Vitarana [shanev](https://github.com/shanev)
* Billy Rennekamp [okwme](https://github.com/okwme)
* Westaking [westaking](https://github.com/westaking)
* Marko [marbar3778](https://github.com/marbar3778)
* JayB [kogisin](https://github.com/kogisin)
* Rick Dudley [AFDudley](https://github.com/AFDudley)
* KamiD [KamiD](https://github.com/KamiD)
* Valery Litvin [litvintech](https://github.com/litvintech)

Sorry if I forgot you from this list, just contact me or add yourself in a PR :)
