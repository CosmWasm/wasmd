# Install Gaia

This guide will explain how to install the `wasmd` and `wasmcli` entrypoints
onto your system. With these installed on a server, you can participate in the
mainnet as either a [Full Node](./join-mainnet.md) or a
[Validator](./validators/validator-setup.md).

## Install Go

Install `go` by following the [official docs](https://golang.org/doc/install).
Remember to set your `$GOPATH` and `$PATH` environment variables, for example:

```bash
mkdir -p $HOME/go/bin
echo "export GOPATH=$HOME/go" >> ~/.bash_profile
echo "export PATH=\$PATH:\$GOPATH/bin" >> ~/.bash_profile
source ~/.bash_profile
```

::: tip
**Go 1.12+** is required for the Cosmos SDK.
:::

## Install the binaries

Next, let's install the latest version of Gaia. Make sure you `git checkout` the
correct [released version](https://github.com/cosmwasm/wasmd/releases).

```bash
git clone -b <latest-release-tag> https://github.com/cosmwasm/wasmd
cd gaia && make install
```

If this command fails due to the following error message, you might have already set `LDFLAGS` prior to running this step.

```
# github.com/cosmwasm/wasmd/cmd/wasmd
flag provided but not defined: -L
usage: link [options] main.o
...
make: *** [install] Error 2
```

Unset this environment variable and try again.

```
LDFLAGS="" make install
```

> _NOTE_: If you still have issues at this step, please check that you have the latest stable version of GO installed.

That will install the `wasmd` and `wasmcli` binaries. Verify that everything is OK:

```bash
$ wasmd version --long
$ wasmcli version --long
```

`wasmcli` for instance should output something similar to:

```shell
name: gaia
server_name: wasmd
client_name: wasmcli
version: 1.0.0
commit: 89e6316a27343304d332aadfe2869847bf52331c
build_tags: netgo,ledger
go: go version go1.12.5 darwin/amd64
```

### Build Tags

Build tags indicate special features that have been enabled in the binary.

| Build Tag | Description                                     |
| --------- | ----------------------------------------------- |
| netgo     | Name resolution will use pure Go code           |
| ledger    | Ledger devices are supported (hardware wallets) |

### Install binary distribution via snap (Linux only)

**Do not use snap at this time to install the binaries for production until we have a reproducible binary system.**

## Developer Workflow

To test any changes made in the SDK or Tendermint, a `replace` clause needs to be added to `go.mod` providing the correct import path.

- Make appropriate changes
- Add `replace github.com/cosmos/cosmos-sdk => /path/to/clone/cosmos-sdk` to `go.mod`
- Run `make clean install` or `make clean build`
- Test changes

## Next

Now you can [join the mainnet](./join-mainnet.md), [the public testnet](./join-testnet.md) or [create you own testnet](./deploy-testnet.md)
