// Package beapoch implements the combined beacon and epochtime
// application.
package beapoch

import (
	"fmt"

	"github.com/tendermint/tendermint/abci/types"

	"github.com/oasisprotocol/oasis-core/go/consensus/api/transaction"
	"github.com/oasisprotocol/oasis-core/go/consensus/tendermint/api"
	beapochState "github.com/oasisprotocol/oasis-core/go/consensus/tendermint/apps/beapoch/state"
)

var _ api.Application = (*beapochApplication)(nil)

type beapochApplication struct {
	state api.ApplicationState
}

func (app *beapochApplication) Name() string {
	return AppName
}

func (app *beapochApplication) ID() uint8 {
	return AppID
}

func (app *beapochApplication) Methods() []transaction.MethodName {
	return Methods
}

func (app *beapochApplication) Blessed() bool {
	return false
}

func (app *beapochApplication) Dependencies() []string {
	return nil
}

func (app *beapochApplication) OnRegister(state api.ApplicationState) {
	app.state = state
}

func (app *beapochApplication) OnCleanup() {
}

func (app *beapochApplication) BeginBlock(ctx *api.Context, req types.RequestBeginBlock) error {
	state := beapochState.NewMutableState(ctx.State())

	params, err := state.ConsensusParameters(ctx)
	if err != nil {
		return fmt.Errorf("beapoch: failed to query consensus parameters: %w", err)
	}

	epochChanged, newEpoch, err := app.onBeginBlockEpoch(ctx, state, params)
	if err != nil {
		return fmt.Errorf("beapoch: failed to handle epoch BeginBlock: %w", err)
	}
	if !epochChanged {
		return nil
	}

	// The epoch changed, deal with the beacon specific operations.
	if err = app.onEpochChangeBeacon(ctx, state, params, newEpoch, req); err != nil {
		return fmt.Errorf("beapoch: failed to handle beacon BeginBlock: %w", err)
	}

	return nil
}

func (app *beapochApplication) ExecuteTx(ctx *api.Context, tx *transaction.Transaction) error {
	state := beapochState.NewMutableState(ctx.State())

	params, err := state.ConsensusParameters(ctx)
	if err != nil {
		return fmt.Errorf("beapoch: failed to query consensus parameters: %w", err)
	}

	switch tx.Method {
	case MethodSetEpoch:
		if !isMockEpochTime(params) {
			return fmt.Errorf("beapoch: method '%s' is disabled via consensus", MethodSetEpoch)
		}
		return app.doTxSetEpoch(ctx, state, tx.Body)
	default:
		return fmt.Errorf("beapoch: invalid method: %s", tx.Method)
	}
}

func (app *beapochApplication) ForeignExecuteTx(ctx *api.Context, other api.Application, tx *transaction.Transaction) error {
	return nil
}

func (app *beapochApplication) EndBlock(ctx *api.Context, req types.RequestEndBlock) (types.ResponseEndBlock, error) {
	return types.ResponseEndBlock{}, nil
}

// New constructs a new beapochtime application instance.
func New() api.Application {
	return &beapochApplication{}
}
