package keeper

import (
	"context"

	nexus "github.com/axelarnetwork/axelar-core/x/nexus/exported"
	"github.com/axelarnetwork/axelar-core/x/nexus/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ types.QueryServiceServer = Keeper{}

// LatestDepositAddress returns the deposit address for the provided recipient
func (k Keeper) LatestDepositAddress(c context.Context, req *types.LatestDepositAddressRequest) (*types.LatestDepositAddressResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	recipientChain, ok := k.GetChain(ctx, req.RecipientChain)
	if !ok {
		return nil, sdkerrors.Wrapf(types.ErrNexus, "%s is not a registered chain", req.RecipientChain)
	}

	depositChain, ok := k.GetChain(ctx, req.DepositChain)
	if !ok {
		return nil, sdkerrors.Wrapf(types.ErrNexus, "%s is not a registered chain", req.DepositChain)
	}

	recipientAddress := nexus.CrossChainAddress{Chain: recipientChain, Address: req.RecipientAddr}
	depositAddress, ok := k.getLatestDepositAddress(ctx, depositChain.Name, recipientAddress)
	if !ok {
		return nil, sdkerrors.Wrapf(types.ErrNexus, "no deposit address found for recipient %s on chain %s", req.RecipientAddr, req.RecipientChain)
	}

	return &types.LatestDepositAddressResponse{DepositAddr: depositAddress.Address}, nil
}
