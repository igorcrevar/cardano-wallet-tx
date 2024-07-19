package core

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	protocolParameters, _ = hex.DecodeString("7b22636f6c6c61746572616c50657263656e74616765223a3135302c22646563656e7472616c697a6174696f6e223a6e756c6c2c22657865637574696f6e556e6974507269636573223a7b2270726963654d656d6f7279223a302e303537372c2270726963655374657073223a302e303030303732317d2c2265787472615072616f73456e74726f7079223a6e756c6c2c226d6178426c6f636b426f647953697a65223a39303131322c226d6178426c6f636b457865637574696f6e556e697473223a7b226d656d6f7279223a36323030303030302c227374657073223a32303030303030303030307d2c226d6178426c6f636b48656164657253697a65223a313130302c226d6178436f6c6c61746572616c496e70757473223a332c226d61785478457865637574696f6e556e697473223a7b226d656d6f7279223a31343030303030302c227374657073223a31303030303030303030307d2c226d6178547853697a65223a31363338342c226d617856616c756553697a65223a353030302c226d696e506f6f6c436f7374223a3137303030303030302c226d696e5554784f56616c7565223a6e756c6c2c226d6f6e6574617279457870616e73696f6e223a302e3030332c22706f6f6c506c65646765496e666c75656e6365223a302e332c22706f6f6c5265746972654d617845706f6368223a31382c2270726f746f636f6c56657273696f6e223a7b226d616a6f72223a382c226d696e6f72223a307d2c227374616b65416464726573734465706f736974223a323030303030302c227374616b65506f6f6c4465706f736974223a3530303030303030302c227374616b65506f6f6c5461726765744e756d223a3530302c227472656173757279437574223a302e322c2274784665654669786564223a3135353338312c22747846656550657242797465223a34342c227574786f436f737450657242797465223a343331307d")
)

func TestTransactionBuilder(t *testing.T) {
	const (
		testNetMagic = 203
		ttl          = uint64(28096)
	)

	walletsKeyHashes := []string{
		"d6b67f93ffa4e2651271cc9bcdbdedb2539911266b534d9c163cba21",
		"cba89c7084bf0ce4bf404346b668a7e83c8c9c250d1cafd8d8996e41",
		"79df3577e4c7d7da04872c2182b8d8829d7b477912dbf35d89287c39",
		"2368e8113bd5f32d713751791d29acee9e1b5a425b0454b963b2558b",
		"06b4c7f5254d6395b527ac3de60c1d77194df7431d85fe55ca8f107d",
	}
	walletsFeeKeyHashes := []string{
		"f0f4837b3a306752a2b3e52394168bc7391de3dce11364b723cc55cf",
		"47344d5bd7b2fea56336ba789579705a944760032585ef64084c92db",
		"f01018c1d8da54c2f557679243b09af1c4dd4d9c671512b01fa5f92b",
		"6837232854849427dae7c45892032d7ded136c5beb13c68fda635d87",
		"d215701e2eb17c741b9d306cba553f9fbaaca1e12a5925a065b90fa8",
	}

	policyScriptMultiSig := NewPolicyScript(walletsKeyHashes, len(walletsKeyHashes)*2/3+1)
	policyScriptFeeMultiSig := NewPolicyScript(walletsFeeKeyHashes, len(walletsFeeKeyHashes)*2/3+1)
	cliUtils := NewCliUtils(ResolveCardanoCliBinary(TestNetNetwork))

	multisigPolicyID, err := cliUtils.GetPolicyID(policyScriptMultiSig)
	require.NoError(t, err)

	feeMultisigPolicyID, err := cliUtils.GetPolicyID(policyScriptFeeMultiSig)
	require.NoError(t, err)

	multiSigAddr, err := NewPolicyScriptAddress(TestNetNetwork, multisigPolicyID)
	require.NoError(t, err)

	multiSigFeeAddr, err := NewPolicyScriptAddress(TestNetNetwork, feeMultisigPolicyID)
	require.NoError(t, err)

	type metaDataKey0 struct {
		Type       string `json:"type" cbor:"type"`
		Signers    int    `json:"signers" cbor:"signers"`
		FeeSigners int    `json:"feeSigners" cbor:"feeSigners"`
	}

	type metaDataKey1 struct {
		Company string `json:"comp" cbor:"comp"`
		City    string `json:"city" cbor:"city"`
	}

	metadata := map[uint64]interface{}{
		0: metaDataKey0{
			Type:       "multi",
			Signers:    len(walletsKeyHashes),
			FeeSigners: len(walletsFeeKeyHashes),
		},
		4: metaDataKey1{
			Company: "Ethernal",
			City:    "Novi Sad",
		},
	}
	outputs := []TxOutput{
		{
			Addr:   "addr_test1vqjysa7p4mhu0l25qknwznvj0kghtr29ud7zp732ezwtzec0w8g3u",
			Amount: MinUTxODefaultValue,
		},
	}
	outputsSum := GetOutputsSum(outputs)

	builder, err := NewTxBuilder(ResolveCardanoCliBinary(TestNetNetwork))
	require.NoError(t, err)

	defer builder.Dispose()

	metadataBytes, err := json.Marshal(metadata)
	require.NoError(t, err)

	multiSigInputs := TxInputs{
		Inputs: []TxInput{
			{
				Hash:  "e99a5bde15aa05f24fcc04b7eabc1520d3397283b1ee720de9fe2653abbb0c9f",
				Index: 0,
			},
			{
				Hash:  "d1fd0d772be7741d9bfaf0b037d02d2867a987ccba3e6ba2ee9aa2a861b73145",
				Index: 2,
			},
		},
		Sum: MinUTxODefaultValue*3 - 10,
	}

	multiSigFeeInputs := TxInputs{
		Inputs: []TxInput{
			{
				Hash:  "098236134e0f2077a6434dd9d7727126fa8b3627bcab3ae030a194d46eded73e",
				Index: 0,
			},
		},
		Sum: MinUTxODefaultValue * 2,
	}

	builder.SetTimeToLive(ttl).SetProtocolParameters(protocolParameters)
	builder.SetMetaData(metadataBytes).SetTestNetMagic(testNetMagic)
	builder.AddOutputs(outputs...).AddOutputs(TxOutput{
		Addr: multiSigAddr.String(),
	}).AddOutputs(TxOutput{
		Addr: multiSigFeeAddr.String(),
	})
	builder.AddInputsWithScript(policyScriptMultiSig, multiSigInputs.Inputs...)
	builder.AddInputsWithScript(policyScriptFeeMultiSig, multiSigFeeInputs.Inputs...)

	fee, err := builder.CalculateFee(0)
	require.NoError(t, err)

	builder.SetFee(fee)

	builder.UpdateOutputAmount(-2, multiSigInputs.Sum-outputsSum)
	builder.UpdateOutputAmount(-1, multiSigFeeInputs.Sum-fee)

	txRaw, txHash, err := builder.Build()
	require.NoError(t, err)

	assert.Equal(t, "84a50083825820098236134e0f2077a6434dd9d7727126fa8b3627bcab3ae030a194d46eded73e00825820d1fd0d772be7741d9bfaf0b037d02d2867a987ccba3e6ba2ee9aa2a861b7314502825820e99a5bde15aa05f24fcc04b7eabc1520d3397283b1ee720de9fe2653abbb0c9f00018382581d60244877c1aeefc7fd5405a6e14d927d91758d45e37c20fa2ac89cb1671a000f424082581d704aaad0f0626a8ce7b097497e542055b6520842ade881f980e002ae661a001e847682581d703ea4c4aef89a27f111e78464d7d6717b099f85ce27109ee9e5fbddec1a001a79bf021a00040ac103196dc0075820802e4d6f15ce98826886a5451e94855e77aae779cb341d3aab1e3bae4fb2f78da10182830304858200581c47344d5bd7b2fea56336ba789579705a944760032585ef64084c92db8200581c6837232854849427dae7c45892032d7ded136c5beb13c68fda635d878200581cd215701e2eb17c741b9d306cba553f9fbaaca1e12a5925a065b90fa88200581cf01018c1d8da54c2f557679243b09af1c4dd4d9c671512b01fa5f92b8200581cf0f4837b3a306752a2b3e52394168bc7391de3dce11364b723cc55cf830304858200581c06b4c7f5254d6395b527ac3de60c1d77194df7431d85fe55ca8f107d8200581c2368e8113bd5f32d713751791d29acee9e1b5a425b0454b963b2558b8200581c79df3577e4c7d7da04872c2182b8d8829d7b477912dbf35d89287c398200581ccba89c7084bf0ce4bf404346b668a7e83c8c9c250d1cafd8d8996e418200581cd6b67f93ffa4e2651271cc9bcdbdedb2539911266b534d9c163cba21f5d90103a100a200a36a6665655369676e65727305677369676e657273056474797065656d756c746904a26463697479684e6f76692053616464636f6d706845746865726e616c", hex.EncodeToString(txRaw))
	assert.Equal(t, "1b9298c51f4dc05c04cae37104124cfb76e9f98f04a7f6b8179cfe02913152ec", txHash)
}

func Test_TxBuilder_UpdateOutputAmountAndRemoveOutput(t *testing.T) {
	builder, err := NewTxBuilder(ResolveCardanoCliBinary(TestNetNetwork))
	require.NoError(t, err)

	defer builder.Dispose()

	builder.AddOutputs(
		TxOutput{Addr: "0x1"},
		TxOutput{Addr: "0x2"},
		TxOutput{Addr: "0x3"},
		TxOutput{Addr: "0x4"},
	)

	require.Len(t, builder.outputs, 4)
	assert.Equal(t, uint64(0), builder.outputs[2].Amount)
	assert.Equal(t, uint64(0), builder.outputs[3].Amount)

	builder.UpdateOutputAmount(2, 200)
	builder.UpdateOutputAmount(-1, 500)

	assert.Equal(t, uint64(200), builder.outputs[2].Amount)
	assert.Equal(t, "0x3", builder.outputs[2].Addr)
	assert.Equal(t, uint64(500), builder.outputs[3].Amount)
	assert.Equal(t, "0x4", builder.outputs[3].Addr)

	builder.RemoveOutput(1)

	require.Len(t, builder.outputs, 3)
	assert.Equal(t, "0x1", builder.outputs[0].Addr)
	assert.Equal(t, uint64(0), builder.outputs[0].Amount)
	assert.Equal(t, "0x3", builder.outputs[1].Addr)
	assert.Equal(t, uint64(200), builder.outputs[1].Amount)
	assert.Equal(t, "0x4", builder.outputs[2].Addr)
	assert.Equal(t, uint64(500), builder.outputs[2].Amount)

	builder.RemoveOutput(0)

	require.Len(t, builder.outputs, 2)
	assert.Equal(t, "0x3", builder.outputs[0].Addr)
	assert.Equal(t, uint64(200), builder.outputs[0].Amount)
	assert.Equal(t, "0x4", builder.outputs[1].Addr)
	assert.Equal(t, uint64(500), builder.outputs[1].Amount)

	builder.RemoveOutput(1)

	require.Len(t, builder.outputs, 1)
	assert.Equal(t, "0x3", builder.outputs[0].Addr)
	assert.Equal(t, uint64(200), builder.outputs[0].Amount)

	builder.RemoveOutput(0)

	require.Len(t, builder.outputs, 0)
}

func Test_TxBuilder_CheckOutputs(t *testing.T) {
	b, err := NewTxBuilder(ResolveCardanoCliBinary(TestNetNetwork))
	require.NoError(t, err)

	defer b.Dispose()

	b.AddOutputs(TxOutput{
		Addr:   "x1",
		Amount: MinUTxODefaultValue,
	}, TxOutput{
		Addr:   "x2",
		Amount: MinUTxODefaultValue + 1,
	})

	require.NoError(t, b.CheckOutputs())

	b.AddOutputs(TxOutput{
		Addr:   "x3",
		Amount: MinUTxODefaultValue - 1,
	}, TxOutput{
		Addr:   "x4",
		Amount: MinUTxODefaultValue - 2,
	})

	err = b.CheckOutputs()
	require.Error(t, err)

	require.True(t, strings.Contains(err.Error(), "output (2, x3) has insufficient amount 999999"))
	require.True(t, strings.Contains(err.Error(), "output (3, x4) has insufficient amount 999998"))

	b.SetMinOutputAmount(MinUTxODefaultValue - 1)

	err = b.CheckOutputs()
	require.Error(t, err)

	require.False(t, strings.Contains(err.Error(), "output (2, x3) has insufficient amount 999999"))
	require.True(t, strings.Contains(err.Error(), "output (3, x4) has insufficient amount 999998"))

	b.SetMinOutputAmount(MinUTxODefaultValue - 2)

	require.NoError(t, b.CheckOutputs())
}
