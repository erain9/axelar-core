package types

import (
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

/*
BtcAddress is used as an address format that can be validated and marshalled.
Golang's reflection cannot deal with private fields, so (un)marshalling of btcutil.Address does not work.
Therefore, we need this data type for communication.
*/
type BtcAddress struct {
	Chain         Chain
	EncodedString string
}

// ParseBtcAddress returns a Bitcoin address that can be marshalled and checked for correct format.
func ParseBtcAddress(address string, chain Chain) (BtcAddress, error) {
	addr := BtcAddress{EncodedString: address, Chain: chain}
	if err := addr.Validate(); err != nil {
		return BtcAddress{}, err
	}
	return addr, nil
}

// Validate does a simple format check
func (a BtcAddress) Validate() error {
	if err := a.Chain.Validate(); err != nil {
		return err
	}

	if _, err := btcutil.DecodeAddress(a.EncodedString, a.Chain.Params()); err != nil {
		return sdkerrors.Wrap(err, "could not decode address")
	}
	return nil
}

// String returns the encoded address string
func (a BtcAddress) String() string {
	return a.EncodedString
}

// Convert decodes the address into a btcutil.Address
func (a BtcAddress) Convert() (btcutil.Address, error) {
	return btcutil.DecodeAddress(a.EncodedString, a.Chain.Params())
}

// PkScript creates a script to pay a transaction output to the address
func (a BtcAddress) PkScript() []byte {
	addr, err := a.Convert()
	if err != nil {
		return nil
	}
	if script, err := txscript.PayToAddrScript(addr); err != nil {
		return nil
	} else {
		return script
	}
}
