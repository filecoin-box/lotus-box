package main

import (
	"context"
	"github.com/filecoin-project/lotus/extern/sector-storage/storiface"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/v1api"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper/basicfs"
	sealing "github.com/filecoin-project/lotus/extern/storage-sealing"
	"github.com/filecoin-project/specs-storage/storage"
	addr "github.com/filecoin-project/go-address"
	lcli "github.com/filecoin-project/lotus/cli"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
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
				Name:  "s-ids",
				Usage: "redo sector ids, if there are more than one, separate commas. ps: 1,2",
				Value: "",
			}, &cli.StringFlag{
				Name:  "s-dir",
				Usage: "The directory where the redo sector is stored",
				Value: "",
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

	maddr, err := getActorAddress(context.Background(), minerApi)
	if err != nil {
		return err
	}

	sectorSize, nv, err := getSectorSize(context.Background(), nodeApi, maddr)
	if err != nil {
		return err
	}

	sdir := cctx.String("s-dir")
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

	sids := cctx.String("s-ids")
	log.Infow("will redo sectors", "sids", sids)

	sidStr := strings.Split(sids, ",")
	for _, sStr := range sidStr {
		sid, err := strconv.Atoi(sStr)

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
			return err
		}

		pi, err := sb.AddPiece(context.TODO(), sidRef, nil, abi.PaddedPieceSize(sectorSize).Unpadded(), sealing.NewNullReader(abi.UnpaddedPieceSize(sectorSize)))
		if err != nil {
			return err
		}

		p1Out, err := sb.SealPreCommit1(context.TODO(), sidRef, sInfo.Ticket.Value, []abi.PieceInfo{pi})
		if err != nil {
			return err
		}

		_, err = sb.SealPreCommit2(context.TODO(), sidRef, p1Out)
		if err != nil {
			return err
		}

		err = sb.FinalizeSector(context.TODO(), sidRef, nil)
		if err != nil {
			return err
		}

		log.Infow("redo successful", "sid", sid)
	}

	return nil
}

func getSectorSize(ctx context.Context, nodeApi v1api.FullNode, maddr address.Address) (abi.SectorSize, network.Version, error) {
	head, err := nodeApi.ChainHead(context.Background())
	if err != nil {
		return 0, 0, err
	}

	tsk, err := types.TipSetKeyFromBytes(head.Key().Bytes())
	if err != nil {
		return 0, 0, err
	}

	mi, err := nodeApi.StateMinerInfo(ctx, maddr, tsk)
	if err != nil {
		return 0, 0, err
	}

	ver, err := nodeApi.StateNetworkVersion(ctx, tsk)
	if err != nil {
		return 0, 0, err
	}

	spt, err := miner.PreferredSealProofTypeFromWindowPoStType(ver, mi.WindowPoStProofType)
	if err != nil {
		return 0, 0, err
	}

	sectorSize, err := spt.SectorSize()
	if err != nil {
		return 0, 0, err
	}

	return sectorSize, ver, nil
}

func getActorAddress(ctx context.Context, minerApi api.StorageMiner) (maddr address.Address, err error) {
	maddr, err = minerApi.ActorAddress(ctx)
	if err != nil {
		return maddr, xerrors.Errorf("getting actor address: %w", err)
	}

	return maddr, nil
}
