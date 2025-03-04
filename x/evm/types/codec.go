package types

import (
	reward "github.com/axelarnetwork/axelar-core/x/reward/exported"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoprototypes "github.com/gogo/protobuf/types"
)

// RegisterLegacyAminoCodec registers concrete types on codec
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&LinkRequest{}, "evm/MsgLink", nil)
	cdc.RegisterConcrete(&VoteConfirmTokenRequest{}, "evm/VoteConfirmToken", nil)
	cdc.RegisterConcrete(&VoteConfirmDepositRequest{}, "evm/VoteConfirmDeposit", nil)
	cdc.RegisterConcrete(&VoteConfirmChainRequest{}, "evm/VoteConfirmChain", nil)
	cdc.RegisterConcrete(&VoteConfirmGatewayDeploymentRequest{}, "evm/VoteConfirmGatewayDeployment", nil)
	cdc.RegisterConcrete(&VoteConfirmTransferKeyRequest{}, "evm/VoteConfirmTransferKey", nil)
	cdc.RegisterConcrete(&ConfirmTokenRequest{}, "evm/ConfirmToken", nil)
	cdc.RegisterConcrete(&ConfirmDepositRequest{}, "evm/ConfirmDeposit", nil)
	cdc.RegisterConcrete(&ConfirmChainRequest{}, "evm/ConfirmChain", nil)
	cdc.RegisterConcrete(&ConfirmGatewayDeploymentRequest{}, "evm/ConfirmGatewayDeployment", nil)
	cdc.RegisterConcrete(&ConfirmTransferKeyRequest{}, "evm/ConfirmTransferKey", nil)
	cdc.RegisterConcrete(&CreatePendingTransfersRequest{}, "evm/CreatePendingTransfers", nil)
	cdc.RegisterConcrete(&CreateDeployTokenRequest{}, "evm/CreateDeployToken", nil)
	cdc.RegisterConcrete(&CreateBurnTokensRequest{}, "evm/CreateBurnTokens", nil)
	cdc.RegisterConcrete(&CreateTransferOwnershipRequest{}, "evm/CreateTransferOwnership", nil)
	cdc.RegisterConcrete(&CreateTransferOperatorshipRequest{}, "evm/CreateTransferOperatorship", nil)
	cdc.RegisterConcrete(&SignCommandsRequest{}, "evm/SignCommands", nil)
	cdc.RegisterConcrete(&AddChainRequest{}, "evm/AddChainRequest", nil)
}

// RegisterInterfaces registers types and interfaces with the given registry
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&LinkRequest{},
		&VoteConfirmTokenRequest{},
		&VoteConfirmDepositRequest{},
		&VoteConfirmChainRequest{},
		&VoteConfirmGatewayDeploymentRequest{},
		&VoteConfirmTransferKeyRequest{},
		&ConfirmTokenRequest{},
		&ConfirmDepositRequest{},
		&ConfirmChainRequest{},
		&ConfirmGatewayDeploymentRequest{},
		&ConfirmTransferKeyRequest{},
		&CreatePendingTransfersRequest{},
		&CreateDeployTokenRequest{},
		&CreateBurnTokensRequest{},
		&CreateTransferOwnershipRequest{},
		&CreateTransferOperatorshipRequest{},
		&SignCommandsRequest{},
		&AddChainRequest{},
	)
	registry.RegisterImplementations((*codec.ProtoMarshaler)(nil),
		&gogoprototypes.BoolValue{},
	)

	registry.RegisterImplementations((*reward.Refundable)(nil),
		&VoteConfirmTokenRequest{},
		&VoteConfirmDepositRequest{},
		&VoteConfirmChainRequest{},
		&VoteConfirmTransferKeyRequest{},
		&VoteConfirmGatewayDeploymentRequest{},
	)
}

var amino = codec.NewLegacyAmino()

// ModuleCdc defines the module codec
var ModuleCdc = codec.NewAminoCodec(amino)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	amino.Seal()
}
