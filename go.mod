module github.com/CosmWasm/wasmd

go 1.13

require (
	github.com/CosmWasm/go-cosmwasm v0.10.0-alpha2
	github.com/cosmos/cosmos-sdk v0.39.1-0.20200727105938-302487cdbcc9
	github.com/golang/mock v1.4.3 // indirect
	github.com/google/gofuzz v1.0.0
	github.com/gorilla/mux v1.7.4
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/otiai10/copy v1.0.2
	github.com/otiai10/curr v0.0.0-20190513014714-f5a3d24e5776 // indirect
	github.com/pkg/errors v0.9.1
	github.com/snikch/goodman v0.0.0-20171125024755-10e37e294daa
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.3
	github.com/stretchr/testify v1.6.1
	github.com/tendermint/go-amino v0.15.1
	github.com/tendermint/tendermint v0.33.6
	github.com/tendermint/tm-db v0.5.1
	go.etcd.io/bbolt v1.3.4 // indirect
	golang.org/x/sys v0.0.0-20200602225109-6fdc65e7d980 // indirect
	gopkg.in/yaml.v2 v2.3.0
)

replace github.com/keybase/go-keychain => github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4

replace github.com/cosmos/cosmos-sdk => github.com/CosmWasm/cosmos-sdk v0.38.1-0.20200727124038-6c7f67fa908f