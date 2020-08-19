// Package beapoch implements the tendermint backed beacon and epochtime
// backends.
package beapoch

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/eapache/channels"
	tmabcitypes "github.com/tendermint/tendermint/abci/types"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	tmtypes "github.com/tendermint/tendermint/types"

	beaconAPI "github.com/oasisprotocol/oasis-core/go/beacon/api"
	"github.com/oasisprotocol/oasis-core/go/common/cbor"
	memorySigner "github.com/oasisprotocol/oasis-core/go/common/crypto/signature/signers/memory"
	"github.com/oasisprotocol/oasis-core/go/common/logging"
	"github.com/oasisprotocol/oasis-core/go/common/pubsub"
	consensus "github.com/oasisprotocol/oasis-core/go/consensus/api"
	"github.com/oasisprotocol/oasis-core/go/consensus/api/transaction"
	tmAPI "github.com/oasisprotocol/oasis-core/go/consensus/tendermint/api"
	app "github.com/oasisprotocol/oasis-core/go/consensus/tendermint/apps/beapoch"
	epochtimeAPI "github.com/oasisprotocol/oasis-core/go/epochtime/api"
)

var testSigner = memorySigner.NewTestSigner("oasis-core epochtime mock key seed")

// BeaconServiceClient is the beacon service client interface.
type BeaconServiceClient interface {
	beaconAPI.Backend
	tmAPI.ServiceClient
}

// EpochTimeServiceClient is the epochtime service client interface.
type EpochTimeServiceClient interface {
	epochtimeAPI.Backend
	tmAPI.ServiceClient
}

type beaconServiceClient struct {
	*serviceClient
}

func (sc *beaconServiceClient) GetBeacon(ctx context.Context, height int64) ([]byte, error) {
	q, err := sc.querier.QueryAt(ctx, height)
	if err != nil {
		return nil, err
	}

	return q.Beacon(ctx)
}

func (sc *beaconServiceClient) StateToGenesis(ctx context.Context, height int64) (*beaconAPI.Genesis, error) {
	g, err := sc.stateToGenesis(ctx, height)
	if err != nil {
		return nil, err
	}

	return &g.Beacon, nil
}

func (sc *beaconServiceClient) ServiceDescriptor() tmAPI.ServiceDescriptor {
	// Only start one fullService.serviceClientWorker.
	//
	// WARNING: If/when the beaconServiceClient needs to hook DeliverEvent
	// or DeliverBlock, it will not work, so this will require minor
	// refactoring.
	return nil
}

type epochtimeServiceClient struct {
	sync.RWMutex
	*serviceClient

	backend  tmAPI.Backend
	notifier *pubsub.Broker

	lastNotified  epochtimeAPI.EpochTime
	epoch         epochtimeAPI.EpochTime
	currentBlock  int64
	initialNotify bool

	baseEpoch epochtimeAPI.EpochTime
	baseBlock int64
}

func (sc *epochtimeServiceClient) GetBaseEpoch(ctx context.Context) (epochtimeAPI.EpochTime, error) {
	return sc.baseEpoch, nil
}

func (sc *epochtimeServiceClient) GetEpoch(ctx context.Context, height int64) (epochtimeAPI.EpochTime, error) {
	q, err := sc.querier.QueryAt(ctx, height)
	if err != nil {
		return epochtimeAPI.EpochInvalid, err
	}

	epoch, _, err := q.Epoch(ctx)
	return epoch, err
}

func (sc *epochtimeServiceClient) GetEpochBlock(ctx context.Context, epoch epochtimeAPI.EpochTime) (int64, error) {
	now, currentBlk := sc.currentEpochBlock()
	if epoch == now {
		return currentBlk, nil
	}

	// The epoch can't be earlier than the initial starting epoch.
	switch {
	case epoch < sc.baseEpoch:
		return -1, fmt.Errorf("epoch predates base (base: %d requested: %d)", sc.baseEpoch, epoch)
	case epoch == sc.baseEpoch:
		return sc.baseBlock, nil
	}

	// Find historic epoch.
	//
	// TODO: This is really really inefficient, and should be optimized,
	// maybe a cache of the last few epochs, or a binary search.
	height := consensus.HeightLatest
	for {
		q, err := sc.querier.QueryAt(ctx, height)
		if err != nil {
			return -1, fmt.Errorf("failed to query epoch: %w", err)
		}

		var pastEpoch epochtimeAPI.EpochTime
		pastEpoch, height, err = q.Epoch(ctx)
		if err != nil {
			return -1, fmt.Errorf("failed to query epoch: %w", err)
		}

		if epoch == pastEpoch {
			return height, nil
		}

		height--

		// The initial height can be > 1, but presumably this does not
		// matter, since we validate that epoch > sc.baseEpoch.
		if pastEpoch == 0 || height <= 1 {
			return -1, fmt.Errorf("failed to find historic epoch (minimum: %d requested: %d)", pastEpoch, epoch)
		}
	}
}

func (sc *epochtimeServiceClient) WatchEpochs() (<-chan epochtimeAPI.EpochTime, *pubsub.Subscription) {
	typedCh := make(chan epochtimeAPI.EpochTime)
	sub := sc.notifier.Subscribe()
	sub.Unwrap(typedCh)

	return typedCh, sub
}

func (sc *epochtimeServiceClient) WatchLatestEpoch() (<-chan epochtimeAPI.EpochTime, *pubsub.Subscription) {
	typedCh := make(chan epochtimeAPI.EpochTime)
	sub := sc.notifier.SubscribeBuffered(1)
	sub.Unwrap(typedCh)

	return typedCh, sub
}

func (sc *epochtimeServiceClient) StateToGenesis(ctx context.Context, height int64) (*epochtimeAPI.Genesis, error) {
	g, err := sc.stateToGenesis(ctx, height)
	if err != nil {
		return nil, err
	}

	g.EpochTime.Base, err = sc.GetEpoch(ctx, height)
	if err != nil {
		return nil, err
	}

	return &g.EpochTime, nil
}

func (sc *epochtimeServiceClient) SetEpoch(ctx context.Context, epoch epochtimeAPI.EpochTime) error {
	ch, sub := sc.WatchEpochs()
	defer sub.Close()

	tx := transaction.NewTransaction(0, nil, app.MethodSetEpoch, epoch)
	if err := consensus.SignAndSubmitTx(ctx, sc.backend, testSigner, tx); err != nil {
		return fmt.Errorf("epochtime: set epoch failed: %w", err)
	}

	for {
		select {
		case newEpoch, ok := <-ch:
			if !ok {
				return context.Canceled
			}
			if newEpoch == epoch {
				return nil
			}
		case <-ctx.Done():
			return context.Canceled
		}
	}
}

func (sc *epochtimeServiceClient) ServiceDescriptor() tmAPI.ServiceDescriptor {
	return tmAPI.NewStaticServiceDescriptor("beapoch", app.EventType, []tmpubsub.Query{app.QueryApp})
}

func (sc *epochtimeServiceClient) DeliverBlock(ctx context.Context, height int64) error {
	if !sc.initialNotify {
		q, err := sc.querier.QueryAt(ctx, height)
		if err != nil {
			return fmt.Errorf("epochtime: failed to query state: %w", err)
		}

		var epoch epochtimeAPI.EpochTime
		epoch, height, err = q.Epoch(ctx)
		if err != nil {
			return fmt.Errorf("epochtime: failed to query epoch: %w", err)
		}

		if sc.updateCached(height, epoch) {
			sc.notifier.Broadcast(epoch)
		}
		sc.initialNotify = true
	}
	return nil
}

func (sc *epochtimeServiceClient) DeliverEvent(ctx context.Context, height int64, tx tmtypes.Tx, ev *tmabcitypes.Event) error {
	for _, pair := range ev.GetAttributes() {
		if bytes.Equal(pair.GetKey(), app.KeyEpoch) {
			var epoch epochtimeAPI.EpochTime
			if err := cbor.Unmarshal(pair.GetValue(), &epoch); err != nil {
				sc.logger.Error("epochtime: malformed epoch",
					"err", err,
				)
				continue
			}

			if sc.updateCached(height, epoch) {
				sc.notifier.Broadcast(sc.epoch)
			}
		}
	}
	return nil
}

func (sc *epochtimeServiceClient) updateCached(height int64, epoch epochtimeAPI.EpochTime) bool {
	sc.Lock()
	defer sc.Unlock()

	sc.epoch = epoch
	sc.currentBlock = height

	if sc.lastNotified != epoch {
		sc.logger.Debug("epoch transition",
			"prev_epoch", sc.lastNotified,
			"epoch", epoch,
			"height", height,
		)
		sc.lastNotified = sc.epoch
		return true
	}
	return false
}

func (sc *epochtimeServiceClient) currentEpochBlock() (epochtimeAPI.EpochTime, int64) {
	sc.RLock()
	defer sc.RUnlock()

	return sc.epoch, sc.currentBlock
}

type serviceClient struct {
	tmAPI.BaseServiceClient

	logger  *logging.Logger
	querier *app.QueryFactory
}

func (sc *serviceClient) stateToGenesis(ctx context.Context, height int64) (*app.Genesis, error) {
	q, err := sc.querier.QueryAt(ctx, height)
	if err != nil {
		return nil, err
	}

	return q.Genesis(ctx)
}

// New constructs new tendermint backed beacon and epochtime Backend instances.
func New(ctx context.Context, backend tmAPI.Backend) (BeaconServiceClient, EpochTimeServiceClient, error) {
	// Initialize and register the tendermint service component.
	a := app.New()
	if err := backend.RegisterApplication(a); err != nil {
		return nil, nil, err
	}

	sc := &serviceClient{
		logger:  logging.GetLogger("beapoch/tendermint"),
		querier: a.QueryFactory().(*app.QueryFactory),
	}

	bsc := &beaconServiceClient{sc}
	esc := &epochtimeServiceClient{
		serviceClient: sc,
		backend:       backend,
	}
	esc.notifier = pubsub.NewBrokerEx(func(ch channels.Channel) {
		esc.RLock()
		defer esc.RUnlock()

		if esc.lastNotified == esc.epoch {
			ch.In() <- esc.epoch
		}
	})

	genDoc, err := backend.GetGenesisDocument(ctx)
	if err != nil {
		return nil, nil, err
	}

	esc.baseEpoch = genDoc.EpochTime.Base
	esc.baseBlock = genDoc.Height

	return bsc, esc, nil
}
