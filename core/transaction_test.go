package core

import (
	"encoding/hex"
	"encoding/json"
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
		ttl          = 28096
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

	policyScriptMultiSig, err := NewPolicyScript(walletsKeyHashes, len(walletsKeyHashes)*2/3+1)
	require.NoError(t, err)

	policyScriptFeeMultiSig, err := NewPolicyScript(walletsFeeKeyHashes, len(walletsFeeKeyHashes)*2/3+1)
	require.NoError(t, err)

	multiSigAddr, err := policyScriptMultiSig.CreateMultiSigAddress(testNetMagic)
	require.NoError(t, err)

	multiSigFeeAddr, err := policyScriptFeeMultiSig.CreateMultiSigAddress(testNetMagic)
	require.NoError(t, err)

	metadata := map[string]interface{}{
		"0": map[string]interface{}{
			"type":       "multi",
			"signers":    len(walletsKeyHashes),
			"feeSigners": len(walletsFeeKeyHashes),
		},
	}
	outputs := []TxOutput{
		{
			Addr:   "addr_test1vqjysa7p4mhu0l25qknwznvj0kghtr29ud7zp732ezwtzec0w8g3u",
			Amount: MinUTxODefaultValue,
		},
	}
	outputsSum := GetOutputsSum(outputs)

	builder, err := NewTxBuilder()
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
		Sum: MinUTxODefaultValue + 20,
	}

	multiSigFeeInputs := TxInputs{
		Inputs: []TxInput{
			{
				Hash:  "098236134e0f2077a6434dd9d7727126fa8b3627bcab3ae030a194d46eded73e",
				Index: 0,
			},
		},
		Sum: MinUTxODefaultValue,
	}

	builder.SetTimeToLive(ttl).SetProtocolParameters(protocolParameters)
	builder.SetMetaData(metadataBytes).SetTestNetMagic(testNetMagic)
	builder.AddOutputs(outputs...).AddOutputs(TxOutput{
		Addr: multiSigAddr,
	}).AddOutputs(TxOutput{
		Addr: multiSigFeeAddr,
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

	assert.Equal(t, "84a50083825820098236134e0f2077a6434dd9d7727126fa8b3627bcab3ae030a194d46eded73e00825820d1fd0d772be7741d9bfaf0b037d02d2867a987ccba3e6ba2ee9aa2a861b7314502825820e99a5bde15aa05f24fcc04b7eabc1520d3397283b1ee720de9fe2653abbb0c9f00018382581d60244877c1aeefc7fd5405a6e14d927d91758d45e37c20fa2ac89cb1671a000f424082581d700c25e4ff24cfa0dfebcec382095161271dc9bb744ca4149ec604dc991482581d70a5caf9ce4bed09c794ee87bddb6505822db5bd476a4f61e0cd4074a21a000b3ca7021a0004059903196dc0075820981cfc0e6c2095e5e630840d9e8c078a2d13677ac5f97d8945d8d0c46a53047ca10182830304858200581cd6b67f93ffa4e2651271cc9bcdbdedb2539911266b534d9c163cba218200581ccba89c7084bf0ce4bf404346b668a7e83c8c9c250d1cafd8d8996e418200581c79df3577e4c7d7da04872c2182b8d8829d7b477912dbf35d89287c398200581c2368e8113bd5f32d713751791d29acee9e1b5a425b0454b963b2558b8200581c06b4c7f5254d6395b527ac3de60c1d77194df7431d85fe55ca8f107d830304858200581cf0f4837b3a306752a2b3e52394168bc7391de3dce11364b723cc55cf8200581c47344d5bd7b2fea56336ba789579705a944760032585ef64084c92db8200581cf01018c1d8da54c2f557679243b09af1c4dd4d9c671512b01fa5f92b8200581c6837232854849427dae7c45892032d7ded136c5beb13c68fda635d878200581cd215701e2eb17c741b9d306cba553f9fbaaca1e12a5925a065b90fa8f5d90103a100a100a36a6665655369676e65727305677369676e657273056474797065656d756c7469", hex.EncodeToString(txRaw))
	assert.Equal(t, "371837ccfdfbf3ecdcc0fdee26c2e349aa988402bfdecd130a852d799d07bb04", txHash)
}
