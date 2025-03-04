package evm

import (
	"context"
	"fmt"
	"math/big"
	mathRand "math/rand"
	"strconv"
	"testing"

	rewardtypes "github.com/axelarnetwork/axelar-core/x/reward/types"
	tmEvents "github.com/axelarnetwork/tm-events/events"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	geth "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/axelarnetwork/axelar-core/app"
	mock2 "github.com/axelarnetwork/axelar-core/cmd/axelard/cmd/vald/broadcaster/types/mock"
	evmRpc "github.com/axelarnetwork/axelar-core/cmd/axelard/cmd/vald/evm/rpc"
	"github.com/axelarnetwork/axelar-core/cmd/axelard/cmd/vald/evm/rpc/mock"
	"github.com/axelarnetwork/axelar-core/testutils"
	"github.com/axelarnetwork/axelar-core/testutils/rand"
	evmTypes "github.com/axelarnetwork/axelar-core/x/evm/types"
	tss "github.com/axelarnetwork/axelar-core/x/tss/exported"
	"github.com/axelarnetwork/axelar-core/x/vote/exported"
)

func TestDecodeTokenDeployEvent_CorrectData(t *testing.T) {
	axelarGateway := common.HexToAddress("0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA")
	tokenDeploySig := ERC20TokenDeploymentSig
	expectedAddr := common.HexToAddress("0xE7481ECB61F9C84b91C03414F3D5d48E5436045D")
	expectedSymbol := "XPTO"
	data := common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000e7481ecb61f9c84b91c03414f3d5d48e5436045d00000000000000000000000000000000000000000000000000000000000000045850544f00000000000000000000000000000000000000000000000000000000")

	l := &geth.Log{Address: axelarGateway, Data: data, Topics: []common.Hash{tokenDeploySig}}

	symbol, tokenAddr, err := decodeERC20TokenDeploymentEvent(l)
	assert.NoError(t, err)
	assert.Equal(t, expectedSymbol, symbol)
	assert.Equal(t, expectedAddr, tokenAddr)
}

func TestDecodeErc20TransferEvent_NotErc20Transfer(t *testing.T) {
	l := geth.Log{
		Topics: []common.Hash{
			common.BytesToHash(rand.Bytes(common.HashLength)),
			common.BytesToHash(common.LeftPadBytes(common.BytesToAddress(rand.Bytes(common.AddressLength)).Bytes(), common.HashLength)),
			common.BytesToHash(common.LeftPadBytes(common.BytesToAddress(rand.Bytes(common.AddressLength)).Bytes(), common.HashLength)),
		},
		Data: common.LeftPadBytes(big.NewInt(2).Bytes(), common.HashLength),
	}

	_, _, err := decodeERC20TransferEvent(&l)

	assert.Error(t, err)
}

func TestDecodeErc20TransferEvent_InvalidErc20Transfer(t *testing.T) {
	erc20TransferEventSig := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	l := geth.Log{
		Topics: []common.Hash{
			erc20TransferEventSig,
			common.BytesToHash(common.LeftPadBytes(common.BytesToAddress(rand.Bytes(common.AddressLength)).Bytes(), common.HashLength)),
		},
		Data: common.LeftPadBytes(big.NewInt(2).Bytes(), common.HashLength),
	}

	_, _, err := decodeERC20TransferEvent(&l)

	assert.Error(t, err)
}

func TestDecodeErc20TransferEvent_CorrectData(t *testing.T) {
	erc20TransferEventSig := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	expectedFrom := common.BytesToAddress(rand.Bytes(common.AddressLength))
	expectedTo := common.BytesToAddress(rand.Bytes(common.AddressLength))
	expectedAmount := sdk.NewUint(uint64(rand.I64Between(1, 10000)))

	l := geth.Log{
		Topics: []common.Hash{
			erc20TransferEventSig,
			common.BytesToHash(common.LeftPadBytes(expectedFrom.Bytes(), common.HashLength)),
			common.BytesToHash(common.LeftPadBytes(expectedTo.Bytes(), common.HashLength)),
		},
		Data: common.LeftPadBytes(expectedAmount.BigInt().Bytes(), common.HashLength),
	}

	actualTo, actualAmount, err := decodeERC20TransferEvent(&l)

	assert.NoError(t, err)
	assert.Equal(t, expectedTo, actualTo)
	assert.Equal(t, expectedAmount, actualAmount)
}

func TestDecodeTransferOwnershipEvent_CorrectData(t *testing.T) {
	transferOwnershipEventSig := common.HexToHash("0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0")
	expectedPrevOwner := common.BytesToAddress(rand.Bytes(common.AddressLength))
	expectedNewOwner := common.BytesToAddress(rand.Bytes(common.AddressLength))

	l := geth.Log{
		Topics: []common.Hash{
			transferOwnershipEventSig,
			common.BytesToHash(common.LeftPadBytes(expectedPrevOwner.Bytes(), common.HashLength)),
			common.BytesToHash(common.LeftPadBytes(expectedNewOwner.Bytes(), common.HashLength)),
		},
		Data: nil,
	}

	actualNewOwner, err := decodeSinglesigKeyTransferEvent(&l, evmTypes.Ownership)

	assert.NoError(t, err)
	assert.Equal(t, expectedNewOwner, actualNewOwner)
}

func TestDecodeMultisigKeyTransferEvent(t *testing.T) {
	t.Run("should decode the new multisig addresses and threshold", testutils.Func(func(t *testing.T) {
		log := geth.Log{
			Topics: []common.Hash{
				common.HexToHash("d167b96814cd24898418cc293e8d47d54afe6dcf0631283f0830e1eae621f6bd"),
			},
			Data: common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000005000000000000000000000000435b66b6d3889c9371a80bb3b42f438fcfb083a50000000000000000000000007b519ffcd280d5a7316b647b8d46a587bbebec140000000000000000000000009372ae5bcc1716741b323f39698e2f859412ced300000000000000000000000044db145b85cebb77b8269516152a931a6d9e0238000000000000000000000000579c2e330dd6a7bcc3abf8a21602adfc483b1f6400000000000000000000000000000000000000000000000000000000000000050000000000000000000000004b379b1aec479cae840b0c921c3c48c2c44c08e9000000000000000000000000d5403824cbdea1288e2ade9cb782ada6aa0c7466000000000000000000000000ea69ec886a7d763f933f7d442a6d437538008cb50000000000000000000000002b7f57804a9e60c25852c825e8400562efa690650000000000000000000000003b94e9fad488db2e57a701522a034311f0e7b1db"),
		}

		expectedAddresses := []common.Address{
			common.HexToAddress("4b379b1aec479cae840b0c921c3c48c2c44c08e9"),
			common.HexToAddress("d5403824cbdea1288e2ade9cb782ada6aa0c7466"),
			common.HexToAddress("ea69ec886a7d763f933f7d442a6d437538008cb5"),
			common.HexToAddress("2b7f57804a9e60c25852c825e8400562efa69065"),
			common.HexToAddress("3b94e9fad488db2e57a701522a034311f0e7b1db"),
		}
		expectedThreshold := uint8(3)
		actualAddresses, actualThreshold, err := decodeMultisigKeyTransferEvent(&log, evmTypes.Ownership)

		assert.NoError(t, err)
		assert.Equal(t, expectedAddresses, actualAddresses)
		assert.Equal(t, expectedThreshold, actualThreshold)
	}))

	t.Run("should return error when event is not about transfer of the correct multisig keys", testutils.Func(func(t *testing.T) {
		// wrong number of topics
		log := geth.Log{
			Topics: []common.Hash{
				common.HexToHash("d3f18f46b5db15b1cee9f5a6e26325d5a43e69fd18f131d2ffe41586765f8e0f"),
				common.HexToHash("d3f18f46b5db15b1cee9f5a6e26325d5a43e69fd18f131d2ffe41586765f8e0f"),
			},
			Data: common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000005000000000000000000000000435b66b6d3889c9371a80bb3b42f438fcfb083a50000000000000000000000007b519ffcd280d5a7316b647b8d46a587bbebec140000000000000000000000009372ae5bcc1716741b323f39698e2f859412ced300000000000000000000000044db145b85cebb77b8269516152a931a6d9e0238000000000000000000000000579c2e330dd6a7bcc3abf8a21602adfc483b1f6400000000000000000000000000000000000000000000000000000000000000050000000000000000000000004b379b1aec479cae840b0c921c3c48c2c44c08e9000000000000000000000000d5403824cbdea1288e2ade9cb782ada6aa0c7466000000000000000000000000ea69ec886a7d763f933f7d442a6d437538008cb50000000000000000000000002b7f57804a9e60c25852c825e8400562efa690650000000000000000000000003b94e9fad488db2e57a701522a034311f0e7b1db"),
		}
		_, _, err := decodeMultisigKeyTransferEvent(&log, evmTypes.Ownership)
		assert.Error(t, err)

		// wrong topics[0]
		log = geth.Log{
			Topics: []common.Hash{
				common.HexToHash("d3f18f46b5db15b1cee9f5a6e26325d5a43e69fd18f131d2ffe41586765f8e0f"),
			},
			Data: common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000005000000000000000000000000435b66b6d3889c9371a80bb3b42f438fcfb083a50000000000000000000000007b519ffcd280d5a7316b647b8d46a587bbebec140000000000000000000000009372ae5bcc1716741b323f39698e2f859412ced300000000000000000000000044db145b85cebb77b8269516152a931a6d9e0238000000000000000000000000579c2e330dd6a7bcc3abf8a21602adfc483b1f6400000000000000000000000000000000000000000000000000000000000000050000000000000000000000004b379b1aec479cae840b0c921c3c48c2c44c08e9000000000000000000000000d5403824cbdea1288e2ade9cb782ada6aa0c7466000000000000000000000000ea69ec886a7d763f933f7d442a6d437538008cb50000000000000000000000002b7f57804a9e60c25852c825e8400562efa690650000000000000000000000003b94e9fad488db2e57a701522a034311f0e7b1db"),
		}
		_, _, err = decodeMultisigKeyTransferEvent(&log, evmTypes.Operatorship)
		assert.Error(t, err)

		// wrong data
		log = geth.Log{
			Topics: []common.Hash{
				common.HexToHash("d3f18f46b5db15b1cee9f5a6e26325d5a43e69fd18f131d2ffe41586765f8e0f"),
			},
			Data: common.Hex2Bytes("ea69ec886a7d763f933f7d442a6d437538008cb5"),
		}
		_, _, err = decodeMultisigKeyTransferEvent(&log, evmTypes.Ownership)
		assert.Error(t, err)
	}))
}

func TestMgr_ProccessDepositConfirmation(t *testing.T) {
	var (
		mgr         *Mgr
		attributes  map[string]string
		rpc         *mock.ClientMock
		broadcaster *mock2.BroadcasterMock
	)
	setup := func() {
		cdc := app.MakeEncodingConfig().Amino
		pollKey := exported.NewPollKey(evmTypes.ModuleName, rand.StrBetween(5, 20))

		burnAddrBytes := rand.Bytes(common.AddressLength)
		tokenAddrBytes := rand.Bytes(common.AddressLength)
		blockNumber := rand.PInt64Gen().Where(func(i int64) bool { return i != 0 }).Next() // restrict to int64 so the block number in the receipt doesn't overflow
		confHeight := rand.I64Between(0, blockNumber-1)
		amount := rand.PosI64() // restrict to int64 so the amount in the receipt doesn't overflow
		attributes = map[string]string{
			evmTypes.AttributeKeyChain:          "Ethereum",
			evmTypes.AttributeKeyTxID:           common.Bytes2Hex(rand.Bytes(common.HashLength)),
			evmTypes.AttributeKeyAmount:         strconv.FormatUint(uint64(amount), 10),
			evmTypes.AttributeKeyDepositAddress: common.Bytes2Hex(burnAddrBytes),
			evmTypes.AttributeKeyTokenAddress:   common.Bytes2Hex(tokenAddrBytes),
			evmTypes.AttributeKeyConfHeight:     strconv.FormatUint(uint64(confHeight), 10),
			evmTypes.AttributeKeyPoll:           string(cdc.MustMarshalJSON(pollKey)),
		}

		rpc = &mock.ClientMock{
			BlockNumberFunc: func(context.Context) (uint64, error) {
				return uint64(blockNumber), nil
			},
			TransactionByHashFunc: func(ctx context.Context, hash common.Hash) (*geth.Transaction, bool, error) {
				return &geth.Transaction{}, false, nil
			},
			TransactionReceiptFunc: func(context.Context, common.Hash) (*geth.Receipt, error) {
				receipt := &geth.Receipt{
					BlockNumber: big.NewInt(rand.I64Between(0, blockNumber-confHeight)),
					Logs: []*geth.Log{
						/* ERC20 transfer to burner address of a random token */
						{
							Address: common.BytesToAddress(rand.Bytes(common.AddressLength)),
							Topics: []common.Hash{
								ERC20TransferSig,
								common.BytesToHash(common.LeftPadBytes(rand.Bytes(common.AddressLength), common.HashLength)),
								common.BytesToHash(common.LeftPadBytes(burnAddrBytes, common.HashLength)),
							},
							Data: common.LeftPadBytes(big.NewInt(rand.PosI64()).Bytes(), common.HashLength),
						},
						/* not a ERC20 transfer */
						{
							Address: common.BytesToAddress(tokenAddrBytes),
							Topics: []common.Hash{
								common.BytesToHash(rand.Bytes(common.HashLength)),
								common.BytesToHash(common.LeftPadBytes(rand.Bytes(common.AddressLength), common.HashLength)),
								common.BytesToHash(common.LeftPadBytes(burnAddrBytes, common.HashLength)),
							},
							Data: common.LeftPadBytes(big.NewInt(rand.PosI64()).Bytes(), common.HashLength),
						},
						/* an invalid ERC20 transfer */
						{
							Address: common.BytesToAddress(tokenAddrBytes),
							Topics: []common.Hash{
								ERC20TransferSig,
								common.BytesToHash(common.LeftPadBytes(rand.Bytes(common.AddressLength), common.HashLength)),
							},
							Data: common.LeftPadBytes(big.NewInt(rand.PosI64()).Bytes(), common.HashLength),
						},
						/* an ERC20 transfer of our concern */
						{
							Address: common.BytesToAddress(tokenAddrBytes),
							Topics: []common.Hash{
								ERC20TransferSig,
								common.BytesToHash(common.LeftPadBytes(rand.Bytes(common.AddressLength), common.HashLength)),
								common.BytesToHash(common.LeftPadBytes(burnAddrBytes, common.HashLength)),
							},
							Data: common.LeftPadBytes(big.NewInt(amount).Bytes(), common.HashLength),
						},
					},
					Status: 1,
				}
				return receipt, nil
			},
		}
		broadcaster = &mock2.BroadcasterMock{}
		evmMap := make(map[string]evmRpc.Client)
		evmMap["ethereum"] = rpc
		mgr = NewMgr(evmMap, client.Context{}, broadcaster, log.TestingLogger(), cdc)
	}
	repeats := 20
	t.Run("happy path", testutils.Func(func(t *testing.T) {
		setup()

		err := mgr.ProcessDepositConfirmation(tmEvents.Event{Attributes: attributes})

		assert.NoError(t, err)
		assert.Len(t, broadcaster.BroadcastCalls(), 1)
		msg := unwrapRefundMsg(broadcaster.BroadcastCalls()[0].Msgs[0])
		assert.True(t, msg.(*evmTypes.VoteConfirmDepositRequest).Confirmed)
	}).Repeat(repeats))

	t.Run("missing attributes", testutils.Func(func(t *testing.T) {
		setup()
		for key := range attributes {
			delete(attributes, key)

			err := mgr.ProcessDepositConfirmation(tmEvents.Event{Attributes: attributes})
			assert.Error(t, err)
			assert.Len(t, broadcaster.BroadcastCalls(), 0)
		}
	}).Repeat(repeats))

	t.Run("no tx receipt", testutils.Func(func(t *testing.T) {
		setup()
		rpc.TransactionReceiptFunc = func(context.Context, common.Hash) (*geth.Receipt, error) { return nil, fmt.Errorf("error") }

		err := mgr.ProcessDepositConfirmation(tmEvents.Event{Attributes: attributes})

		assert.NoError(t, err)
		assert.Len(t, broadcaster.BroadcastCalls(), 1)
		msg := unwrapRefundMsg(broadcaster.BroadcastCalls()[0].Msgs[0])
		assert.False(t, msg.(*evmTypes.VoteConfirmDepositRequest).Confirmed)
	}).Repeat(repeats))

	t.Run("no block number", testutils.Func(func(t *testing.T) {
		setup()
		rpc.BlockNumberFunc = func(context.Context) (uint64, error) {
			return 0, fmt.Errorf("error")
		}

		err := mgr.ProcessDepositConfirmation(tmEvents.Event{Attributes: attributes})

		assert.NoError(t, err)
		assert.Len(t, broadcaster.BroadcastCalls(), 1)
		msg := unwrapRefundMsg(broadcaster.BroadcastCalls()[0].Msgs[0])
		assert.False(t, msg.(*evmTypes.VoteConfirmDepositRequest).Confirmed)
	}).Repeat(repeats))

	t.Run("amount mismatch", testutils.Func(func(t *testing.T) {
		setup()
		attributes[evmTypes.AttributeKeyAmount] = strconv.FormatUint(mathRand.Uint64(), 10)

		err := mgr.ProcessDepositConfirmation(tmEvents.Event{Attributes: attributes})

		assert.NoError(t, err)
		assert.Len(t, broadcaster.BroadcastCalls(), 1)
		msg := unwrapRefundMsg(broadcaster.BroadcastCalls()[0].Msgs[0])
		assert.False(t, msg.(*evmTypes.VoteConfirmDepositRequest).Confirmed)
	}).Repeat(repeats))
}

func TestMgr_ProccessTokenConfirmation(t *testing.T) {
	var (
		mgr              *Mgr
		attributes       map[string]string
		rpc              *mock.ClientMock
		broadcaster      *mock2.BroadcasterMock
		gatewayAddrBytes []byte
	)
	setup := func() {
		cdc := app.MakeEncodingConfig().Amino
		pollKey := exported.NewPollKey(evmTypes.ModuleName, rand.StrBetween(5, 20))

		gatewayAddrBytes = rand.Bytes(common.AddressLength)
		tokenAddrBytes := rand.Bytes(common.AddressLength)
		blockNumber := rand.PInt64Gen().Where(func(i int64) bool { return i != 0 }).Next() // restrict to int64 so the block number in the receipt doesn't overflow
		confHeight := rand.I64Between(0, blockNumber-1)

		symbol := rand.StrBetween(5, 20)
		attributes = map[string]string{
			evmTypes.AttributeKeyChain:          "Ethereum",
			evmTypes.AttributeKeyTxID:           common.Bytes2Hex(rand.Bytes(common.HashLength)),
			evmTypes.AttributeKeyGatewayAddress: common.Bytes2Hex(gatewayAddrBytes),
			evmTypes.AttributeKeyTokenAddress:   common.Bytes2Hex(tokenAddrBytes),
			evmTypes.AttributeKeySymbol:         symbol,
			evmTypes.AttributeKeyAsset:          "satoshi",
			evmTypes.AttributeKeyConfHeight:     strconv.FormatUint(uint64(confHeight), 10),
			evmTypes.AttributeKeyPoll:           string(cdc.MustMarshalJSON(pollKey)),
		}

		rpc = &mock.ClientMock{
			BlockNumberFunc: func(context.Context) (uint64, error) {
				return uint64(blockNumber), nil
			},
			TransactionByHashFunc: func(ctx context.Context, hash common.Hash) (*geth.Transaction, bool, error) {
				return &geth.Transaction{}, false, nil
			},
			TransactionReceiptFunc: func(context.Context, common.Hash) (*geth.Receipt, error) {
				receipt := &geth.Receipt{
					BlockNumber: big.NewInt(rand.I64Between(0, blockNumber-confHeight)),
					Logs: createTokenLogs(
						symbol,
						common.BytesToAddress(gatewayAddrBytes),
						common.BytesToAddress(tokenAddrBytes),
						ERC20TokenDeploymentSig,
						true,
					),
					Status: 1,
				}
				return receipt, nil
			},
		}
		broadcaster = &mock2.BroadcasterMock{}
		evmMap := make(map[string]evmRpc.Client)
		evmMap["ethereum"] = rpc
		mgr = NewMgr(evmMap, client.Context{}, broadcaster, log.TestingLogger(), cdc)
	}

	repeats := 20
	t.Run("happy path", testutils.Func(func(t *testing.T) {
		setup()

		err := mgr.ProcessTokenConfirmation(tmEvents.Event{Attributes: attributes})

		assert.NoError(t, err)
		assert.Len(t, broadcaster.BroadcastCalls(), 1)
		msg := unwrapRefundMsg(broadcaster.BroadcastCalls()[0].Msgs[0])
		assert.True(t, msg.(*evmTypes.VoteConfirmTokenRequest).Confirmed)
	}).Repeat(repeats))

	t.Run("missing attributes", testutils.Func(func(t *testing.T) {
		setup()
		for key := range attributes {
			delete(attributes, key)

			err := mgr.ProcessTokenConfirmation(tmEvents.Event{Attributes: attributes})
			assert.Error(t, err)
			assert.Len(t, broadcaster.BroadcastCalls(), 0)
		}
	}).Repeat(repeats))

	t.Run("no tx receipt", testutils.Func(func(t *testing.T) {
		setup()
		rpc.TransactionReceiptFunc = func(context.Context, common.Hash) (*geth.Receipt, error) { return nil, fmt.Errorf("error") }

		err := mgr.ProcessTokenConfirmation(tmEvents.Event{Attributes: attributes})

		assert.NoError(t, err)
		assert.Len(t, broadcaster.BroadcastCalls(), 1)
		msg := unwrapRefundMsg(broadcaster.BroadcastCalls()[0].Msgs[0])
		assert.False(t, msg.(*evmTypes.VoteConfirmTokenRequest).Confirmed)
	}).Repeat(repeats))

	t.Run("no block number", testutils.Func(func(t *testing.T) {
		setup()
		rpc.BlockNumberFunc = func(context.Context) (uint64, error) {
			return 0, fmt.Errorf("error")
		}

		err := mgr.ProcessTokenConfirmation(tmEvents.Event{Attributes: attributes})

		assert.NoError(t, err)
		assert.Len(t, broadcaster.BroadcastCalls(), 1)
		msg := unwrapRefundMsg(broadcaster.BroadcastCalls()[0].Msgs[0])
		assert.False(t, msg.(*evmTypes.VoteConfirmTokenRequest).Confirmed)
	}).Repeat(repeats))

	t.Run("no deploy event", testutils.Func(func(t *testing.T) {
		setup()
		receipt, _ := rpc.TransactionReceipt(context.Background(), common.Hash{})
		var correctLogIdx int
		for i, l := range receipt.Logs {
			if l.Address == common.BytesToAddress(gatewayAddrBytes) {
				correctLogIdx = i
				break
			}
		}
		// remove the deploy event
		receipt.Logs = append(receipt.Logs[:correctLogIdx], receipt.Logs[correctLogIdx+1:]...)
		rpc.TransactionReceiptFunc = func(context.Context, common.Hash) (*geth.Receipt, error) { return receipt, nil }

		err := mgr.ProcessTokenConfirmation(tmEvents.Event{Attributes: attributes})

		assert.NoError(t, err)
		assert.Len(t, broadcaster.BroadcastCalls(), 1)
		msg := unwrapRefundMsg(broadcaster.BroadcastCalls()[0].Msgs[0])
		assert.False(t, msg.(*evmTypes.VoteConfirmTokenRequest).Confirmed)
	}).Repeat(repeats))

	t.Run("wrong deploy event", testutils.Func(func(t *testing.T) {
		setup()
		receipt, _ := rpc.TransactionReceipt(context.Background(), common.Hash{})
		for _, l := range receipt.Logs {
			if l.Address == common.BytesToAddress(gatewayAddrBytes) {
				l.Data = rand.Bytes(int(rand.I64Between(0, 1000)))
				break
			}
		}
		rpc.TransactionReceiptFunc = func(context.Context, common.Hash) (*geth.Receipt, error) { return receipt, nil }

		err := mgr.ProcessTokenConfirmation(tmEvents.Event{Attributes: attributes})

		assert.NoError(t, err)
		assert.Len(t, broadcaster.BroadcastCalls(), 1)
		msg := unwrapRefundMsg(broadcaster.BroadcastCalls()[0].Msgs[0])
		assert.False(t, msg.(*evmTypes.VoteConfirmTokenRequest).Confirmed)
	}).Repeat(repeats))
}

func TestMgr_ProcessTransferKeyConfirmation(t *testing.T) {
	var (
		mgr                   *Mgr
		attributes            map[string]string
		rpc                   *mock.ClientMock
		broadcaster           *mock2.BroadcasterMock
		prevNewOwnerAddrBytes []byte
	)
	setup := func() {
		cdc := app.MakeEncodingConfig().Amino
		pollKey := exported.NewPollKey(evmTypes.ModuleName, rand.StrBetween(5, 20))

		gatewayAddrBytes := rand.Bytes(common.AddressLength)
		newOwnerAddrBytes := rand.Bytes(common.AddressLength)
		prevNewOwnerAddrBytes = rand.Bytes(common.AddressLength)
		blockNumber := rand.PInt64Gen().Where(func(i int64) bool { return i != 0 }).Next() // restrict to int64 so the block number in the receipt doesn't overflow
		confHeight := rand.I64Between(0, blockNumber-1)

		attributes = map[string]string{
			evmTypes.AttributeKeyChain:           "Ethereum",
			evmTypes.AttributeKeyTxID:            common.Bytes2Hex(rand.Bytes(common.HashLength)),
			evmTypes.AttributeKeyTransferKeyType: evmTypes.Ownership.SimpleString(),
			evmTypes.AttributeKeyKeyType:         tss.Threshold.SimpleString(),
			evmTypes.AttributeKeyGatewayAddress:  common.Bytes2Hex(gatewayAddrBytes),
			evmTypes.AttributeKeyAddress:         common.Bytes2Hex(newOwnerAddrBytes),
			evmTypes.AttributeKeyThreshold:       "",
			evmTypes.AttributeKeyConfHeight:      strconv.FormatUint(uint64(confHeight), 10),
			evmTypes.AttributeKeyPoll:            string(cdc.MustMarshalJSON(pollKey)),
		}

		rpc = &mock.ClientMock{
			BlockNumberFunc: func(context.Context) (uint64, error) {
				return uint64(blockNumber), nil
			},
			TransactionByHashFunc: func(ctx context.Context, hash common.Hash) (*geth.Transaction, bool, error) {
				return &geth.Transaction{}, false, nil
			},
			TransactionReceiptFunc: func(context.Context, common.Hash) (*geth.Receipt, error) {
				receipt := &geth.Receipt{
					BlockNumber: big.NewInt(rand.I64Between(0, blockNumber-confHeight)),
					Logs: []*geth.Log{
						/* previous transfer ownership event */
						{
							Address: common.BytesToAddress(gatewayAddrBytes),
							Topics: []common.Hash{
								SinglesigTransferOwnershipSig,
								common.BytesToHash(common.LeftPadBytes(rand.Bytes(common.AddressLength), common.HashLength)),
								common.BytesToHash(common.LeftPadBytes(prevNewOwnerAddrBytes, common.HashLength)),
							},
							Data: nil,
						},
						/* a transfer ownership of our concern */
						{
							Address: common.BytesToAddress(gatewayAddrBytes),
							Topics: []common.Hash{
								SinglesigTransferOwnershipSig,
								common.BytesToHash(common.LeftPadBytes(rand.Bytes(common.AddressLength), common.HashLength)),
								common.BytesToHash(common.LeftPadBytes(newOwnerAddrBytes, common.HashLength)),
							},
							Data: nil,
						},
						/* an invalid transfer ownership */
						{
							Address: common.BytesToAddress(gatewayAddrBytes),
							Topics: []common.Hash{
								SinglesigTransferOwnershipSig,
								common.BytesToHash(common.LeftPadBytes(rand.Bytes(common.AddressLength), common.HashLength)),
							},
							Data: nil,
						},
						/* not a transfer ownership event */
						{
							Address: common.BytesToAddress(gatewayAddrBytes),
							Topics: []common.Hash{
								common.BytesToHash(rand.Bytes(common.HashLength)),
								common.BytesToHash(common.LeftPadBytes(rand.Bytes(common.AddressLength), common.HashLength)),
								common.BytesToHash(common.LeftPadBytes(newOwnerAddrBytes, common.HashLength)),
							},
							Data: nil,
						},
						/* transfer ownership event from a random address */
						{
							Address: common.BytesToAddress(rand.Bytes(common.AddressLength)),
							Topics: []common.Hash{
								SinglesigTransferOwnershipSig,
								common.BytesToHash(common.LeftPadBytes(rand.Bytes(common.AddressLength), common.HashLength)),
								common.BytesToHash(common.LeftPadBytes(newOwnerAddrBytes, common.HashLength)),
							},
							Data: nil,
						},
					},
					Status: 1,
				}
				return receipt, nil
			},
		}
		broadcaster = &mock2.BroadcasterMock{}
		evmMap := make(map[string]evmRpc.Client)
		evmMap["ethereum"] = rpc
		mgr = NewMgr(evmMap, client.Context{}, broadcaster, log.TestingLogger(), cdc)
	}

	repeats := 20
	t.Run("happy path", testutils.Func(func(t *testing.T) {
		setup()

		err := mgr.ProcessTransferKeyConfirmation(tmEvents.Event{Attributes: attributes})

		assert.NoError(t, err)
		assert.Len(t, broadcaster.BroadcastCalls(), 1)
		msg := unwrapRefundMsg(broadcaster.BroadcastCalls()[0].Msgs[0])
		assert.True(t, msg.(*evmTypes.VoteConfirmTransferKeyRequest).Confirmed)
	}).Repeat(repeats))

	t.Run("missing attributes", testutils.Func(func(t *testing.T) {
		setup()
		for key := range attributes {
			delete(attributes, key)

			err := mgr.ProcessTransferKeyConfirmation(tmEvents.Event{Attributes: attributes})
			assert.Error(t, err)
			assert.Len(t, broadcaster.BroadcastCalls(), 0)
		}
	}).Repeat(repeats))

	t.Run("no tx receipt", testutils.Func(func(t *testing.T) {
		setup()
		rpc.TransactionReceiptFunc = func(context.Context, common.Hash) (*geth.Receipt, error) { return nil, fmt.Errorf("error") }

		err := mgr.ProcessTransferKeyConfirmation(tmEvents.Event{Attributes: attributes})

		assert.NoError(t, err)
		assert.Len(t, broadcaster.BroadcastCalls(), 1)
		msg := unwrapRefundMsg(broadcaster.BroadcastCalls()[0].Msgs[0])
		assert.False(t, msg.(*evmTypes.VoteConfirmTransferKeyRequest).Confirmed)
	}).Repeat(repeats))

	t.Run("no block number", testutils.Func(func(t *testing.T) {
		setup()
		rpc.BlockNumberFunc = func(context.Context) (uint64, error) {
			return 0, fmt.Errorf("error")
		}

		err := mgr.ProcessTransferKeyConfirmation(tmEvents.Event{Attributes: attributes})

		assert.NoError(t, err)
		assert.Len(t, broadcaster.BroadcastCalls(), 1)
		msg := unwrapRefundMsg(broadcaster.BroadcastCalls()[0].Msgs[0])
		assert.False(t, msg.(*evmTypes.VoteConfirmTransferKeyRequest).Confirmed)
	}).Repeat(repeats))

	t.Run("new owner mismatch", testutils.Func(func(t *testing.T) {
		setup()

		attributes[evmTypes.AttributeKeyAddress] = common.BytesToAddress(rand.Bytes(common.AddressLength)).Hex()

		err := mgr.ProcessTransferKeyConfirmation(tmEvents.Event{Attributes: attributes})

		assert.NoError(t, err)
		assert.Len(t, broadcaster.BroadcastCalls(), 1)
		msg := unwrapRefundMsg(broadcaster.BroadcastCalls()[0].Msgs[0])
		assert.False(t, msg.(*evmTypes.VoteConfirmTransferKeyRequest).Confirmed)
	}).Repeat(repeats))

	t.Run("receipt status failed", testutils.Func(func(t *testing.T) {
		setup()
		rpc.TransactionReceiptFunc = func(context.Context, common.Hash) (*geth.Receipt, error) {
			receipt := &geth.Receipt{
				BlockNumber: big.NewInt(1),
				Logs:        nil,
				Status:      0,
			}
			return receipt, nil
		}
		err := mgr.ProcessTransferKeyConfirmation(tmEvents.Event{Attributes: attributes})

		assert.NoError(t, err)
		assert.Len(t, broadcaster.BroadcastCalls(), 1)
		msg := unwrapRefundMsg(broadcaster.BroadcastCalls()[0].Msgs[0])
		assert.False(t, msg.(*evmTypes.VoteConfirmTransferKeyRequest).Confirmed)
	}).Repeat(repeats))

	t.Run("new owner not last transfer event", testutils.Func(func(t *testing.T) {
		setup()

		attributes[evmTypes.AttributeKeyAddress] = common.BytesToAddress(prevNewOwnerAddrBytes).Hex()

		err := mgr.ProcessTransferKeyConfirmation(tmEvents.Event{Attributes: attributes})

		assert.NoError(t, err)
		assert.Len(t, broadcaster.BroadcastCalls(), 1)
		msg := unwrapRefundMsg(broadcaster.BroadcastCalls()[0].Msgs[0])
		assert.False(t, msg.(*evmTypes.VoteConfirmTransferKeyRequest).Confirmed)
	}).Repeat(repeats))
}

func createTokenLogs(denom string, gateway, tokenAddr common.Address, deploySig common.Hash, hasCorrectLog bool) []*geth.Log {
	numLogs := rand.I64Between(1, 100)
	correctPos := rand.I64Between(0, numLogs)
	var logs []*geth.Log

	for i := int64(0); i < numLogs; i++ {
		stringType, err := abi.NewType("string", "string", nil)
		if err != nil {
			panic(err)
		}
		addressType, err := abi.NewType("address", "address", nil)
		if err != nil {
			panic(err)
		}
		args := abi.Arguments{{Type: stringType}, {Type: addressType}}

		switch {
		case hasCorrectLog && i == correctPos:
			data, err := args.Pack(denom, tokenAddr)
			if err != nil {
				panic(err)
			}
			logs = append(logs, &geth.Log{Address: gateway, Data: data, Topics: []common.Hash{deploySig}})
		default:
			randDenom := rand.StrBetween(5, 20)
			randAddr := common.BytesToAddress(rand.Bytes(common.AddressLength))
			randData, err := args.Pack(randDenom, randAddr)
			if err != nil {
				panic(err)
			}
			logs = append(logs, &geth.Log{
				Address: common.BytesToAddress(rand.Bytes(common.AddressLength)),
				Data:    randData,
				Topics:  []common.Hash{common.BytesToHash(rand.Bytes(common.HashLength))},
			})
		}
	}

	return logs
}

func unwrapRefundMsg(msg sdk.Msg) sdk.Msg {
	return msg.(*rewardtypes.RefundMsgRequest).GetInnerMessage()
}
