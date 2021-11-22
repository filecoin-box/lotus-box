package util

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/api/v1api"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

func GetSectorSize(ctx context.Context, nodeApi v1api.FullNode, maddr address.Address) (abi.SectorSize, network.Version, error) {
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

func GetActorAddress(cctx *cli.Context) (maddr address.Address, err error) {
	if cctx.IsSet("actor") {
		maddr, err = address.NewFromString(cctx.String("actor"))
		if err != nil {
			return maddr, err
		}
		return
	}

	minerApi, closer, err := lcli.GetStorageMinerAPI(cctx)
	if err != nil {
		return maddr, err
	}
	defer closer()

	maddr, err = minerApi.ActorAddress(context.Background())
	if err != nil {
		return maddr, xerrors.Errorf("getting actor address: %w", err)
	}

	return maddr, nil
}
