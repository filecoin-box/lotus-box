module lotus-box

go 1.16

require (
	github.com/filecoin-project/lotus v1.13.0
	github.com/ipfs/go-log/v2 v2.3.0
	github.com/urfave/cli/v2 v2.2.0
)

replace github.com/filecoin-project/filecoin-ffi => ../lotus/extern/filecoin-ffi
