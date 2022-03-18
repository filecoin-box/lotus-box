module github.com/luluup777/lotus-box

go 1.16

require (
	github.com/filecoin-project/filecoin-ffi v0.30.4-0.20200910194244-f640612a1a1f
	github.com/filecoin-project/go-address v0.0.6
	github.com/filecoin-project/go-bitfield v0.2.4
	github.com/filecoin-project/go-state-types v0.1.3
	github.com/filecoin-project/lotus v1.15.0
	github.com/filecoin-project/specs-actors/v2 v2.3.6
	github.com/filecoin-project/specs-storage v0.2.0
	github.com/ipfs/go-ipld-cbor v0.0.6
	github.com/ipfs/go-log/v2 v2.5.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/urfave/cli/v2 v2.3.0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/lotus/extern/filecoin-ffi

replace github.com/filecoin-project/lotus => ./extern/lotus
