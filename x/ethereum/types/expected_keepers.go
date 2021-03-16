package types

import (
	"crypto/ecdsa"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/axelarnetwork/axelar-core/x/nexus/exported"
	snapshot "github.com/axelarnetwork/axelar-core/x/snapshot/exported"
	tss "github.com/axelarnetwork/axelar-core/x/tss/exported"
	tssTypes "github.com/axelarnetwork/axelar-core/x/tss/types"
	voting "github.com/axelarnetwork/axelar-core/x/vote/exported"
)

//go:generate moq -out ./mock/expected_keepers.go -pkg mock . Voter Signer Nexus Snapshotter

// Voter wraps around the existing exported.Voter interface to adhere to the Cosmos convention of keeping all
// expected keepers from other modules in the expected_keepers.go file
type Voter interface {
	voting.Voter
}

// Nexus provides functionality to manage cross-chain transfers
type Nexus interface {
	LinkAddresses(ctx sdk.Context, sender exported.CrossChainAddress, recipient exported.CrossChainAddress)
	GetRecipient(ctx sdk.Context, sender exported.CrossChainAddress) (exported.CrossChainAddress, bool)
	EnqueueForTransfer(ctx sdk.Context, sender exported.CrossChainAddress, amount sdk.Coin) error
	GetPendingTransfersForChain(ctx sdk.Context, chain exported.Chain) []exported.CrossChainTransfer
	GetArchivedTransfersForChain(ctx sdk.Context, chain exported.Chain) []exported.CrossChainTransfer
	ArchivePendingTransfer(ctx sdk.Context, transfer exported.CrossChainTransfer)
	GetChain(ctx sdk.Context, chain string) (exported.Chain, bool)
	IsAssetRegistered(ctx sdk.Context, chainName, denom string) bool
	RegisterAsset(ctx sdk.Context, chainName, denom string)
}

// Signer provides keygen and signing functionality
type Signer interface {
	StartSign(ctx sdk.Context, voter tssTypes.Voter, keyID string, sigID string, msg []byte, snapshot snapshot.Snapshot) error
	GetCurrentMasterKeyID(ctx sdk.Context, chain exported.Chain) (string, bool)
	GetSig(ctx sdk.Context, sigID string) (tss.Signature, bool)
	GetKey(ctx sdk.Context, keyID string) (ecdsa.PublicKey, bool)
	GetCurrentMasterKey(ctx sdk.Context, chain exported.Chain) (ecdsa.PublicKey, bool)
	GetNextMasterKey(ctx sdk.Context, chain exported.Chain) (ecdsa.PublicKey, bool)
	GetKeyForSigID(ctx sdk.Context, sigID string) (ecdsa.PublicKey, bool)
	GetSnapshotCounterForKeyID(ctx sdk.Context, keyID string) (int64, bool)
}

type Snapshotter interface {
	GetSnapshot(ctx sdk.Context, counter int64) (snapshot.Snapshot, bool)
}
