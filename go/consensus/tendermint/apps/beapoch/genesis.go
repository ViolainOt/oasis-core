package beapoch

import (
	"context"
	"fmt"

	"github.com/tendermint/tendermint/abci/types"

	beacon "github.com/oasisprotocol/oasis-core/go/beacon/api"
	"github.com/oasisprotocol/oasis-core/go/consensus/tendermint/api"
	beapochState "github.com/oasisprotocol/oasis-core/go/consensus/tendermint/apps/beapoch/state"
	epochtime "github.com/oasisprotocol/oasis-core/go/epochtime/api"
	genesis "github.com/oasisprotocol/oasis-core/go/genesis/api"
)

// Genesis is the combined beacon/epochtime genesis.
type Genesis struct {
	Beacon    beacon.Genesis    `json:"beacon"`
	EpochTime epochtime.Genesis `json:"epochtime"`
}

func (app *beapochApplication) InitChain(ctx *api.Context, req types.RequestInitChain, doc *genesis.Document) error {
	params := &beapochState.ConsensusParameters{
		Beacon:    doc.Beacon.Parameters,
		EpochTime: doc.EpochTime.Parameters,
	}

	// Note: If we ever decide that we need a beacon for the 0th epoch
	// (that is *only* for the genesis state), it should be initiailized
	// here.
	//
	// It is not super important for now as the epoch will transition
	// immediately on the first block under normal circumstances.
	state := beapochState.NewMutableState(ctx.State())

	if err := state.SetConsensusParameters(ctx, params); err != nil {
		return fmt.Errorf("failed to set consensus parameters: %w", err)
	}
	if doc.Beacon.Parameters.DebugDeterministic {
		ctx.Logger().Warn("Determistic beacon entropy is NOT FOR PRODUCTION USE")
	}

	// If the backend is configured to use explicitly set epochs, there
	// is nothing further to do.
	if isMockEpochTime(params) {
		return nil
	}

	// Set the initial epoch.
	baseEpoch := doc.EpochTime.Base
	if err := state.SetEpoch(ctx, baseEpoch, ctx.InitialHeight()); err != nil {
		return fmt.Errorf("failed to set initial epoch: %w", err)
	}

	app.doEmitEpochEvent(ctx, baseEpoch)

	// Arm the initial epoch transition.
	if err := app.scheduleEpochTransitionBlock(ctx, state, params, baseEpoch+1); err != nil {
		return fmt.Errorf("failed to schedule epoch transition: %w", err)
	}

	return nil
}

func (bq *beapochQuerier) Genesis(ctx context.Context) (*Genesis, error) {
	params, err := bq.state.ConsensusParameters(ctx)
	if err != nil {
		return nil, err
	}
	epoch, _, err := bq.state.GetEpoch(ctx)
	if err != nil {
		return nil, err
	}

	genesis := &Genesis{
		Beacon: beacon.Genesis{
			Parameters: params.Beacon,
		},
		EpochTime: epochtime.Genesis{
			Parameters: params.EpochTime,
			Base:       epoch,
		},
	}
	return genesis, nil
}
