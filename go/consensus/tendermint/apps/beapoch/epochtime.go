package beapoch

import (
	"fmt"

	"github.com/oasisprotocol/oasis-core/go/common/cbor"
	"github.com/oasisprotocol/oasis-core/go/consensus/tendermint/api"
	beapochState "github.com/oasisprotocol/oasis-core/go/consensus/tendermint/apps/beapoch/state"
	epochtime "github.com/oasisprotocol/oasis-core/go/epochtime/api"
)

func (app *beapochApplication) onBeginBlockEpoch(
	ctx *api.Context,
	state *beapochState.MutableState,
	params *beapochState.ConsensusParameters,
) (bool, epochtime.EpochTime, error) {
	var newEpoch epochtime.EpochTime

	future, err := state.GetFutureEpoch(ctx)
	if err != nil {
		return false, newEpoch, fmt.Errorf("beapoch: failed to get future epoch: %w", err)
	}
	if future == nil {
		return false, newEpoch, nil
	}

	height := ctx.BlockHeight() + 1 // Current height is ctx.BlockHeight() + 1
	switch {
	case future.Height < height:
		// What the fuck, we missed transitioning the epoch?
		ctx.Logger().Error("height mismatch in defered set",
			"height", height,
			"expected_height", future.Height,
		)
		return false, newEpoch, fmt.Errorf("beapoch: height mismatch in defered set")
	case future.Height > height:
		// The epoch transition is scheduled to happen in the grim
		// darkness of the far future.
		return false, newEpoch, nil
	case future.Height == height:
		// Time to fire the scheduled epoch transition.
	}

	ctx.Logger().Info("setting epoch",
		"epoch", future.Epoch,
		"current_height", height,
	)

	if err = state.SetEpoch(ctx, future.Epoch, height); err != nil {
		return false, newEpoch, fmt.Errorf("beapoch: failed to set epoch: %w", err)
	}
	if err = state.ClearFutureEpoch(ctx); err != nil {
		return false, newEpoch, fmt.Errorf("beapoch: failed to clear future epoch: %w", err)
	}
	if !isMockEpochTime(params) {
		// Schedule the next epoch transition.  When this is beacon
		// driven this will need to be done differently, but for now
		// just implement fixed interval based epochs based on height.
		if err = app.scheduleEpochTransitionBlock(ctx, state, params, future.Epoch+1); err != nil {
			return false, newEpoch, err
		}
	}

	ctx.EmitEvent(api.NewEventBuilder(app.Name()).Attribute(KeyEpoch, cbor.Marshal(future.Epoch)))

	return true, future.Epoch, nil
}

func (app *beapochApplication) scheduleEpochTransitionBlock(
	ctx *api.Context,
	state *beapochState.MutableState,
	params *beapochState.ConsensusParameters,
	nextEpoch epochtime.EpochTime,
) error {
	// BUG: This totally ignores the genesis base epoch, because
	// this specific method of setting the epoch doesn't use it,
	// so it will perpetually be 0.
	nextHeight := int64(nextEpoch) * params.EpochTime.Interval

	ctx.Logger().Info("scheduling block interval epoch transition",
		"epoch", nextEpoch,
		"next_height", nextHeight,
		"is_check_only", ctx.IsCheckOnly(),
	)

	if err := state.SetFutureEpoch(ctx, nextEpoch, nextHeight); err != nil {
		return fmt.Errorf("beapoch: failed to set future epoch from interval: %w", err)
	}
	return nil
}

func (app *beapochApplication) doTxSetEpoch(ctx *api.Context, state *beapochState.MutableState, txBody []byte) error {
	now, _, err := state.GetEpoch(ctx)
	if err != nil {
		return err
	}

	var epoch epochtime.EpochTime
	if err := cbor.Unmarshal(txBody, &epoch); err != nil {
		return err
	}

	if epoch <= now {
		ctx.Logger().Error("explicit epoch transition does not advance time",
			"epoch", now,
			"new_epoch", epoch,
		)
		return fmt.Errorf("beapoch: explicit epoch does not advance time")
	}

	height := ctx.BlockHeight() + 1 // Current height is ctx.BlockHeight() + 1

	ctx.Logger().Info("scheduling explicit epoch transition",
		"epoch", epoch,
		"next_height", height+1,
		"is_check_only", ctx.IsCheckOnly(),
	)

	if err := state.SetFutureEpoch(ctx, epoch, height+1); err != nil {
		return err
	}
	return nil
}

func isMockEpochTime(params *beapochState.ConsensusParameters) bool {
	return params.EpochTime.DebugMockBackend
}
