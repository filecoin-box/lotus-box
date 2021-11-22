package main

import (
	"context"
	"crypto/rand"
	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/specs-actors/v2/actors/runtime/proof"
	cbor "github.com/ipfs/go-ipld-cbor"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"lotus-box/util"
	"strconv"
	"strings"
)

var log = logging.Logger("wdpost")

func main() {
	_ = logging.SetLogLevel("*", "INFO")

	app := &cli.App{
		Name:                 "lotus-wdpost",
		Usage:                "lotus wdpost emulator",
		Version:              "v0.1",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			sectorEmulator,
			partitionEmulator,
			deadlineEmulator,
		},
	}

	app.Setup()
	lcli.RunApp(app)
}

var sectorEmulator = &cli.Command{
	Name:  "s-emulator",
	Usage: "sector WindowPost simulator",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "sids",
			Usage: "simulate sector ids, if there are more than one, separate commas. ps: 1,2",
			Value: "",
		}, &cli.StringFlag{
			Name:  "sdir",
			Usage: "The directory where the redo sector is stored",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "actor",
			Usage: "miner actor id",
		},
	},
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := lcli.GetFullNodeAPIV1(cctx)
		if err != nil {
			return err
		}
		defer closer()

		sidsStr := cctx.String("sids")
		sids := strings.Split(sidsStr, ",")

		sbit := bitfield.New()
		for _, id := range sids {
			sid, err := strconv.Atoi(id)
			if err != nil {
				log.Warnw("sector id parsing failed", "id", id)
				continue
			}
			sbit.Set(uint64(sid))
		}

		sdir, err := getSdir(cctx)
		if err != nil {
			return err
		}

		maddr, err := util.GetActorAddress(cctx)
		if err != nil {
			return err
		}

		amid, err := addr.IDFromAddress(maddr)
		if err != nil {
			return err
		}

		sInfo, err := getSectorInfo(nodeApi, maddr, sbit)
		if err != nil {
			return err
		}

		err = wdpostEmulator(util.NewProvider(sdir), abi.ActorID(amid), sInfo)
		if err != nil {
			return err
		}

		sectorIds, _ := sbit.All(100000)
		log.Infow("wdpost simulation is successful", "sids", sectorIds)
		return nil
	},
}

var partitionEmulator = &cli.Command{
	Name:  "p-emulator",
	Usage: "partition WindowPost simulator",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "deadline",
			Usage: "deadline id",
			Value: 0,
		}, &cli.IntFlag{
			Name:  "partition",
			Usage: "partition id",
			Value: 0,
		}, &cli.StringFlag{
			Name:  "sdir",
			Usage: "The directory where the redo sector is stored",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "actor",
			Usage: "miner actor id",
		},
	},
	Action: func(cctx *cli.Context) error {
		deadlineID := cctx.Int("deadline")
		partitionID := cctx.Int("partition")
		if deadlineID < 0 || deadlineID > 47 {
			return xerrors.New("--deadline must be between 0 and 47")
		}

		nodeApi, closer, err := lcli.GetFullNodeAPIV1(cctx)
		if err != nil {
			return err
		}
		defer closer()

		maddr, err := util.GetActorAddress(cctx)
		if err != nil {
			return err
		}

		amid, err := addr.IDFromAddress(maddr)
		if err != nil {
			return err
		}

		head, err := nodeApi.ChainHead(context.Background())
		if err != nil {
			return err
		}

		mact, err := nodeApi.StateGetActor(context.Background(), maddr, head.Key())
		if err != nil {
			return err
		}

		tbs := blockstore.NewTieredBstore(blockstore.NewAPIBlockstore(nodeApi), blockstore.NewMemory())
		mas, err := miner.Load(adt.WrapStore(context.Background(), cbor.NewCborStore(tbs)), mact)
		if err != nil {
			return err
		}

		dl, err := mas.LoadDeadline(uint64(deadlineID))
		if err != nil {
			return err
		}

		part, err := dl.LoadPartition(uint64(partitionID))
		if err != nil {
			return err
		}

		liveSector, err := part.LiveSectors()
		if err != nil {
			return err
		}

		sInfo, err := getSectorInfo(nodeApi, maddr, liveSector)
		if err != nil {
			return err
		}

		sdir, err := getSdir(cctx)
		if err != nil {
			return err
		}

		err = wdpostEmulator(util.NewProvider(sdir), abi.ActorID(amid), sInfo)
		if err != nil {
			return err
		}

		sids, _ := liveSector.All(1000000)
		log.Infow("wdpost simulation is successful", "sids", sids)
		return nil
	},
}
var deadlineEmulator = &cli.Command{
	Name:  "d-emulator",
	Usage: "deadline WindowPost simulator",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "deadline",
			Usage: "deadline id",
			Value: 0,
		}, &cli.StringFlag{
			Name:  "sdir",
			Usage: "The directory where the redo sector is stored",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "actor",
			Usage: "miner actor id",
		},
	},
	Action: func(cctx *cli.Context) error {
		deadlineID := cctx.Int("deadline")
		if deadlineID < 0 || deadlineID > 47 {
			return xerrors.New("--deadline must be between 0 and 47")
		}

		nodeApi, closer, err := lcli.GetFullNodeAPIV1(cctx)
		if err != nil {
			return err
		}
		defer closer()

		maddr, err := util.GetActorAddress(cctx)
		if err != nil {
			return err
		}

		amid, err := addr.IDFromAddress(maddr)
		if err != nil {
			return err
		}

		sdir, err := getSdir(cctx)
		if err != nil {
			return err
		}

		head, err := nodeApi.ChainHead(context.Background())
		if err != nil {
			return err
		}

		mact, err := nodeApi.StateGetActor(context.Background(), maddr, head.Key())
		if err != nil {
			return err
		}

		tbs := blockstore.NewTieredBstore(blockstore.NewAPIBlockstore(nodeApi), blockstore.NewMemory())
		mas, err := miner.Load(adt.WrapStore(context.Background(), cbor.NewCborStore(tbs)), mact)
		if err != nil {
			return err
		}

		dl, err := mas.LoadDeadline(uint64(deadlineID))
		if err != nil {
			return err
		}

		err = dl.ForEachPartition(func(idx uint64, part miner.Partition) error {
			liveSector, err := part.LiveSectors()
			if err != nil {
				return err
			}

			sInfo, err := getSectorInfo(nodeApi, maddr, liveSector)
			if err != nil {
				return err
			}

			err = wdpostEmulator(util.NewProvider(sdir), abi.ActorID(amid), sInfo)
			if err != nil {
				log.Warnw("wdpost emulator err", "deadlineID", deadlineID, "partitionID", idx)
				return err
			}

			sids, _ := liveSector.All(1000000)
			log.Infow("wdpost simulation is successful", "sids", sids)
			return nil
		})

		return nil
	},
}

func getSdir(cctx *cli.Context) (string, error) {
	sdir := cctx.String("sdir")
	if sdir == "" {
		return "", xerrors.New("--sdir err")
	}
	return sdir, nil
}

func getSectorInfo(nodeApi api.FullNode, maddr addr.Address, sectors bitfield.BitField) ([]proof.SectorInfo, error) {
	ss, err := nodeApi.StateMinerSectors(context.Background(), maddr, &sectors, types.EmptyTSK)
	if err != nil || len(ss) == 0 {
		return nil, err
	}

	substitute := proof.SectorInfo{
		SectorNumber: ss[0].SectorNumber,
		SealedCID:    ss[0].SealedCID,
		SealProof:    ss[0].SealProof,
	}

	sectorByID := make(map[uint64]proof.SectorInfo, len(ss))
	for _, sector := range ss {
		sectorByID[uint64(sector.SectorNumber)] = proof.SectorInfo{
			SectorNumber: sector.SectorNumber,
			SealedCID:    sector.SealedCID,
			SealProof:    sector.SealProof,
		}
	}

	proofSectors := make([]proof.SectorInfo, 0, len(ss))
	if err := sectors.ForEach(func(sectorNo uint64) error {
		if info, found := sectorByID[sectorNo]; found {
			proofSectors = append(proofSectors, info)
		} else {
			proofSectors = append(proofSectors, substitute)
		}
		return nil
	}); err != nil {
		return nil, xerrors.Errorf("iterating partition sector bitmap: %w", err)
	}

	return proofSectors, nil
}

func wdpostEmulator(e util.Emulator, aid abi.ActorID, sInfo []proof.SectorInfo) error {
	var challenge [32]byte
	rand.Read(challenge[:])
	proofs, faulty, skp, err := e.GenerateWindowPoSt(context.Background(), aid, sInfo, challenge[:])
	if err != nil {
		return err
	}

	if len(skp) != 0 {
		log.Error("skip sectors: ", skp)
	}

	if len(faulty) != 0 {
		log.Error("faulty sectors: ", faulty)
	}

	ok, err := ffiwrapper.ProofVerifier.VerifyWindowPoSt(context.TODO(), proof.WindowPoStVerifyInfo{
		Randomness:        challenge[:],
		Proofs:            proofs,
		ChallengedSectors: sInfo,
		Prover:            aid,
	})
	if err != nil || !ok {
		log.Error("window post verification failed")
		return err
	}

	return nil
}
