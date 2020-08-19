package beapoch

import (
	"github.com/oasisprotocol/oasis-core/go/consensus/api/transaction"
	"github.com/oasisprotocol/oasis-core/go/consensus/tendermint/api"
	epochtime "github.com/oasisprotocol/oasis-core/go/epochtime/api"
)

const (
	// AppID is the unique application identifier.
	AppID uint8 = 0x08

	// AppName is the ABCI application name.
	// Run before all other applications.
	AppName string = "000_beapoch"
)

var (
	// EventType is the ABCI event type for beacon/epoch events.
	EventType = api.EventTypeForApp(AppName)

	// QueryApp is the query for filtering events procecessed by the
	// beapochtime application.
	QueryApp = api.QueryForApp(AppName)

	// KeyEpoch is an ABCI event attribute for specifying the set epoch.
	KeyEpoch = []byte("epoch")

	// KeyBeacon is the ABCI event attribute key for the new
	// beacons (value is a CBOR serialized beacon.GenerateEvent).
	KeyBeacon = []byte("beacon")

	// MethodSetEpoch is the method name for setting epochs.
	MethodSetEpoch = transaction.NewMethodName(AppName, "SetEpoch", epochtime.EpochTime(0))

	// Methods is a list of all methods supported by the epochtime mock application.
	Methods = []transaction.MethodName{
		MethodSetEpoch,
	}
)
