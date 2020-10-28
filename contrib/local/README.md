# Dev scripts
For manual testing. Works on my box(*) ... 



*) OSX

```
make install
cd contrib/local
rm -rf /tmp/trash
HOME=/tmp/trash bash setup_wasmd.sh
HOME=/tmp/trash bash start_node.sh
```

Next shell:

```
cd contrib/local
./01-accounts.sh 
./02-contracts.sh
```