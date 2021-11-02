
# Miner Tools
This is a toolbox project to facilitate miners to better participate in filecoin.

## lotus-redo
This tool provides sector redo function, which can repair sectors when they are damaged.

**build**:

- git clone https://github.com/filecoin-project/lotus.git
- cd lotus/ && RUSTFLAGS="-C target-cpu=native -g" FFI_BUILD_FROM_SOURCE="1" make clean all
- cd ../ && git clone https://github.com/luluup777/lotus-box.git
- cd lotus-box/ && go mod tidy
- go build -o lotus-redo cmd/lotus-redo/main.go

**start**:

```
./lotus-redo -h           
NAME:
   lotus-redo - lotus redo sector

USAGE:
   lotus-redo [global options] command [command options] [arguments...]

VERSION:
   v0.1

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --sids value      redo sector ids, if there are more than one, separate commas. ps: 1,2
   --sdir value      The directory where the redo sector is stored
   --parallel value  num run in parallel (default: 1)
   --help, -h        show help (default: false)
   --version, -v     print the version (default: false)
```

Need to set environment variables:

- `FULLNODE_API_INFO`
- `MINER_API_INFO`

## lotus-wdpost

This tool can simulate WindowsPost to determine whether the sector is correct. Allows you to determine whether a sector is intact.

