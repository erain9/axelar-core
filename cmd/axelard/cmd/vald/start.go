package vald

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tendermint/tendermint/libs/pubsub/query"

	"github.com/axelarnetwork/tm-events/pkg/pubsub"
	"github.com/axelarnetwork/tm-events/pkg/tendermint/client"
	tmEvents "github.com/axelarnetwork/tm-events/pkg/tendermint/events"
	eventTypes "github.com/axelarnetwork/tm-events/pkg/tendermint/types"
	sdkClient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/log"
	rpcclient "github.com/tendermint/tendermint/rpc/client"

	tmTypes "github.com/tendermint/tendermint/types"

	"github.com/axelarnetwork/axelar-core/app"
	"github.com/axelarnetwork/axelar-core/cmd/axelard/cmd/utils"
	"github.com/axelarnetwork/axelar-core/cmd/axelard/cmd/vald/broadcaster"
	broadcasterTypes "github.com/axelarnetwork/axelar-core/cmd/axelard/cmd/vald/broadcaster/types"
	"github.com/axelarnetwork/axelar-core/cmd/axelard/cmd/vald/btc"
	btcRPC "github.com/axelarnetwork/axelar-core/cmd/axelard/cmd/vald/btc/rpc"
	"github.com/axelarnetwork/axelar-core/cmd/axelard/cmd/vald/events"
	"github.com/axelarnetwork/axelar-core/cmd/axelard/cmd/vald/evm"
	evmRPC "github.com/axelarnetwork/axelar-core/cmd/axelard/cmd/vald/evm/rpc"
	"github.com/axelarnetwork/axelar-core/cmd/axelard/cmd/vald/jobs"
	"github.com/axelarnetwork/axelar-core/cmd/axelard/cmd/vald/tss"
	utils2 "github.com/axelarnetwork/axelar-core/utils"
	btcTypes "github.com/axelarnetwork/axelar-core/x/bitcoin/types"
	evmTypes "github.com/axelarnetwork/axelar-core/x/evm/types"
	tssTypes "github.com/axelarnetwork/axelar-core/x/tss/types"
)

// RW grants -rw------- file permissions
const RW = 0600

// RWX grants -rwx------ file permissions
const RWX = 0700

var once sync.Once
var cleanupCommands []func()

// GetValdCommand returns the command to start vald
func GetValdCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "vald-start",
		RunE: func(cmd *cobra.Command, args []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			logger := serverCtx.Logger.With("module", "vald")

			// in case of panic we still want to try and cleanup resources,
			// but we have to make sure it's not called more than once if the program is stopped by an interrupt signal
			defer once.Do(cleanUp)

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				sig := <-sigs
				logger.Info(fmt.Sprintf("captured signal \"%s\"", sig))
				once.Do(cleanUp)
			}()

			node, err := cmd.Flags().GetString(flags.FlagNode)
			if err != nil {
				return err
			}

			cliCtx, err := sdkClient.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cliCtx.WithNodeURI(node)

			// dynamically adjust gas limit by simulating the tx first
			txf := tx.NewFactoryCLI(cliCtx, cmd.Flags()).WithSimulateAndExecute(true)

			hub, err := newHub(node, logger)
			if err != nil {
				return err
			}

			axConf := app.DefaultConfig()
			if err := serverCtx.Viper.Unmarshal(&axConf); err != nil {
				panic(err)
			}

			valAddr := serverCtx.Viper.GetString("validator-addr")
			if valAddr == "" {
				return fmt.Errorf("validator address not set")
			}

			valdHome := filepath.Join(cliCtx.HomeDir, "vald")
			if _, err := os.Stat(valdHome); os.IsNotExist(err) {
				logger.Info(fmt.Sprintf("folder %s does not exist, creating...", valdHome))
				err := os.Mkdir(valdHome, RWX)
				if err != nil {
					return err
				}
			}

			var recoveryJSON []byte
			recoveryFile := serverCtx.Viper.GetString("tofnd-recovery")
			if recoveryFile != "" {
				recoveryJSON, err = ioutil.ReadFile(recoveryFile)
				if err != nil {
					return err
				}
				if len(recoveryJSON) == 0 {
					return fmt.Errorf("JSON file is empty")
				}
			}

			fPath := filepath.Join(valdHome, "state.json")
			stateSource := NewRWFile(fPath)

			logger.Info("start listening to events")
			listen(cliCtx, hub, txf, axConf, valAddr, recoveryJSON, stateSource, logger)
			logger.Info("shutting down")
			return nil
		},
	}
	setPersistentFlags(cmd)
	flags.AddTxFlagsToCmd(cmd)

	values := map[string]string{
		flags.FlagKeyringBackend: "test",
		flags.FlagGasAdjustment:  "2",
		flags.FlagBroadcastMode:  flags.BroadcastSync,
	}
	utils.OverwriteFlagDefaults(cmd, values, true)

	return cmd
}

func cleanUp() {
	for _, cmd := range cleanupCommands {
		cmd()
	}
}

func setPersistentFlags(cmd *cobra.Command) {
	defaultConf := tssTypes.DefaultConfig()
	cmd.PersistentFlags().String("tofnd-host", defaultConf.Host, "host name for tss daemon")
	cmd.PersistentFlags().String("tofnd-port", defaultConf.Port, "port for tss daemon")
	cmd.PersistentFlags().String("tofnd-recovery", "", "json file with recovery request")
	cmd.PersistentFlags().String("validator-addr", "", "the address of the validator operator")
	cmd.PersistentFlags().String(flags.FlagChainID, app.Name, "The network chain ID")
}

func newHub(node string, logger log.Logger) (*tmEvents.Hub, error) {
	c, err := client.NewClient(node, client.DefaultWSEndpoint, logger)
	if err != nil {
		return nil, err
	}

	hub := tmEvents.NewHub(c, logger)
	return &hub, nil
}

func listen(ctx sdkClient.Context, hub *tmEvents.Hub, txf tx.Factory, axelarCfg app.Config, valAddr string, recoveryJSON []byte, stateSource ReadWriter, logger log.Logger) {
	encCfg := app.MakeEncodingConfig()
	cdc := encCfg.Amino
	sender, err := ctx.Keyring.Key(axelarCfg.BroadcastConfig.From)
	if err != nil {
		panic(sdkerrors.Wrap(err, "failed to read broadcaster account info from keyring"))
	}
	ctx = ctx.
		WithFromAddress(sender.GetAddress()).
		WithFromName(sender.GetName())

	bc := createBroadcaster(ctx, txf, axelarCfg, logger)

	stateStore := NewStateStore(stateSource)
	startBlock, err := stateStore.GetState()
	if err != nil {
		logger.Error(err.Error())
		startBlock = 0
	}

	tmClient, err := ctx.GetNode()
	if err != nil {
		panic(err)
	}
	// in order to subscribe to events, the client needs to be running
	if !tmClient.IsRunning() {
		if err := tmClient.Start(); err != nil {
			panic(fmt.Errorf("unable to start client: %v", err))
		}
	}
	eventBus := createEventBus(tmClient, startBlock, logger)

	tssMgr := createTSSMgr(bc, ctx.FromAddress, axelarCfg, logger, valAddr, cdc)
	if recoveryJSON != nil && len(recoveryJSON) > 0 {
		if err = tssMgr.Recover(recoveryJSON); err != nil {
			panic(fmt.Errorf("unable to perform tss recovery: %v", err))
		}
	}

	btcMgr := createBTCMgr(axelarCfg, bc, ctx.FromAddress, logger, cdc)
	evmMgr := createEVMMgr(axelarCfg, bc, ctx.FromAddress, logger, cdc)

	// we have two processes listening to block headers
	blockHeaderForTSS := tmEvents.MustSubscribeNewBlockHeader(hub)
	blockHeaderForStateUpdate := tmEvents.MustSubscribeNewBlockHeader(hub)

	keygenAck := tmEvents.MustSubscribeTx(eventBus, tssTypes.EventTypeAck, tssTypes.ModuleName, tssTypes.AttributeValueKeygen)
	signAck := tmEvents.MustSubscribeTx(eventBus, tssTypes.EventTypeAck, tssTypes.ModuleName, tssTypes.AttributeValueSign)

	queryKeygen := createNewBlockEventQuery(tssTypes.EventTypeKeygen, tssTypes.ModuleName, tssTypes.AttributeValueStart)
	keygenStart, err := tmEvents.Subscribe(eventBus, queryKeygen)
	if err != nil {
		panic(fmt.Errorf("unable to subscribe with keygen event query: %v", err))
	}

	querySign := createNewBlockEventQuery(tssTypes.EventTypeSign, tssTypes.ModuleName, tssTypes.AttributeValueStart)
	signStart, err := tmEvents.Subscribe(eventBus, querySign)
	if err != nil {
		panic(fmt.Errorf("unable to subscribe with sign event query: %v", err))
	}

	keygenMsg := tmEvents.MustSubscribeTx(eventBus, tssTypes.EventTypeKeygen, tssTypes.ModuleName, tssTypes.AttributeValueMsg)
	signMsg := tmEvents.MustSubscribeTx(eventBus, tssTypes.EventTypeSign, tssTypes.ModuleName, tssTypes.AttributeValueMsg)

	btcConf := tmEvents.MustSubscribeTx(eventBus, btcTypes.EventTypeOutpointConfirmation, btcTypes.ModuleName, btcTypes.AttributeValueStart)

	evmNewChain := tmEvents.MustSubscribeTx(hub, evmTypes.EventTypeNewChain, evmTypes.ModuleName, evmTypes.AttributeValueUpdate)
	evmChainConf := tmEvents.MustSubscribeTx(hub, evmTypes.EventTypeChainConfirmation, evmTypes.ModuleName, evmTypes.AttributeValueStart)
	evmDepConf := tmEvents.MustSubscribeTx(eventBus, evmTypes.EventTypeDepositConfirmation, evmTypes.ModuleName, evmTypes.AttributeValueStart)
	evmTokConf := tmEvents.MustSubscribeTx(eventBus, evmTypes.EventTypeTokenConfirmation, evmTypes.ModuleName, evmTypes.AttributeValueStart)
	evmTraConf := tmEvents.MustSubscribeTx(eventBus, evmTypes.EventTypeTransferKeyConfirmation, evmTypes.ModuleName, evmTypes.AttributeValueStart)

	eventCtx, cancelEventCtx := context.WithCancel(context.Background())
	// stop the jobs if process gets interrupted/terminated
	cleanupCommands = append(cleanupCommands, func() {
		logger.Info("stopping listening for blocks...")
		blockHeaderForTSS.Close()
		logger.Info("block listener stopped")
		logger.Info("stop listening for events...")
		cancelEventCtx()
		<-eventBus.Done()
		logger.Info("event listener stopped")
	})

	fetchEvents := func(errChan chan<- error) { errChan <- <-eventBus.FetchEvents(eventCtx) }
	js := []jobs.Job{
		fetchEvents,
		events.Consume(blockHeaderForStateUpdate, func(height int64, _ []sdk.Attribute) error { return stateStore.SetState(height) }),
		events.Consume(blockHeaderForTSS, events.OnlyBlockHeight(tssMgr.ProcessNewBlockHeader)),
		events.Consume(keygenAck, tssMgr.ProcessKeygenAck),
		events.Consume(keygenStart, tssMgr.ProcessKeygenStart),
		events.Consume(keygenMsg, events.OnlyAttributes(tssMgr.ProcessKeygenMsg)),
		events.Consume(signAck, tssMgr.ProcessSignAck),
		events.Consume(signStart, tssMgr.ProcessSignStart),
		events.Consume(signMsg, events.OnlyAttributes(tssMgr.ProcessSignMsg)),
		events.Consume(btcConf, events.OnlyAttributes(btcMgr.ProcessConfirmation)),
		events.Consume(evmNewChain, events.OnlyAttributes(evmMgr.ProcessNewChain)),
		events.Consume(evmChainConf, events.OnlyAttributes(evmMgr.ProcessChainConfirmation)),
		events.Consume(evmDepConf, events.OnlyAttributes(evmMgr.ProcessDepositConfirmation)),
		events.Consume(evmTokConf, events.OnlyAttributes(evmMgr.ProcessTokenConfirmation)),
		events.Consume(evmTraConf, events.OnlyAttributes(evmMgr.ProcessTransferOwnershipConfirmation)),
	}

	// errGroup runs async processes and cancels their context if ANY of them returns an error.
	// Here, we don't want to stop on errors, but simply log it and continue, so errGroup doesn't cut it
	logErr := func(err error) { logger.Error(err.Error()) }
	mgr := jobs.NewMgr(logErr)
	mgr.AddJobs(js...)
	mgr.Wait()
}

func createNewBlockEventQuery(eventType, module, action string) tmEvents.Query {
	return tmEvents.Query{
		TMQuery: query.MustParse(fmt.Sprintf("%s='%s' AND %s.%s='%s'",
			tmTypes.EventTypeKey, tmTypes.EventNewBlock, eventType, sdk.AttributeKeyModule, module)),
		Predicate: func(e eventTypes.Event) bool {
			return e.Type == eventType && e.Module == module && e.Action == action
		},
	}
}

func createEventBus(client rpcclient.Client, startBlock int64, logger log.Logger) *events.EventBus {
	notifier := events.NewBlockNotifier(NewBlockClient(client), logger).StartingAt(startBlock)
	return events.NewEventBus(events.NewBlockSource(client, notifier), pubsub.NewBus, logger)
}

func createBroadcaster(ctx sdkClient.Context, txf tx.Factory, axelarCfg app.Config, logger log.Logger) broadcasterTypes.Broadcaster {
	pipeline := broadcaster.NewPipelineWithRetry(10000, axelarCfg.MaxRetries, utils2.LinearBackOff(axelarCfg.MinTimeout), logger)
	return broadcaster.NewBroadcaster(ctx, txf, pipeline, logger)
}

func createTSSMgr(broadcaster broadcasterTypes.Broadcaster, sender sdk.AccAddress, axelarCfg app.Config, logger log.Logger, valAddr string, cdc *codec.LegacyAmino) *tss.Mgr {
	create := func() (*tss.Mgr, error) {
		gg20client, err := tss.CreateTOFNDClient(axelarCfg.TssConfig.Host, axelarCfg.TssConfig.Port, axelarCfg.TssConfig.DialTimeout, logger)
		if err != nil {
			return nil, err
		}
		tssMgr := tss.NewMgr(gg20client, 2*time.Hour, valAddr, broadcaster, sender, logger, cdc)

		return tssMgr, nil
	}
	mgr, err := create()
	if err != nil {
		panic(sdkerrors.Wrap(err, "failed to create tss manager"))
	}

	return mgr
}

func createBTCMgr(axelarCfg app.Config, b broadcasterTypes.Broadcaster, sender sdk.AccAddress, logger log.Logger, cdc *codec.LegacyAmino) *btc.Mgr {
	rpc, err := btcRPC.NewRPCClient(axelarCfg.BtcConfig, logger)
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}
	// clean up btcRPC connection on process shutdown
	cleanupCommands = append(cleanupCommands, rpc.Shutdown)

	logger.Info("Successfully connected to Bitcoin bridge ")

	btcMgr := btc.NewMgr(rpc, b, sender, logger, cdc)
	return btcMgr
}

func createEVMMgr(axelarCfg app.Config, b broadcasterTypes.Broadcaster, sender sdk.AccAddress, logger log.Logger, cdc *codec.LegacyAmino) *evm.Mgr {
	rpcs := make(map[string]evmRPC.Client)

	for _, evmChainConf := range axelarCfg.EVMConfig {
		if !evmChainConf.WithBridge {
			continue
		}

		if _, found := rpcs[strings.ToLower(evmChainConf.Name)]; found {
			msg := fmt.Errorf("duplicate bridge configuration found for EVM chain %s", evmChainConf.Name)
			logger.Error(msg.Error())
			panic(msg)
		}

		rpc, err := evmRPC.NewClient(evmChainConf.RPCAddr)
		if err != nil {
			logger.Error(err.Error())
			panic(err)
		}
		// clean up evmRPC connection on process shutdown
		cleanupCommands = append(cleanupCommands, rpc.Close)

		rpcs[strings.ToLower(evmChainConf.Name)] = rpc
		logger.Info(fmt.Sprintf("Successfully connected to EVM bridge for chain %s", evmChainConf.Name))
	}

	evmMgr := evm.NewMgr(rpcs, b, sender, logger, cdc)
	return evmMgr
}

// RWFile implements the ReadWriter interface for an underlying file
type RWFile struct {
	path string
}

// NewRWFile returns a new RWFile instance for the given file path
func NewRWFile(path string) RWFile {
	return RWFile{path: path}
}

// ReadAll returns the full content of the file
func (f RWFile) ReadAll() ([]byte, error) { return os.ReadFile(f.path) }

// WriteAll writes the given bytes to a file. Creates a new fille if it does not exist, overwrites the previous content otherwise.
func (f RWFile) WriteAll(bz []byte) error { return os.WriteFile(f.path, bz, RW) }

type blockClient struct{ rpcclient.Client }

func (b blockClient) LatestBlockHeight(ctx context.Context) (int64, error) {
	status, err := b.Status(ctx)
	if err != nil {
		return 0, err
	}
	return status.SyncInfo.LatestBlockHeight, nil
}

// NewBlockClient returns a new events.BlockClient instance
func NewBlockClient(client rpcclient.Client) events.BlockClient {
	return blockClient{client}
}
