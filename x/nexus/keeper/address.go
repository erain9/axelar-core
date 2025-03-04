package keeper

import (
	"fmt"

	"github.com/axelarnetwork/axelar-core/utils"
	"github.com/axelarnetwork/axelar-core/x/nexus/exported"
	"github.com/axelarnetwork/axelar-core/x/nexus/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) setLatestDepositAddress(ctx sdk.Context, depositChain string, recipientAddress, depositAddress exported.CrossChainAddress) {
	k.getStore(ctx).Set(latestDepositAddressPrefix.AppendStr(depositChain).Append(utils.LowerCaseKey(recipientAddress.String())), &depositAddress)
}

func (k Keeper) getLatestDepositAddress(ctx sdk.Context, depositChain string, recipientAddress exported.CrossChainAddress) (depositAddress exported.CrossChainAddress, ok bool) {
	return depositAddress, k.getStore(ctx).Get(latestDepositAddressPrefix.AppendStr(depositChain).Append(utils.LowerCaseKey(recipientAddress.String())), &depositAddress)
}

func (k Keeper) setLinkedAddresses(ctx sdk.Context, linkedAddresses types.LinkedAddresses) {
	k.getStore(ctx).Set(linkedAddressesPrefix.Append(utils.LowerCaseKey(linkedAddresses.DepositAddress.String())), &linkedAddresses)
}

func (k Keeper) getLinkedAddresses(ctx sdk.Context, depositAddress exported.CrossChainAddress) (linkedAddresses types.LinkedAddresses, ok bool) {
	return linkedAddresses, k.getStore(ctx).Get(linkedAddressesPrefix.Append(utils.LowerCaseKey(depositAddress.String())), &linkedAddresses)
}

func (k Keeper) getAllLinkedAddresses(ctx sdk.Context) (results []types.LinkedAddresses) {
	iter := k.getStore(ctx).Iterator(linkedAddressesPrefix)
	defer utils.CloseLogError(iter, k.Logger(ctx))

	for ; iter.Valid(); iter.Next() {
		var linkedAddresses types.LinkedAddresses
		iter.UnmarshalValue(&linkedAddresses)

		results = append(results, linkedAddresses)
	}

	return results
}

// LinkAddresses links a sender address to a cross-chain recipient address
func (k Keeper) LinkAddresses(ctx sdk.Context, depositAddress exported.CrossChainAddress, recipientAddress exported.CrossChainAddress) error {
	if validator := k.GetRouter().GetAddressValidator(depositAddress.Chain.Module); validator == nil {
		return fmt.Errorf("unknown module for sender's chain %s", depositAddress.Chain.String())
	} else if err := validator(ctx, depositAddress); err != nil {
		return err
	}

	if validator := k.GetRouter().GetAddressValidator(recipientAddress.Chain.Module); validator == nil {
		return fmt.Errorf("unknown module for recipient's chain %s", recipientAddress.Chain.String())
	} else if err := validator(ctx, recipientAddress); err != nil {
		return err
	}

	linkedAddresses := types.NewLinkedAddresses(depositAddress, recipientAddress)

	k.setLinkedAddresses(ctx, linkedAddresses)
	k.setLatestDepositAddress(ctx, depositAddress.Chain.Name, recipientAddress, depositAddress)

	return nil
}

// GetRecipient retrieves the cross chain recipient associated to the specified sender
func (k Keeper) GetRecipient(ctx sdk.Context, depositAddress exported.CrossChainAddress) (exported.CrossChainAddress, bool) {
	if linkedAddresses, ok := k.getLinkedAddresses(ctx, depositAddress); ok {
		return linkedAddresses.RecipientAddress, true
	}

	return exported.CrossChainAddress{}, false
}
