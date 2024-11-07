# Demo setup for layer

## Local build

### Compile and wipe out old config

```bash
make install
which layerd

ls -l ~/.layer
rm -rf ~/.layer
```

### Set up a key for staking

```bash
layerd keys add staker
layerd keys list
```

### Set up genesis with tokens

```bash
layerd init localnode --chain-id layer-local --default-denom ulayer
ls -l ~/.layer/config

# add some tokens
STAKER=$(layerd keys show -a staker)
layerd genesis add-genesis-account $STAKER 123456789000000ulayer

# commit to our validator
layerd genesis gentx staker 54321000000ulayer --home=$HOME/.layer \
    --chain-id=layer-local \
    --moniker="default-validator" \
    --commission-max-change-rate=0.01 \
    --commission-max-rate=1.0 \
    --commission-rate=0.07 \
    --details="This is the default staker"

layerd genesis collect-gentxs
```

### Adjust Config

**TODO** We want to make 1s blocks, adjust some of the ports, etc

### Start Chain

```bash
layerd start

# check it is up
curl localhost:26657/status
curl localhost:26657/status | jq .result.node_info
curl localhost:26657/status | jq .result.sync_info
```

## Docker

### Compile and wipe out old config

```bash
docker build -t lay3rlabs/layerd:0.53.0 .

docker run --rm lay3rlabs/layerd:0.53.0 version
docker run --rm lay3rlabs/layerd:0.53.0 help

# note: we could use a docker volume rather than filesystem for deploy
ls -l ~/.layer
rm -rf ~/.layer
```

### Set up a key for staking

```bash
docker run --rm -v $HOME/.layer:/root/.layer lay3rlabs/layerd:0.53.0 keys --keyring-backend=test list
docker run --rm -v $HOME/.layer:/root/.layer lay3rlabs/layerd:0.53.0 keys --keyring-backend=test add staker
docker run --rm -v $HOME/.layer:/root/.layer lay3rlabs/layerd:0.53.0 keys --keyring-backend=test list
```

### Set up genesis with tokens

```bash
docker run --rm -v $HOME/.layer:/root/.layer lay3rlabs/layerd:0.53.0 init localnode --chain-id layer-local --default-denom ulayer
ls -l ~/.layer/config

# add some tokens
STAKER=$(docker run --rm -v $HOME/.layer:/root/.layer lay3rlabs/layerd:0.53.0 keys --keyring-backend=test show -a staker)
docker run --rm -v $HOME/.layer:/root/.layer lay3rlabs/layerd:0.53.0 genesis add-genesis-account $STAKER 123456789000000ulayer

# commit to our validator
docker run --rm -v $HOME/.layer:/root/.layer lay3rlabs/layerd:0.53.0 genesis gentx staker 54321000000ulayer --home=/root/.layer \
    --keyring-backend=test \
    --chain-id=layer-local \
    --moniker="default-validator" \
    --commission-max-change-rate=0.01 \
    --commission-max-rate=1.0 \
    --commission-rate=0.07 \
    --details="This is the default staker"

docker run --rm -v $HOME/.layer:/root/.layer lay3rlabs/layerd:0.53.0 genesis collect-gentxs
```

### Adjust Config

**TODO** We want to make 1s blocks, adjust some of the ports, etc

Need to serve on 0.0.0.0 not localhost

### Start Chain

```bash
docker run --rm -p 26657:26657 -p 9090:9090 -p 1317:1317 -v $HOME/.layer:/root/.layer lay3rlabs/layerd:0.53.0 start --rpc.laddr tcp://0.0.0.0:26657

# check it is up
curl localhost:26657/status
curl localhost:26657/status | jq .result.node_info
curl localhost:26657/status | jq .result.sync_info
```
