package util

import (
	"context"
	ffi "github.com/filecoin-project/filecoin-ffi"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper/basicfs"
	"github.com/filecoin-project/lotus/extern/sector-storage/storiface"
	proof2 "github.com/filecoin-project/specs-actors/v2/actors/runtime/proof"
	"github.com/filecoin-project/specs-storage/storage"
	"golang.org/x/xerrors"
	"strings"
)

type Emulator interface {
	GenerateWindowPoSt(ctx context.Context, minerID abi.ActorID, sectorInfo []proof2.SectorInfo, randomness abi.PoStRandomness) ([]proof2.PoStProof, []abi.SectorID, []abi.SectorID, error)
}

type Provider struct {
	ps []ffiwrapper.SectorProvider
}

func NewProvider(sdir string) *Provider {
	var ss = []ffiwrapper.SectorProvider{}

	for _, d := range strings.Split(sdir, ",") {
		bp := &basicfs.Provider{
			Root: d,
		}

		ss = append(ss, bp)
	}

	return &Provider{
		ps: ss,
	}
}

func (e *Provider) GenerateWindowPoSt(ctx context.Context, minerID abi.ActorID, sectorInfo []proof2.SectorInfo, randomness abi.PoStRandomness) ([]proof2.PoStProof, []abi.SectorID, []abi.SectorID, error) {
	randomness[31] &= 0x3f
	privsectors, skipped, done, err := e.pubSectorToPriv(ctx, minerID, sectorInfo, nil, abi.RegisteredSealProof.RegisteredWindowPoStProof)
	if err != nil {
		return nil, nil, nil, xerrors.Errorf("gathering sector info: %w", err)
	}
	defer done()

	if len(privsectors.Values()) == 0 {
		return nil, nil, skipped, nil
	}

	proof, faulty, err := ffi.GenerateWindowPoSt(minerID, privsectors, randomness)

	var faultyIDs []abi.SectorID
	for _, f := range faulty {
		faultyIDs = append(faultyIDs, abi.SectorID{
			Miner:  minerID,
			Number: f,
		})
	}

	return proof, faultyIDs, skipped, err
}

func (e *Provider) pubSectorToPriv(ctx context.Context, mid abi.ActorID, sectorInfo []proof2.SectorInfo, faults []abi.SectorNumber, rpt func(abi.RegisteredSealProof) (abi.RegisteredPoStProof, error)) (ffi.SortedPrivateSectorInfo, []abi.SectorID, func(), error) {
	fmap := map[abi.SectorNumber]struct{}{}
	for _, fault := range faults {
		fmap[fault] = struct{}{}
	}

	var doneFuncs []func()
	done := func() {
		for _, df := range doneFuncs {
			df()
		}
	}

	var skipped []abi.SectorID
	var out []ffi.PrivateSectorInfo
	for _, s := range sectorInfo {
		if _, faulty := fmap[s.SectorNumber]; faulty {
			continue
		}

		sid := storage.SectorRef{
			ID:        abi.SectorID{Miner: mid, Number: s.SectorNumber},
			ProofType: s.SealProof,
		}

		var paths storiface.SectorPaths
		d := func() {}
		var err error
		for _, provider := range e.ps {
			paths, d, err = provider.AcquireSector(ctx, sid, storiface.FTCache|storiface.FTSealed, 0, storiface.PathStorage)
			if err == nil {
				break
			}
		}

		if err != nil {
			skipped = append(skipped, sid.ID)
			continue
		}
		doneFuncs = append(doneFuncs, d)

		postProofType, err := rpt(s.SealProof)
		if err != nil {
			done()
			return ffi.SortedPrivateSectorInfo{}, nil, nil, xerrors.Errorf("acquiring registered PoSt proof from sector info %+v: %w", s, err)
		}

		out = append(out, ffi.PrivateSectorInfo{
			CacheDirPath:     paths.Cache,
			PoStProofType:    postProofType,
			SealedSectorPath: paths.Sealed,
			SectorInfo:       s,
		})
	}

	return ffi.NewSortedPrivateSectorInfo(out...), skipped, done, nil
}
