package evm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/axelarnetwork/axelar-core/x/evm/types"
)

// BeginBlocker check for infraction evidence or downtime of validators
// on every begin block
func BeginBlocker(_ sdk.Context, _ abci.RequestBeginBlock, _ types.BaseKeeper) {}

// EndBlocker called every block, process inflation, update validator set.
func EndBlocker(_ sdk.Context, _ abci.RequestEndBlock, _ types.BaseKeeper) []abci.ValidatorUpdate {
	return nil
}
