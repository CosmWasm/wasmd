# Relayer tests

These scripts helps to test go-relayer with two local wasmd chains. \
Make sure you run below scripts under `wasmd/contrib/relayer-tests` directory.

- `./init_two_chainz_relayer.sh` will spin two chains and runs
- `./one_chain.sh` will spin a single chain. This script used by the one above
-`./test_ibc_transfer.sh` will setup a path between chains and send tokens between chains.
