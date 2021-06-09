package evm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/axelarnetwork/axelar-core/x/evm/client/cli"
	"github.com/axelarnetwork/axelar-core/x/evm/client/rest"
	"github.com/axelarnetwork/axelar-core/x/evm/keeper"
	"github.com/axelarnetwork/axelar-core/x/evm/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// AppModuleBasic implements module.AppModuleBasic
type AppModuleBasic struct {
}

// Name returns the name of the module
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the types necessary in this module with the given codec
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers the module's interface types
func (AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns the default genesis state
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONMarshaler) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis checks the given genesis state for validity
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONMarshaler, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Validate()
}

// RegisterRESTRoutes registers the REST routes for this module
func (AppModuleBasic) RegisterRESTRoutes(clientCtx client.Context, rtr *mux.Router) {
	rest.RegisterRoutes(clientCtx, rtr)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(client.Context, *runtime.ServeMux) {
}

// GetTxCmd returns all CLI tx commands for this module
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns all CLI query commands for this module
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd(types.QuerierRoute)
}

// AppModule implements module.AppModule
type AppModule struct {
	AppModuleBasic
	logger      log.Logger
	keeper      keeper.Keeper
	voter       types.Voter
	nexus       types.Nexus
	rpcs        map[string]types.RPCClient
	signer      types.Signer
	snapshotter types.Snapshotter
}

// NewAppModule creates a new AppModule object
func NewAppModule(
	k keeper.Keeper,
	voter types.Voter,
	signer types.Signer,
	nexus types.Nexus,
	snapshotter types.Snapshotter,
	rpcs map[string]types.RPCClient,
	logger log.Logger) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		logger:         logger,
		keeper:         k,
		voter:          voter,
		signer:         signer,
		nexus:          nexus,
		snapshotter:    snapshotter,
		rpcs:           rpcs,
	}
}

// RegisterInvariants registers this module's invariants
func (AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {
	// No invariants yet
}

// InitGenesis initializes the module's keeper from the given genesis state
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONMarshaler, gs json.RawMessage) []abci.ValidatorUpdate {
	var genState types.GenesisState
	cdc.MustUnmarshalJSON(gs, &genState)
	InitGenesis(ctx, am.keeper, genState)

	var toRemove []string
	// TODO: this needs to be removed eventually, alongside all usage of RPCs across axelar-core
	for chain, rpc := range am.rpcs {
		id, err := rpc.ChainID(context.Background())
		if err != nil {
			panic(err)
		}

		actualNetwork, found := am.keeper.GetNetworkByID(ctx, chain, id)
		if !found {
			am.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName)).Error(
				fmt.Sprintf(
					"unable to find network name for for chain %s with ID %s",
					chain,
					id.String(),
				))
			toRemove = append(toRemove, chain)
			continue
		}

		network, found := am.keeper.GetNetwork(ctx, chain)
		if !found {
			panic(fmt.Sprintf(
				"unable to find chain %s",
				chain,
			))
		}

		if network != actualNetwork {
			panic(fmt.Sprintf(
				"local %s client not configured correctly: expected network %s, got %s",
				chain,
				network,
				actualNetwork,
			))
		}

	}

	for _, chain := range toRemove {
		delete(am.rpcs, chain)
	}

	return []abci.ValidatorUpdate{}
}

// ExportGenesis exports a genesis state from the module's keeper
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONMarshaler) json.RawMessage {
	genState := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(&genState)
}

// Route returns the module's route
func (am AppModule) Route() sdk.Route {
	return sdk.NewRoute(types.RouterKey, NewHandler(am.keeper, am.voter, am.signer, am.nexus, am.snapshotter))
}

// QuerierRoute returns this module's query route
func (AppModule) QuerierRoute() string {
	return types.QuerierRoute
}

// LegacyQuerierHandler returns a new query handler for this module
func (am AppModule) LegacyQuerierHandler(*codec.LegacyAmino) sdk.Querier {
	return keeper.NewQuerier(am.rpcs, am.keeper, am.signer, am.nexus)
}

// RegisterServices registers a GRPC query service to respond to the
// module-specific GRPC queries.
func (am AppModule) RegisterServices(module.Configurator) {
}

// BeginBlock executes all state transitions this module requires at the beginning of each new block
func (am AppModule) BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock) {
	BeginBlocker(ctx, req, am.keeper)
}

// EndBlock executes all state transitions this module requires at the end of each new block
func (am AppModule) EndBlock(ctx sdk.Context, req abci.RequestEndBlock) []abci.ValidatorUpdate {
	return EndBlocker(ctx, req, am.keeper)
}
