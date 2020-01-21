approach 1:
Start from the outside
```sh
WORK_DIR=$(pwd)/tmp
mkdir -p ${WORK_DIR}

docker run -v ${WORK_DIR}/.wasmd:/root/.wasmd -it wasmd:manual wasmd init --chain-id=testing testing
docker run -v ${WORK_DIR}:/root -it wasmd:manual wasmcli keys add validator
docker run -v ${WORK_DIR}:/root -it wasmd:manual wasmd add-genesis-account '$(wasmcli keys show validator -a)' 1000000000stake,1000000000validatortoken
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