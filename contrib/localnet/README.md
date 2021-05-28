## Localnet configuration scripts

For wasmd 0.14.x, CosmWasm 0.13.x.

 - `localnet_*.sh`: Scripts to initialize and launch `wasmd` in a localnet configuration.

In particular, `localnet_vars.sh` can be used to customize and setup the localnet.

It can be invoked from the command line if needed (`source ./localnet_vars.sh`)
in order to have the variables set in the current environment.
In any case, it is being invoked by the other scripts to setup their environment.

 - `contract_*.sh`: Scripts to store, instantiate and query a smart contract.

The `contract_instantiate.sh` and `contract_query.sh` scripts are tailored to the
[`cosmwasm_experiment` smart contract](https://github.com/jstuczyn/cosmwasm_experiment/tree/master/contract).

They will in principle have to be modified to work with other contracts.