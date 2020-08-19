package beapoch

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/tendermint/tendermint/abci/types"
	"golang.org/x/crypto/sha3"

	"github.com/oasisprotocol/oasis-core/go/consensus/tendermint/api"
	beapochState "github.com/oasisprotocol/oasis-core/go/consensus/tendermint/apps/beapoch/state"
	epochtime "github.com/oasisprotocol/oasis-core/go/epochtime/api"
)

var (
	prodEntropyCtx  = []byte("EkB-tmnt")
	DebugEntropyCtx = []byte("Ekb-Dumm")
	DebugEntropy    = []byte("If you change this, you will fuck up the byzantine tests!!")
)

// GetBeacon derives the actual beacon from the epoch and entropy source.
func GetBeacon(beaconEpoch epochtime.EpochTime, entropyCtx, entropy []byte) []byte {
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], uint64(beaconEpoch))

	h := sha3.New256()
	_, _ = h.Write(entropyCtx)
	_, _ = h.Write(entropy)
	_, _ = h.Write(tmp[:])
	return h.Sum(nil)
}

func (app *beapochApplication) onNewBeacon(ctx *api.Context, beacon []byte) error {
	state := beapochState.NewMutableState(ctx.State())

	if err := state.SetBeacon(ctx, beacon); err != nil {
		ctx.Logger().Error("onNewBeacon: failed to set beacon",
			"err", err,
		)
		return fmt.Errorf("beapoch/beacon: failed to set beacon: %w", err)
	}

	ctx.EmitEvent(api.NewEventBuilder(app.Name()).Attribute(KeyBeacon, beacon))

	return nil
}

func (app *beapochApplication) onEpochChangeBeacon(
	ctx *api.Context,
	state *beapochState.MutableState,
	params *beapochState.ConsensusParameters,
	epoch epochtime.EpochTime,
	req types.RequestBeginBlock,
) error {
	var entropyCtx, entropy []byte

	switch params.Beacon.DebugDeterministic {
	case false:
		entropyCtx = prodEntropyCtx

		height := ctx.BlockHeight()
		if height <= ctx.InitialHeight() {
			// No meaningful previous commit, use the block hash.  This isn't
			// fantastic, but it's only for one epoch.
			ctx.Logger().Debug("onBeaconEpochChange: using block hash as entropy")
			entropy = req.Hash
		} else {
			// Use the previous commit hash as the entropy input, under the theory
			// that the merkle root of all the commits that went into the last
			// block is harder for any single validator to game than the block
			// hash.
			//
			// TODO: This still isn't ideal, and an entirely different beacon
			// entropy source should be written, be it based around SCRAPE,
			// a VDF, naive commit-reveal, or even just calling an SGX enclave.
			ctx.Logger().Debug("onBeaconEpochChange: using commit hash as entropy")
			entropy = req.Header.GetLastCommitHash()
		}
		if len(entropy) == 0 {
			return fmt.Errorf("beapoch/beacon: failed to obtain entropy")
		}
	case true:
		// UNSAFE/DEBUG - Deterministic beacon.
		entropyCtx = DebugEntropyCtx
		// We're setting this random seed so that we have suitable
		// committee schedules for Byzantine E2E scenarios, where we
		// want nodes to be scheduled for only one committee.
		//
		// The permutations derived from this on the first epoch
		// need to have (i) an index that's compute worker only and
		// (ii) an index that's merge worker only. See
		// go/oasis-test-runner/scenario/e2e/byzantine.go for the
		// permutations generated from this seed. These permutations
		// are generated independently of the deterministic node IDs.
		entropy = DebugEntropy
	}

	b := GetBeacon(epoch, entropyCtx, entropy)

	ctx.Logger().Debug("onBeaconEpochChange: generated beacon",
		"epoch", epoch,
		"beacon", hex.EncodeToString(b),
		"block_hash", hex.EncodeToString(entropy),
		"height", ctx.BlockHeight(),
	)

	return app.onNewBeacon(ctx, b)
}
