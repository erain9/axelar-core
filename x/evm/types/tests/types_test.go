package tests

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/libs/log"

	paramsKeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"

	"github.com/axelarnetwork/axelar-core/app"
	"github.com/axelarnetwork/axelar-core/testutils/fake"
	"github.com/axelarnetwork/axelar-core/x/evm/exported"
	"github.com/axelarnetwork/axelar-core/x/evm/keeper"
	"github.com/axelarnetwork/axelar-core/x/evm/types"
	nexus "github.com/axelarnetwork/axelar-core/x/nexus/exported"
)

func TestCreateMintCommandData_SingleMint(t *testing.T) {
	chainID := big.NewInt(1)
	var commandID types.CommandID
	copy(commandID[:], common.FromHex("0xec78d9c22c08bb9f0ecd5d95571ae83e3f22219c5a9278c3270691d50abfd91b"))
	address := "0x63FC2aD3d021a4D7e64323529a55a9442C444dA0"
	denom := "aat"
	amount := sdk.NewInt(9999)

	expected := "0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000140000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000027100000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000096d696e74546f6b656e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000006000000000000000000000000063fc2ad3d021a4d7e64323529a55a9442c444da0000000000000000000000000000000000000000000000000000000000000270f00000000000000000000000000000000000000000000000000000000000000036161740000000000000000000000000000000000000000000000000000000000"
	x := common.Hex2Bytes(expected)
	assert.NotNil(t, x)
	actual, err := types.CreateMintCommandData(
		chainID,
		[]nexus.CrossChainTransfer{
			{
				Recipient: nexus.CrossChainAddress{
					Chain:   exported.Ethereum,
					Address: address,
				},
				Asset: sdk.NewCoin(denom, amount),
				ID:    10000,
			},
		},
	)

	assert.NoError(t, err)
	assert.Equal(t, expected, common.Bytes2Hex(actual))
}

func TestCreateMintCommandData_MultipleMint(t *testing.T) {
	chainID := big.NewInt(1)
	var commandID types.CommandID
	copy(commandID[:], common.FromHex("0xec78d9c22c08bb9f0ecd5d95571ae83e3f22219c5a9278c3270691d50abfd91b"))
	addressA := "0x63FC2aD3d021a4D7e64323529a55a9442C444dA0"
	addressB := "0x4183d62963434056e75e9854BC4ba92AA43A2d08"
	denomA := "aat"
	denomB := "abtc"
	amountA := sdk.NewInt(9999)
	amountB := sdk.NewInt(9999999)

	expected := "0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000e000000000000000000000000000000000000000000000000000000000000001c000000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000096d696e74546f6b656e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000096d696e74546f6b656e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000006000000000000000000000000063fc2ad3d021a4d7e64323529a55a9442c444da0000000000000000000000000000000000000000000000000000000000000270f0000000000000000000000000000000000000000000000000000000000000003616174000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000600000000000000000000000004183d62963434056e75e9854bc4ba92aa43a2d08000000000000000000000000000000000000000000000000000000000098967f00000000000000000000000000000000000000000000000000000000000000046162746300000000000000000000000000000000000000000000000000000000"
	actual, err := types.CreateMintCommandData(
		chainID,
		[]nexus.CrossChainTransfer{
			{Recipient: nexus.CrossChainAddress{Chain: exported.Ethereum, Address: addressA}, Asset: sdk.NewCoin(denomA, amountA), ID: 1},
			{Recipient: nexus.CrossChainAddress{Chain: exported.Ethereum, Address: addressB}, Asset: sdk.NewCoin(denomB, amountB), ID: 2},
		})

	assert.NoError(t, err)
	assert.Equal(t, expected, common.Bytes2Hex(actual))
}

func TestCreateDeployTokenCommandData_CorrectData(t *testing.T) {
	chainID := big.NewInt(1)
	var commandID types.CommandID
	copy(commandID[:], common.FromHex("0x5763814b98a3aa86f212797af3273868b5dd6e2a532d764a79b98ca859e7bbad"))
	tokenName := "an awesome token"
	symbol := "aat"
	decimals := uint8(18)
	capacity := sdk.NewInt(10000)

	expected := "0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000015763814b98a3aa86f212797af3273868b5dd6e2a532d764a79b98ca859e7bbad00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000b6465706c6f79546f6b656e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000001200000000000000000000000000000000000000000000000000000000000027100000000000000000000000000000000000000000000000000000000000000010616e20617765736f6d6520746f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000036161740000000000000000000000000000000000000000000000000000000000"
	actual, err := types.CreateDeployTokenCommandData(
		chainID,
		commandID,
		tokenName,
		symbol,
		decimals,
		capacity,
	)

	assert.NoError(t, err)
	assert.Equal(t, expected, common.Bytes2Hex(actual))
}

func TestCreateBurnCommandData_SingleBurn(t *testing.T) {
	chainID := big.NewInt(1)
	symbol := "aat"
	salt := types.Hash(common.HexToHash("0x35f28b34202f4e3de20c1710696e3f294ebe4df686b17be00fedf991190f9654"))
	height := int64(50)

	expected := "0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000016f98406daa3f58892d52627d81196a731883e8d814706898058c75cc255ac28e0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000096275726e546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000004035f28b34202f4e3de20c1710696e3f294ebe4df686b17be00fedf991190f965400000000000000000000000000000000000000000000000000000000000000036161740000000000000000000000000000000000000000000000000000000000"
	actual, err := types.CreateBurnCommandData(
		chainID,
		height,
		[]types.BurnerInfo{{Symbol: symbol, Salt: salt}},
	)

	assert.NoError(t, err)
	assert.Equal(t, expected, common.Bytes2Hex(actual))
}

func TestCreateBurnCommandData_MultipleBurn(t *testing.T) {
	chainID := big.NewInt(1)
	symbolA := "aat"
	symbolB := "abtc"
	saltA := types.Hash(common.HexToHash("0x35f28b34202f4e3de20c1710696e3f294ebe4df686b17be00fedf991190f9654"))
	saltB := types.Hash(common.HexToHash("0xf15b565f2e52197b78d55c1cc9c5e27f28dcce75ae0e89d75e768a0542dac1ab"))
	height := int64(50)

	expected := "0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000e000000000000000000000000000000000000000000000000000000000000001c000000000000000000000000000000000000000000000000000000000000000026f98406daa3f58892d52627d81196a731883e8d814706898058c75cc255ac28ec023632ac2841de5093427fd892e7e3945eb204893bc6a35538e695324a2f05800000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000096275726e546f6b656e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000096275726e546f6b656e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000004035f28b34202f4e3de20c1710696e3f294ebe4df686b17be00fedf991190f96540000000000000000000000000000000000000000000000000000000000000003616174000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000040f15b565f2e52197b78d55c1cc9c5e27f28dcce75ae0e89d75e768a0542dac1ab00000000000000000000000000000000000000000000000000000000000000046162746300000000000000000000000000000000000000000000000000000000"
	actual, err := types.CreateBurnCommandData(
		chainID,
		height,
		[]types.BurnerInfo{
			{Symbol: symbolA, Salt: saltA},
			{Symbol: symbolB, Salt: saltB},
		},
	)

	assert.NoError(t, err)
	assert.Equal(t, expected, common.Bytes2Hex(actual))
}

func TestGetEthereumSignHash_CorrectEthereumSignHash(t *testing.T) {
	data := common.FromHex("0000000000000000000000000000000000000000000000000000000000000001ec78d9c22c08bb9f0ecd5d95571ae83e3f22219c5a9278c3270691d50abfd91b000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000000096d696e74546f6b656e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e000000000000000000000000000000000000000000000000000000000000000014141540000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000063fc2ad3d021a4d7e64323529a55a9442c444da00000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000270f")

	expected := "0xe7bce8f57491e71212d930096bacf9288c711e5f27200946edd570e3a93546bf"
	actual := types.GetEthereumSignHash(data)

	assert.Equal(t, expected, actual.Hex())
}

func TestCreateExecuteData_CorrectExecuteData(t *testing.T) {
	commandData := common.FromHex("0000000000000000000000000000000000000000000000000000000000000001ec78d9c22c08bb9f0ecd5d95571ae83e3f22219c5a9278c3270691d50abfd91b000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000000096d696e74546f6b656e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e000000000000000000000000000000000000000000000000000000000000000014141540000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000063fc2ad3d021a4d7e64323529a55a9442c444da00000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000270f")
	commandSig := types.Signature{}
	copy(commandSig[:], common.FromHex("42b936b3c37fb7deed86f52154798d0c9abfe5ba838b2488f4a7e5193a9bb60b5d8c521f5c8c64f9442fc745ecd3bc496b04dc03a81b4e89c72342ab5903284d1c"))

	expected := "09c5eabe000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000002e00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000026000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000001ec78d9c22c08bb9f0ecd5d95571ae83e3f22219c5a9278c3270691d50abfd91b000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000000096d696e74546f6b656e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e000000000000000000000000000000000000000000000000000000000000000014141540000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000063fc2ad3d021a4d7e64323529a55a9442c444da00000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000270f000000000000000000000000000000000000000000000000000000000000004142b936b3c37fb7deed86f52154798d0c9abfe5ba838b2488f4a7e5193a9bb60b5d8c521f5c8c64f9442fc745ecd3bc496b04dc03a81b4e89c72342ab5903284d1c00000000000000000000000000000000000000000000000000000000000000"
	actual, err := types.CreateExecuteData(commandData, commandSig)

	assert.NoError(t, err)
	assert.Equal(t, expected, common.Bytes2Hex(actual))
}

func TestGetTokenAddress_CorrectData(t *testing.T) {
	encCfg := app.MakeEncodingConfig()
	ctx := sdk.NewContext(fake.NewMultiStore(), tmproto.Header{}, false, log.TestingLogger())
	paramsK := paramsKeeper.NewKeeper(encCfg.Marshaler, encCfg.Amino, sdk.NewKVStoreKey("subspace"), sdk.NewKVStoreKey("tsubspace"))
	k := keeper.NewKeeper(encCfg.Marshaler, sdk.NewKVStoreKey("testKey"), paramsK)

	chain := "Ethereum"
	axelarGateway := common.HexToAddress("0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA")
	tokenName := "axelar token"
	tokenSymbol := "at"
	decimals := uint8(18)
	capacity := sdk.NewIntFromUint64(uint64(10000))

	expected := common.HexToAddress("0xE7481ECB61F9C84b91C03414F3D5d48E5436045D")

	k.SetParams(ctx, types.DefaultParams()...)
	account, err := sdk.AccAddressFromBech32("cosmos1vjyc4qmsdtdl5a4ruymnjqpchm5gyqde63sqdh")
	assert.NoError(t, err)
	keeper := k.ForChain(ctx, chain)
	keeper.SetTokenInfo(ctx, &types.SignDeployTokenRequest{Sender: account, TokenName: tokenName, Symbol: tokenSymbol, Decimals: decimals, Capacity: capacity})

	actual, err := keeper.GetTokenAddress(ctx, tokenSymbol, axelarGateway)

	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestGetBurnerAddressAndSalt_CorrectData(t *testing.T) {
	encCfg := app.MakeEncodingConfig()
	ctx := sdk.NewContext(fake.NewMultiStore(), tmproto.Header{}, false, log.TestingLogger())
	paramsK := paramsKeeper.NewKeeper(encCfg.Marshaler, encCfg.Amino, sdk.NewKVStoreKey("subspace"), sdk.NewKVStoreKey("tsubspace"))
	k := keeper.NewKeeper(encCfg.Marshaler, sdk.NewKVStoreKey("testKey"), paramsK)

	axelarGateway := common.HexToAddress("0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA")
	recipient := "1KDeqnsTRzFeXRaENA6XLN1EwdTujchr4L"
	tokenAddr := common.HexToAddress("0xE7481ECB61F9C84b91C03414F3D5d48E5436045D")
	expectedBurnerAddr := common.HexToAddress("0x5f185DAFBD08F00E2826c195087A722B0A094059")
	expectedSalt := common.Hex2Bytes("35f28b34202f4e3de20c1710696e3f294ebe4df686b17be00fedf991190f9654")

	k.SetParams(ctx, types.DefaultParams()...)

	actualburnerAddr, actualSalt, err := k.ForChain(ctx, exported.Ethereum.Name).GetBurnerAddressAndSalt(ctx, tokenAddr, recipient, axelarGateway)

	assert.NoError(t, err)
	assert.Equal(t, expectedBurnerAddr, actualburnerAddr)
	assert.Equal(t, expectedSalt, actualSalt[:])
}

func TestCreateTransferOwnershipCommandData_CorrectData(t *testing.T) {
	chainID := big.NewInt(1)
	var commandID types.CommandID
	copy(commandID[:], common.FromHex("0x5763814b98a3aa86f212797af3273868b5dd6e2a532d764a79b98ca859e7bbad"))
	newOwnerAddr := common.HexToAddress("0xE5251FcFFde3a5BA84A427158A60a07816502590")

	expected := "0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000015763814b98a3aa86f212797af3273868b5dd6e2a532d764a79b98ca859e7bbad0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000117472616e736665724f776e657273686970000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000020000000000000000000000000e5251fcffde3a5ba84a427158a60a07816502590"

	actual, err := types.CreateTransferOwnershipCommandData(
		chainID,
		commandID,
		newOwnerAddr,
	)

	assert.NoError(t, err)
	assert.Equal(t, expected, common.Bytes2Hex(actual))
}
