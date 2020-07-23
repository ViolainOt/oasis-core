// Packaged ledger contains the common constants and functions related to Ledger devices
package ledger

import (
	ledger "github.com/oasisprotocol/oasis-core-ledger/temp"
)

const (
	// PathPurpose is set to 44 to indicate use of the BIP-0044 specification.
	PathPurpose uint32 = 44

	// ListingPathCoinType is set to 474, the index registered to Oasis in the SLIP-0044 registry.
	ListingPathCoinType uint32 = 474
	// ListingPathAccount is the account index used to list and connect to Ledger devices by address.
	ListingPathAccount uint32 = 0
	// ListingPathChange indicates an external chain.
	ListingPathChange uint32 = 0
	// ListingPathIndex is the address index used to list and connect to Ledger devices by address.
	ListingPathIndex uint32 = 0
)

var (
	// ListingDerivationPath is the path used to list and connect to devices by address.
	ListingDerivationPath = []uint32{PathPurpose, ListingPathCoinType, ListingPathAccount, ListingPathChange, ListingPathIndex}
)

// Device is a Ledger device.
type Device = ledger.LedgerOasis

// ListDevices will list Ledger devices by address, derived from ListingDerivationPath.
func ListDevices() {
	ledger.ListOasisDevices(ListingDerivationPath)
}

// ConnectToDevice attempts to connect to a Ledger device by address, which is derived by ListingDerivationPath.
func ConnectToDevice(address string) (*Device, error) {
	return ledger.ConnectLedgerOasisApp(address, ListingDerivationPath)
}
