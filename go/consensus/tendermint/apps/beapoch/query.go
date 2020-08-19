package beapoch

import (
	"context"

	abciAPI "github.com/oasisprotocol/oasis-core/go/consensus/tendermint/api"
	beapochState "github.com/oasisprotocol/oasis-core/go/consensus/tendermint/apps/beapoch/state"
	epochtime "github.com/oasisprotocol/oasis-core/go/epochtime/api"
)

// Query is the beacon query interface.
type Query interface {
	Beacon(context.Context) ([]byte, error)
	Epoch(context.Context) (epochtime.EpochTime, int64, error)
	Genesis(context.Context) (*Genesis, error)
}

// QueryFactory is the beacon query factory.
type QueryFactory struct {
	state abciAPI.ApplicationQueryState
}

// QueryAt returns the beacon query interface for a specific height.
func (sf *QueryFactory) QueryAt(ctx context.Context, height int64) (Query, error) {
	state, err := beapochState.NewImmutableState(ctx, sf.state, height)
	if err != nil {
		return nil, err
	}
	return &beapochQuerier{state}, nil
}

type beapochQuerier struct {
	state *beapochState.ImmutableState
}

func (bq *beapochQuerier) Beacon(ctx context.Context) ([]byte, error) {
	return bq.state.Beacon(ctx)
}

func (bq *beapochQuerier) Epoch(ctx context.Context) (epochtime.EpochTime, int64, error) {
	return bq.state.GetEpoch(ctx)
}

func (app *beapochApplication) QueryFactory() interface{} {
	return &QueryFactory{app.state}
}

// NewQueryFactory returns a new QueryFactory backed by the given state
// instance.
func NewQueryFactory(state abciAPI.ApplicationQueryState) *QueryFactory {
	return &QueryFactory{state}
}
