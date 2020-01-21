approach 1:

Build: `docker build  -t wasmd:manual .`


Start from the outside
```sh
WORK_DIR=$(pwd)/tmp
mkdir -p ${WORK_DIR}

docker run -v ${WORK_DIR}/.wasmd:/root/.wasmd -it wasmd:manual wasmd init --chain-id=testing testing
docker run -v ${WORK_DIR}:/root -it wasmd:manual wasmcli keys add validator

# # here you must blindly type the passphrase, we tail to remove the prompt
# VALIDATOR=$(docker run -v ${WORK_DIR}:/root -it wasmd:manual wasmcli keys show validator -a | tail -1)
# # but we don't need this approach
# docker run -v ${WORK_DIR}:/root -it wasmd:manual wasmd add-genesis-account $VALIDATOR 1000000000stake,1000000000validatortoken

docker run -v ${WORK_DIR}:/root -it wasmd:manual wasmd add-genesis-account validator 1000000000stake,1000000000validatortoken

docker run -v ${WORK_DIR}:/root -it wasmd:manual wasmd gentx --name validator
docker run -v ${WORK_DIR}:/root -it wasmd:manual wasmd collect-gentxs
docker run -v ${WORK_DIR}:/root -it wasmd:manual wasmd start

```
Fails with `'$(wasmcli keys show validator -a)' `

approach 2:
Start within the container:           

build 
`docker build  -t wasmd:demo -f Dockerfile_demo .`

fails with
```sh
Step 9/14 : RUN echo "xxxxxxxxx" | wasmcli keys add validator
 ---> Running in cd1928c57cae
EOF
panic: too many failed passphrase attempts

goroutine 1 [running]:
github.com/cosmos/cosmos-sdk/crypto/keys.keyringKeybase.writeInfo(0x13b4720, 0xc0008ea0f0, 0x7ffdf0302f1a, 0x9, 0x13b4860, 0xc0008ea2a0)
```

approach 3:

Use scripts inside docker:

Build: `docker build  -t wasmd:manual .`

Run:

```sh
docker volume rm -f wasmd_data

# pick a simple (8 char) passphrase for testing.. you will type it many times
docker run --rm -it \
    --mount type=volume,source=wasmd_data,target=/root \
    wasmd:manual ./setup.sh

docker run --rm -it -p 26657:26657 -p 26656:26656 -p 1317:1317 \
    --mount type=volume,source=wasmd_data,target=/root \
    wasmd:manual ./run.sh

# view logs in another shell
docker run --rm -it \
    --mount type=volume,source=wasmd_data,target=/root,readonly \
    wasmd:manual ./logs.sh
```
