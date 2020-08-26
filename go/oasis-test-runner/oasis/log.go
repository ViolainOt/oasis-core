package oasis

import (
	"github.com/oasisprotocol/oasis-core/go/common/logging"
	"github.com/oasisprotocol/oasis-core/go/consensus/tendermint/abci"
	tendermint "github.com/oasisprotocol/oasis-core/go/consensus/tendermint/api"
	"github.com/oasisprotocol/oasis-core/go/oasis-test-runner/log"
	roothash "github.com/oasisprotocol/oasis-core/go/roothash/api"
	upgrade "github.com/oasisprotocol/oasis-core/go/upgrade/api"
	workerStorage "github.com/oasisprotocol/oasis-core/go/worker/storage/committee"
)

// LogAssertEvent returns a handler which checks whether a specific log event was
// emitted based on JSON log output.
func LogAssertEvent(event, message string) log.WatcherHandlerFactory {
	return log.AssertJSONContains(logging.LogEvent, event, message)
}

// LogAssertNotEvent returns a handler which checks whether a specific log event
// was not emitted based on JSON log output.
func LogAssertNotEvent(event, message string) log.WatcherHandlerFactory {
	return log.AssertNotJSONContains(logging.LogEvent, event, message)
}

// LogAssertTimeouts returns a handler which checks whether a timeout was
// detected based on JSON log output.
func LogAssertTimeouts() log.WatcherHandlerFactory {
	return LogAssertEvent(roothash.LogEventTimerFired, "timeout not detected")
}

// LogAssertNoTimeouts returns a handler which checks whether a timeout was
// detected based on JSON log output.
func LogAssertNoTimeouts() log.WatcherHandlerFactory {
	return LogAssertNotEvent(roothash.LogEventTimerFired, "timeout detected")
}

// LogAssertNoRoundFailures returns a handler which checks whether a round failure
// was detected based on JSON log output.
func LogAssertNoRoundFailures() log.WatcherHandlerFactory {
	return LogAssertNotEvent(roothash.LogEventRoundFailed, "round failure detected")
}

// LogAssertExecutionDiscrepancyDetected returns a handler which checks whether an
// execution discrepancy was detected based on JSON log output.
func LogAssertExecutionDiscrepancyDetected() log.WatcherHandlerFactory {
	return LogAssertEvent(roothash.LogEventExecutionDiscrepancyDetected, "execution discrepancy not detected")
}

// LogAssertNoExecutionDiscrepancyDetected returns a handler which checks whether an
// execution discrepancy was not detected based on JSON log output.
func LogAssertNoExecutionDiscrepancyDetected() log.WatcherHandlerFactory {
	return LogAssertNotEvent(roothash.LogEventExecutionDiscrepancyDetected, "execution discrepancy detected")
}

// LogAssertPeerExchangeDisabled returns a handler which checks whether a peer
// exchange disabled event was detected based on JSON log output.
func LogAssertPeerExchangeDisabled() log.WatcherHandlerFactory {
	return LogAssertEvent(tendermint.LogEventPeerExchangeDisabled, "peer exchange not disabled")
}

// LogAssertUpgradeStartup returns a handler which checks whether a startup migration
// handler was run based on JSON log output.
func LogAssertUpgradeStartup() log.WatcherHandlerFactory {
	return LogAssertEvent(upgrade.LogEventStartupUpgrade, "expected startup upgrade did not run")
}

// LogAssertUpgradeConsensus returns a handler which checks whether a consensus migration
// handler was run based on JSON log output.
func LogAssertUpgradeConsensus() log.WatcherHandlerFactory {
	return LogAssertEvent(upgrade.LogEventConsensusUpgrade, "expected consensus upgrade did not run")
}

// LogEventABCIPruneDelete returns a handler which checks whether a ABCI pruning delete
// was detected based on JSON log output.
func LogEventABCIPruneDelete() log.WatcherHandlerFactory {
	return LogAssertEvent(abci.LogEventABCIPruneDelete, "expected ABCI pruning to be done")
}

// LogAssertRoothashRoothashReindexing returns a handler which checks whether roothash reindexing was
// run based on JSON log output.
func LogAssertRoothashRoothashReindexing() log.WatcherHandlerFactory {
	return LogAssertEvent(roothash.LogEventHistoryReindexing, "roothash runtime reindexing not detected")
}

// LogAssertCheckpointSyncreturns a handler which checks whether initial storage sync from
// a checkpoint was successful or not.
func LogAssertCheckpointSync() log.WatcherHandlerFactory {
	return LogAssertEvent(workerStorage.LogEventCheckpointSyncSuccess, "checkpoint sync did not succeed")
}
