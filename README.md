
# Miner Tools
This is a toolbox project to facilitate miners to better participate in filecoin.

**build**:

- git clone https://github.com/luluup777/lotus-box.git
- [set the **environment variables** for compiling **lotus**](https://lotus.filecoin.io/docs/set-up/install/#native-filecoin-ffi)
- make all

## lotus-redo
This tool provides sector redo function, which can repair sectors when they are damaged.

### start:

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
   --sids value         redo sector ids, if there are more than one, separate commas. ps: 1,2
   --seal-dir value     redo sector seal directory
   --storage-dir value  the storage directory where the redo sector is stored
   --parallel value     num run in parallel (default: 1)
   --help, -h           show help (default: false)
   --version, -v        print the version (default: false)
```

Need to set environment variables:

- `FULLNODE_API_INFO`
- `MINER_API_INFO`
- `FIL_PROOFS_USE_MULTICORE_SDR=1`
- `FIL_PROOFS_MAXIMIZE_CACHING=1`
- `FIL_PROOFS_USE_GPU_COLUMN_BUILDER=1`
- `FIL_PROOFS_USE_GPU_TREE_BUILDER=1`

### todo

- deal sector redo

## lotus-wdpost

This tool can simulate WindowsPost to determine whether the sector is correct. Allows you to determine whether a sector is intact.

**start**

```
NAME:
   lotus-wdpost - lotus wdpost emulator

USAGE:
   lotus-wdpost [global options] command [command options] [arguments...]

VERSION:
   v0.1

COMMANDS:
   s-emulator  sector WindowPost simulator
   p-emulator  partition WindowPost simulator
   d-emulator  deadline WindowPost simulator
   help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help (default: false)
   --version, -v  print the version (default: false)
```

Need to set environment variables:

- `FULLNODE_API_INFO`
