package main

import (
	"context"
	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper/basicfs"
	"github.com/filecoin-project/lotus/extern/sector-storage/storiface"
	sealing "github.com/filecoin-project/lotus/extern/storage-sealing"
	"github.com/filecoin-project/specs-storage/storage"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"lotus-box/util"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var log = logging.Logger("lotus-redo")

func main() {
	_ = logging.SetLogLevel("*", "DEBUG")

	app := &cli.App{
		Name:    "lotus-redo",
		Usage:   "lotus redo sector",
		Version: "v0.1",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "sids",
				Usage: "redo sector ids, if there are more than one, separate commas. ps: 1,2",
				Value: "",
			}, &cli.StringFlag{
				Name:  "sdir",
				Usage: "The directory where the redo sector is stored",
				Value: "",
			},
			&cli.IntFlag{
				Name:  "parallel",
				Usage: "num run in parallel",
				Value: 1,
			},
		},
		EnableBashCompletion: true,
		Action: func(cctx *cli.Context) error {
			return redo(cctx)
		},
	}

	app.Setup()
	lcli.RunApp(app)
}

func redo(cctx *cli.Context) error {
	minerApi, closer, err := lcli.GetStorageMinerAPI(cctx)
	if err != nil {
		return err
	}
	defer closer()

	nodeApi, closer, err := lcli.GetFullNodeAPIV1(cctx)
	if err != nil {
		return err
	}
	defer closer()

	maddr, err := util.GetActorAddress(cctx)
	if err != nil {
		return err
	}


	sectorSize, nv, err := util.GetSectorSize(context.Background(), nodeApi, maddr)
	if err != nil {
		return err
	}

	sdir := cctx.String("sdir")
	if sdir == "" {
		home, _ := os.LookupEnv("HOME")
		if home == "" {
			return xerrors.New("No storage directory is set and get $HOME fail.")
		}
		sdir = filepath.Join(home, "redo")
		log.Infow("No storage directory is set, the default directory will be used", "path", sdir)
	}

	for _, t := range storiface.PathTypes {
		if err := os.MkdirAll(filepath.Join(sdir, t.String()), 0755); err != nil {
			return err
		}
	}

	sbfs := &basicfs.Provider{
		Root: sdir,
	}

	sb, err := ffiwrapper.New(sbfs)
	if err != nil {
		return err
	}

	amid, err := addr.IDFromAddress(maddr)
	if err != nil {
		return err
	}
	actor := abi.ActorID(amid)

	spt, err := miner.SealProofTypeFromSectorSize(sectorSize, nv)
	if err != nil {
		return err
	}

	p1Limit := cctx.Int("parallel")
	if p1Limit <= 0 {
		return xerrors.New("parallel must be greater than 0")
	}
	log.Infof("preCommit1 parallel number: %d", p1Limit)
	preCommit1Sema := make(chan struct{}, p1Limit)

	p2Limit := 1
	preCommit2Sema := make(chan struct{}, p2Limit)

	p1Start := func() {
		preCommit1Sema <- struct{}{}
	}

	p1Done := func() {
		<-preCommit1Sema
	}

	p2Start := func() {
		preCommit2Sema <- struct{}{}
	}

	p2Done := func() {
		<-preCommit2Sema
	}

	sids := cctx.String("sids")
	sidStr := strings.Split(sids, ",")
	log.Infow("will redo sectors", "sids", sids)

	var parallelNum sync.WaitGroup
	for _, sStr := range sidStr {
		sid, err := strconv.Atoi(sStr)
		if err != nil {
			log.Errorw("sid parse fail", "err", err)
			continue
		}

		p1Start()
		parallelNum.Add(1)
		go func(sid int) {
			defer parallelNum.Done()
			log.Infow("redo sector", "sid", sid)

			sidRef := storage.SectorRef{
				ID: abi.SectorID{
					Miner:  abi.ActorID(actor),
					Number: abi.SectorNumber(sid),
				},
				ProofType: spt,
			}

			sInfo, err := minerApi.SectorsStatus(context.TODO(), abi.SectorNumber(sid), false)
			if err != nil {
				log.Errorw("API error: SectorsStatus", "err", err)
				p1Done()
				return
			}

			pi, err := sb.AddPiece(context.TODO(), sidRef, nil, abi.PaddedPieceSize(sectorSize).Unpadded(), sealing.NewNullReader(abi.UnpaddedPieceSize(sectorSize)))
			if err != nil {
				log.Errorw("AddPiece error", "err", err)
				p1Done()
				return
			}

			p1Out, err := sb.SealPreCommit1(context.TODO(), sidRef, sInfo.Ticket.Value, []abi.PieceInfo{pi})
			if err != nil {
				log.Errorw("SealPreCommit1 error", "err", err)
				p1Done()
				return
			}

			p1Done()

			p2Start()
			_, err = sb.SealPreCommit2(context.TODO(), sidRef, p1Out)
			if err != nil {
				log.Errorw("SealPreCommit2 error", "err", err)
				p2Done()
				return
			}
			p2Done()

			err = sb.FinalizeSector(context.TODO(), sidRef, nil)
			if err != nil {
				log.Errorw("FinalizeSector error", "err", err)
				return
			}

			log.Infow("redo successful", "sid", sid)
		}(sid)
	}

	parallelNum.Wait()
	return nil
}
