package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/axelarnetwork/axelar-core/utils"
	"github.com/axelarnetwork/axelar-core/x/nexus/exported"
	"github.com/axelarnetwork/axelar-core/x/nexus/types"
)

func (k Keeper) getChainStates(ctx sdk.Context) (chainStates []types.ChainState) {
	iter := k.getStore(ctx).Iterator(chainStatePrefix)
	defer utils.CloseLogError(iter, k.Logger(ctx))

	for ; iter.Valid(); iter.Next() {
		var chainState types.ChainState
		iter.UnmarshalValue(&chainState)

		chainStates = append(chainStates, chainState)
	}

	return chainStates
}

func (k Keeper) setChainState(ctx sdk.Context, chainState types.ChainState) {
	k.getStore(ctx).Set(chainStatePrefix.Append(utils.LowerCaseKey(chainState.Chain.Name)), &chainState)
}

func (k Keeper) getChainState(ctx sdk.Context, chain exported.Chain) (chainState types.ChainState, ok bool) {
	return chainState, k.getStore(ctx).Get(chainStatePrefix.Append(utils.LowerCaseKey(chain.Name)), &chainState)
}

func (k Keeper) subtractFromChainTotal(ctx sdk.Context, chain exported.Chain, coin sdk.Coin) {
	chainState, _ := k.getChainState(ctx, chain)
	chainState.Chain = chain
	chainState.Total = chainState.Total.Sub(sdk.NewCoins(coin))

	k.setChainState(ctx, chainState)
}

func (k Keeper) getChainTotal(ctx sdk.Context, chain exported.Chain, denom string) sdk.Coin {
	chainState, ok := k.getChainState(ctx, chain)
	if !ok {
		return sdk.NewCoin(denom, sdk.ZeroInt())
	}

	for _, coin := range chainState.Total {
		if coin.Denom == denom {
			return coin
		}
	}

	return sdk.NewCoin(denom, sdk.ZeroInt())
}

// RegisterAsset indicates that the specified asset is supported by the given chain
func (k Keeper) RegisterAsset(ctx sdk.Context, chain exported.Chain, denom string) {
	chainState, _ := k.getChainState(ctx, chain)
	chainState.Chain = chain
	chainState.AddAsset(denom)

	k.setChainState(ctx, chainState)
}

// IsAssetRegistered returns true if the specified asset is supported by the given chain
func (k Keeper) IsAssetRegistered(ctx sdk.Context, chain exported.Chain, denom string) bool {
	chainState, ok := k.getChainState(ctx, chain)
	if !ok {
		return false
	}

	return chainState.HasAsset(denom)
}

// AddToChainTotal add balance for an asset for a chain
func (k Keeper) AddToChainTotal(ctx sdk.Context, chain exported.Chain, coin sdk.Coin) {
	chainState, _ := k.getChainState(ctx, chain)
	chainState.Chain = chain
	chainState.Total = chainState.Total.Add(coin)

	k.setChainState(ctx, chainState)
}

// ActivateChain activates the given chain
func (k Keeper) ActivateChain(ctx sdk.Context, chain exported.Chain) {
	chainState, _ := k.getChainState(ctx, chain)
	chainState.Chain = chain
	chainState.Activated = true

	k.setChainState(ctx, chainState)
}

// IsChainActivated returns true if the given chain is activated; false otherwise
func (k Keeper) IsChainActivated(ctx sdk.Context, chain exported.Chain) bool {
	chainState, ok := k.getChainState(ctx, chain)
	if !ok {
		return false
	}

	return chainState.Activated
}

// GetChainMaintainers returns the maintainers of the given chain
func (k Keeper) GetChainMaintainers(ctx sdk.Context, chain exported.Chain) []sdk.ValAddress {
	chainState, ok := k.getChainState(ctx, chain)
	if !ok {
		return []sdk.ValAddress{}
	}

	return chainState.Maintainers
}

// IsChainMaintainer returns true if the given address is one of the given chain's maintainers; false otherwise
func (k Keeper) IsChainMaintainer(ctx sdk.Context, chain exported.Chain, maintainer sdk.ValAddress) bool {
	chainState, ok := k.getChainState(ctx, chain)
	if !ok {
		return false
	}

	return chainState.HasMaintainer(maintainer)
}

// AddChainMaintainer adds the given address to be one of the given chain's maintainers
func (k Keeper) AddChainMaintainer(ctx sdk.Context, chain exported.Chain, maintainer sdk.ValAddress) error {
	chainState, _ := k.getChainState(ctx, chain)
	chainState.Chain = chain

	if err := chainState.AddMaintainer(maintainer); err != nil {
		return err
	}

	k.setChainState(ctx, chainState)

	return nil
}

// RemoveChainMaintainer removes the given address from the given chain's maintainers
func (k Keeper) RemoveChainMaintainer(ctx sdk.Context, chain exported.Chain, maintainer sdk.ValAddress) error {
	chainState, _ := k.getChainState(ctx, chain)
	chainState.Chain = chain

	if err := chainState.RemoveMaintainer(maintainer); err != nil {
		return err
	}

	k.setChainState(ctx, chainState)

	return nil
}

// GetChains retrieves the specification for all supported blockchains
func (k Keeper) GetChains(ctx sdk.Context) (chains []exported.Chain) {
	iter := k.getStore(ctx).Iterator(chainPrefix)
	defer utils.CloseLogError(iter, k.Logger(ctx))

	for ; iter.Valid(); iter.Next() {
		var chain exported.Chain
		iter.UnmarshalValue(&chain)

		chains = append(chains, chain)
	}

	return chains
}

// GetChain retrieves the specification for a supported blockchain
func (k Keeper) GetChain(ctx sdk.Context, chainName string) (chain exported.Chain, ok bool) {
	return chain, k.getStore(ctx).Get(chainPrefix.Append(utils.LowerCaseKey(chainName)), &chain)
}

// SetChain sets the specification for a supported chain
func (k Keeper) SetChain(ctx sdk.Context, chain exported.Chain) {
	k.getStore(ctx).Set(chainPrefix.Append(utils.LowerCaseKey(chain.Name)), &chain)
}
