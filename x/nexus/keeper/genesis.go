package keeper

import (
	"fmt"

	"github.com/axelarnetwork/axelar-core/x/nexus/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the reward module's state from a given genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	k.SetParams(ctx, genState.Params)

	k.setNonce(ctx, genState.Nonce)

	for _, chainState := range genState.ChainStates {
		if _, ok := k.getChainState(ctx, chainState.Chain); ok {
			panic(fmt.Errorf("chain state %s already set", chainState.Chain.Name))
		}

		k.setChainState(ctx, chainState)
	}

	for _, chain := range genState.Chains {
		if _, ok := k.GetChain(ctx, chain.Name); ok {
			panic(fmt.Errorf("chain %s already set", chain.Name))
		}

		k.SetChain(ctx, chain)
		k.RegisterAsset(ctx, chain, chain.NativeAsset)
	}

	for _, linkedAddresses := range genState.LinkedAddresses {
		if _, ok := k.getLinkedAddresses(ctx, linkedAddresses.DepositAddress); ok {
			panic(fmt.Errorf("linked addresses for deposit address %s on chain %s already set", linkedAddresses.DepositAddress.Address, linkedAddresses.DepositAddress.Chain.Name))
		}

		k.setLinkedAddresses(ctx, linkedAddresses)
	}

	transferSeen := make(map[uint64]bool)
	for _, transfer := range genState.Transfers {
		if transferSeen[transfer.ID] {
			panic(fmt.Errorf("transfer %d already set", transfer.ID))
		}

		k.setTransfer(ctx, transfer)
		transferSeen[transfer.ID] = true
	}
}

// ExportGenesis returns the reward module's genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return types.NewGenesisState(
		k.GetParams(ctx),
		k.getNonce(ctx),
		k.GetChains(ctx),
		k.getChainStates(ctx),
		k.getAllLinkedAddresses(ctx),
		k.getTransfers(ctx),
	)
}
