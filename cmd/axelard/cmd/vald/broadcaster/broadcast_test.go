package broadcaster

import (
	"context"
	"fmt"
	mathRand "math/rand"
	"sync"
	"testing"
	"time"

	rewardtypes "github.com/axelarnetwork/axelar-core/x/reward/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"

	"github.com/axelarnetwork/axelar-core/app"
	mock2 "github.com/axelarnetwork/axelar-core/cmd/axelard/cmd/vald/broadcaster/types/mock"
	"github.com/axelarnetwork/axelar-core/testutils/rand"
	"github.com/axelarnetwork/axelar-core/utils"
	evm "github.com/axelarnetwork/axelar-core/x/evm/types"
	"github.com/axelarnetwork/axelar-core/x/vote/exported"
)

func TestBroadcast(t *testing.T) {
	t.Run("called sequentially", func(t *testing.T) {
		b, ctx := setup()

		iterations := int(rand.I64Between(20, 100))
		for i := 0; i < iterations; i++ {
			msgs := createMsgsWithRandomSigner()

			_, err := b.Broadcast(ctx, msgs...)
			assert.NoError(t, err)
		}

		assert.Len(t, ctx.Client.(*mock2.ClientMock).BroadcastTxSyncCalls(), iterations)
	})

	t.Run("sequence number updated correctly", func(t *testing.T) {
		b, ctx := setup()

		iterations := int(rand.I64Between(200, 1000))
		wg := &sync.WaitGroup{}
		wg.Add(iterations)
		for i := 0; i < iterations; i++ {
			go func(broadcaster *Broadcaster) {
				defer wg.Done()
				msgs := createMsgsWithRandomSigner()
				_, err := broadcaster.Broadcast(ctx, msgs...)
				assert.NoError(t, err)
			}(b)
		}
		wg.Wait()

		foundSeqNo := make([]bool, iterations)
		for _, call := range ctx.Client.(*mock2.ClientMock).BroadcastTxSyncCalls() {
			decodedTx, err := ctx.TxConfig.TxDecoder()(call.Tx)
			assert.NoError(t, err)
			sigs, err := decodedTx.(authsigning.SigVerifiableTx).GetSignaturesV2()
			assert.NoError(t, err)
			for _, sig := range sigs {
				foundSeqNo[sig.Sequence] = true
			}
		}
		assert.Equal(t, iterations, int(b.txFactory.Sequence()))
		assert.NotContains(t, foundSeqNo, false)
	})

	t.Run("sequence number on blockchain trailing behind", func(t *testing.T) {
		accNo := mathRand.Uint64()
		seqNoOnChain := uint64(0)
		b, ctx := setup()
		ctx.AccountRetriever.(*mock2.AccountRetrieverMock).GetAccountNumberSequenceFunc =
			func(client.Context, sdk.AccAddress) (uint64, uint64, error) {
				return accNo, seqNoOnChain, nil
			}
		ctx.Client.(*mock2.ClientMock).BroadcastTxSyncFunc = func(context.Context, types.Tx) (*coretypes.ResultBroadcastTx, error) {
			return &coretypes.ResultBroadcastTx{Code: abci.CodeTypeOK}, nil
		}

		iterations := int(rand.I64Between(200, 1000))
		wg := &sync.WaitGroup{}
		wg.Add(iterations)
		for i := 0; i < iterations; i++ {
			go func(broadcaster *Broadcaster) {
				defer wg.Done()
				msgs := createMsgsWithRandomSigner()
				_, err := broadcaster.Broadcast(ctx, msgs...)
				assert.NoError(t, err)
			}(b)
		}
		wg.Wait()

		foundSeqNo := make([]bool, iterations)
		for _, call := range ctx.Client.(*mock2.ClientMock).BroadcastTxSyncCalls() {
			decodedTx, err := ctx.TxConfig.TxDecoder()(call.Tx)
			assert.NoError(t, err)
			sigs, err := decodedTx.(authsigning.SigVerifiableTx).GetSignaturesV2()
			assert.NoError(t, err)
			for _, sig := range sigs {
				foundSeqNo[sig.Sequence] = true
			}
		}
		assert.Equal(t, iterations, int(b.txFactory.Sequence()))
		assert.NotContains(t, foundSeqNo, false)
	})
}

func TestRetryPipeline_Push(t *testing.T) {
	testCases := []struct {
		label    string
		strategy func(minTimeOut time.Duration) utils.BackOff
	}{
		{"exponential", utils.ExponentialBackOff},
		{"linear", utils.LinearBackOff}}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("failed broadcast with %s backoff", testCase.label), func(t *testing.T) {

			retries := int(rand.I64Between(1, 20))
			backOff := testCase.strategy(20 * time.Nanosecond)
			pipeCap := int(rand.I64Between(10, 100000))
			p := NewPipelineWithRetry(pipeCap, retries, backOff, log.TestingLogger())

			iterations := int(rand.I64Between(5, 30))

			wg := &sync.WaitGroup{}
			wg.Add(iterations)
			for i := 0; i < iterations; i++ {
				go func(i int) {
					defer wg.Done()
					retry := 0
					err := p.Push(func() error {
						retry++
						return fmt.Errorf("retry %d, iteration %d", retry, i)
					}, func(_ error) bool { return true })
					assert.Error(t, err)
				}(i)
			}
			wg.Wait()
		})
	}

	t.Run("called concurrently", func(t *testing.T) {
		retries := int(rand.I64Between(1, 20))
		backOff := utils.LinearBackOff(2 * time.Microsecond)
		p := NewPipelineWithRetry(int(rand.I64Between(10, 100000)), retries, backOff, log.TestingLogger())

		iterations := int(rand.I64Between(20, 100))

		// introducing a data race on purpose to assert that broadcast calls are serialized
		callCounter := 0
		mockFunc := func() error {
			c := callCounter

			// simulate blocking
			timeout := time.Duration(rand.I64Between(0, 20)) * time.Millisecond
			time.Sleep(timeout)

			c++
			callCounter = c
			return nil
		}
		wg := &sync.WaitGroup{}
		wg.Add(iterations)
		for i := 0; i < iterations; i++ {
			go func() {
				defer wg.Done()
				assert.NoError(t, p.Push(mockFunc, func(_ error) bool { return true }))
			}()
		}
		wg.Wait()
		// assert the func has been called the expected amount of times and no data races occurred
		assert.Equal(t, iterations, callCounter)
	})

	t.Run("no retry if retry filter is false", func(t *testing.T) {
		retries := int(rand.I64Between(1, 20))
		backOff := utils.LinearBackOff(2 * time.Microsecond)
		p := NewPipelineWithRetry(int(rand.I64Between(10, 100000)), retries, backOff, log.TestingLogger())

		iterations := int(rand.I64Between(20, 100))

		wg := &sync.WaitGroup{}
		wg.Add(iterations)
		for i := 0; i < iterations; i++ {
			go func(i int) {
				defer wg.Done()
				retry := 0
				err := p.Push(func() error {
					retry++
					return fmt.Errorf("retry %d, iteration %d", retry, i)
				}, func(_ error) bool { return false })
				assert.NoError(t, err)
				assert.True(t, retry == 1)
			}(i)
		}
		wg.Wait()
	})

}

func setup() (*Broadcaster, client.Context) {
	pk, err := cryptocodec.FromTmPubKeyInterface(ed25519.GenPrivKey().PubKey())
	if err != nil {
		panic(err)
	}
	key := &mock2.InfoMock{
		GetPubKeyFunc: func() cryptotypes.PubKey {
			return pk
		},
	}
	ctx := client.Context{
		BroadcastMode: flags.BroadcastSync,
		Client: &mock2.ClientMock{
			BroadcastTxSyncFunc: func(context.Context, types.Tx) (*coretypes.ResultBroadcastTx, error) {
				return &coretypes.ResultBroadcastTx{Code: abci.CodeTypeOK}, nil
			}},
		AccountRetriever: &mock2.AccountRetrieverMock{},
		ChainID:          rand.StrBetween(5, 20),
		TxConfig:         app.MakeEncodingConfig().TxConfig,
		Keyring: &mock2.KeyringMock{
			KeyFunc: func(string) (keyring.Info, error) {
				return key, nil
			},
		},
	}

	fs := pflag.NewFlagSet("test", pflag.PanicOnError)
	txf := tx.NewFactoryCLI(ctx, fs).WithSignMode(txsigning.SignMode_SIGN_MODE_UNSPECIFIED)
	p := NewPipelineWithRetry(100000, 1, func(int) time.Duration {
		return 0
	}, log.TestingLogger())

	b := NewBroadcaster(txf, p, log.TestingLogger())
	return b, ctx
}

func createMsgsWithRandomSigner() []sdk.Msg {
	var msgs []sdk.Msg
	signer := rand.AccAddr()
	for i := int64(0); i < rand.I64Between(1, 20); i++ {

		msg := evm.NewVoteConfirmDepositRequest(
			signer,
			rand.StrBetween(5, 10),
			exported.NewPollKey(evm.ModuleName, rand.StrBetween(5, 100)),
			common.BytesToHash(rand.Bytes(common.HashLength)),
			evm.Address(common.BytesToAddress(rand.Bytes(common.AddressLength))),
			rand.Bools(0.5).Next(),
		)
		msgs = append(msgs, rewardtypes.NewRefundMsgRequest(signer, msg))
	}
	return msgs
}
