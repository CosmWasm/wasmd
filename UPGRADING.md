# Upgrading

With stargate, we have access to the `x/upgrade` module, which we can use to perform
inline upgrades. Please first read both the basic 
[x/upgrade spec](https://github.com/cosmos/cosmos-sdk/blob/master/x/upgrade/spec/01_concepts.md)
and [go docs](https://godoc.org/github.com/cosmos/cosmos-sdk/x/upgrade#hdr-Performing_Upgrades)
for the background on the module.

In this case, we will demo an update with no state migration. This is for cases when
there is a state-machine-breaking (but not state-breaking) bugfix or enhancement.
There are some
[open issues running some state migrations](https://github.com/cosmos/cosmos-sdk/issues/8265)
and we will wait for that to be fixed before trying those.

The following will lead through running an upgrade on a local node, but the same process
would work on a real network (with more ops and governance coordination).

## Setup

We need to have two different versions of `wasmd` which depend on state-compatible
versions of the Cosmos SDK. We only focus on upgrade starting with stargate. You will
have to use the "dump state and restart" approach to move from launchpad to stargate.

For this demo, we will show an upgrade to our Musselnet going from `v0.12.1` to
`v0.14.0`.

### Handler

You will need to register a handler for the upgrade. This is specific to a particular
testnet and upgrade path, and the default `wasmd` will never have a registered handler
on master. In this case, we make a `musselnet` branch off of `v0.14.0` just
registering one handler with a given name. 

Look at [PR 351](https://github.com/CosmWasm/wasmd/pull/351/files) for an example
of a minimal handler. We do not make any state migrations, but rather use this
as a flag to coordinate all validators to stop the old version at one height, and
start the specified v2 version on the next block.

### Prepare binaries

Let's get the two binaries we want to test, the pre-upgrade and the post-upgrade
binaries. In this case the pre-release is already a published to docker hub and
can be downloaded simply via:

`docker pull cosmwasm/wasmd:v0.12.1`

The post-release is not published, so we can build it ourselves. Check out this
`wasmd` repo, and the proper `musselnet` branch:

```
# use musselnet-v2 tag once that exists
git checkout musselnet
docker build . -t wasmd:musselnet-v2
```

Verify they are both working for you locally:

```
docker run cosmwasm/wasmd:v0.12.1 wasmd version
docker run wasmd:musselnet-v2 wasmd version
```

## Start the pre-release chain

Follow the normal setup stage, but in this case we will want to have super short
governance voting period, 5 minutes rather than 2 days (or 2 weeks!).

**Setup a client with private key**

```sh
## TODO: I think we need to do this locally???
docker volume rm -f musselnet_client

docker run --rm -it \
    -e PASSWORD=1234567890 \
    --mount type=volume,source=musselnet_client,target=/root \
    cosmwasm/wasmd:v0.12.1 /opt/setup_wasmd.sh

# enter "1234567890" when prompted
docker run --rm -it \
    --mount type=volume,source=musselnet_client,target=/root \
    cosmwasm/wasmd:v0.12.1 wasmd keys show -a validator
# use the address returned above here
CLIENT=wasm1anavj4eyxkdljp27sedrdlt9dm26c8a7a8p44l
```

**Setup the blockchain node**

```sh
docker volume rm -f musselnet

# add your testing address here, so you can do something with the client
docker run --rm -it \
    --mount type=volume,source=musselnet,target=/root \
    cosmwasm/wasmd:v0.12.1 /opt/setup_wasmd.sh $CLIENT

# Update the voting times in the genesis file
docker run --rm -it \
    --mount type=volume,source=musselnet,target=/root \
    cosmwasm/wasmd:v0.12.1 sed -ie 's/172800s/300s/' /root/.wasmd/config/genesis.json

# This will start both wasmd and rest-server, only rest-serve output is shown on the screen
docker run --rm -it -p 26657:26657 -p 26656:26656 -p 1317:1317 \
    --mount type=volume,source=musselnet,target=/root \
    cosmwasm/wasmd:v0.12.1 /opt/run_wasmd.sh
```

## Sanity checks

**TODO** move some tokens around

## Vote on the upgrade

**TODO**

## Swap out binaries

**TODO**

## Check final state

**TODO** Same balances in the final one