package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	cardanowallet "github.com/igorcrevar/cardano-wallet-tx/core"
)

const (
	socketPath              = "/home/bbs/Apps/card/node.socket"
	testNetMagic            = uint(2)
	ogmiosUrl               = "http://localhost:1337"
	blockfrostUrl           = "https://cardano-preview.blockfrost.io/api/v0"
	blockfrostProjectApiKey = ""
	potentialFee            = uint64(300_000)
	providerName            = "blockfrost"
	receiverAddr            = "addr_test1wz4k6frsfd9q98rya6zjxtpcmzn83pwc8uyl9yqw25p8qqcx3e0c0"
	receiverMultisigAddr    = "addr_test1vrhltc3r25sha3khwrpkdqqscfmplgyx8tap96tvl79zypgr4mc9f"
)

func getSplitedStr(s string, mxlen int) (res []string) {
	for i := 0; i < len(s); i += mxlen {
		end := i + mxlen
		if end > len(s) {
			end = len(s)
		}

		res = append(res, s[i:end])
	}

	return res
}

func getKeyHashes(wallets []cardanowallet.IWallet) []string {
	keyHashes := make([]string, len(wallets))
	for i, w := range wallets {
		keyHashes[i], _ = cardanowallet.GetKeyHash(w.GetVerificationKey())
	}

	return keyHashes
}

func createTx(
	cardanoCliBinary string,
	txProvider cardanowallet.ITxProvider, wallet cardanowallet.IWallet,
	testNetMagic uint, potentialFee uint64, receiverAddr string,
) ([]byte, string, error) {
	enterptiseAddress, err := cardanowallet.NewEnterpriseAddress(
		cardanowallet.TestNetNetwork, wallet.GetVerificationKey())
	if err != nil {
		return nil, "", err
	}

	address := enterptiseAddress.String()

	fmt.Println("address =", address)

	metadata := map[string]interface{}{
		"0": map[string]interface{}{
			"type": "single",
		},
	}
	outputs := []cardanowallet.TxOutput{
		{
			Addr:   receiverAddr,
			Amount: cardanowallet.MinUTxODefaultValue,
		},
	}
	outputsSum := cardanowallet.GetOutputsSum(outputs)

	builder, err := cardanowallet.NewTxBuilder(cardanoCliBinary)
	if err != nil {
		return nil, "", err
	}

	defer builder.Dispose()

	if err := builder.SetProtocolParametersAndTTL(context.Background(), txProvider, 0); err != nil {
		return nil, "", err
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, "", err
	}

	inputs, err := cardanowallet.GetUTXOsForAmount(
		context.Background(), txProvider, address,
		outputsSum+potentialFee+cardanowallet.MinUTxODefaultValue,
		outputsSum+potentialFee+cardanowallet.MinUTxODefaultValue)
	if err != nil {
		return nil, "", err
	}

	builder.SetMetaData(metadataBytes).SetTestNetMagic(testNetMagic)
	builder.AddOutputs(outputs...).AddOutputs(cardanowallet.TxOutput{
		Addr: address,
	})
	builder.AddInputs(inputs.Inputs...)

	fee, err := builder.CalculateFee(1)
	if err != nil {
		return nil, "", err
	}

	builder.SetFee(fee)

	builder.UpdateOutputAmount(-1, inputs.Sum-outputsSum-fee)

	txRaw, txHash, err := builder.Build()
	if err != nil {
		return nil, "", err
	}

	witness, err := cardanowallet.CreateTxWitness(txHash, wallet)
	if err != nil {
		return nil, "", err
	}

	txSigned, err := builder.AssembleTxWitnesses(txRaw, [][]byte{witness})
	if err != nil {
		return nil, "", err
	}

	return txSigned, txHash, nil
}

func createMultiSigTx(
	cardanoCliBinary string, txProvider cardanowallet.ITxProvider,
	signers []cardanowallet.IWallet, feeSigners []cardanowallet.IWallet,
	testNetMagic uint, potentialFee uint64, receiverAddr string,
) ([]byte, string, error) {
	policyScriptMultiSig := cardanowallet.NewPolicyScript(getKeyHashes(signers), len(signers)*2/3+1)
	policyScriptFeeMultiSig := cardanowallet.NewPolicyScript(getKeyHashes(feeSigners), len(signers)*2/3+1)
	cliUtils := cardanowallet.NewCliUtils(cardanoCliBinary)

	multisigPolicyID, err := cliUtils.GetPolicyID(policyScriptMultiSig)
	if err != nil {
		return nil, "", err
	}

	feeMultisigPolicyID, err := cliUtils.GetPolicyID(policyScriptFeeMultiSig)
	if err != nil {
		return nil, "", err
	}

	multiSigAddr, err := cardanowallet.NewPolicyScriptAddress(cardanowallet.TestNetNetwork, multisigPolicyID)
	if err != nil {
		return nil, "", err
	}

	multiSigFeeAddr, err := cardanowallet.NewPolicyScriptAddress(cardanowallet.TestNetNetwork, feeMultisigPolicyID)
	if err != nil {
		return nil, "", err
	}

	fmt.Println("multi-address sig =", multiSigAddr, " multi-address fee =", multiSigFeeAddr)

	metadata := map[string]interface{}{
		"0": map[string]interface{}{
			"type": "multi",
		},
		"1": map[string]interface{}{
			"destinationChainId": "vector",
			"senderAddr": getSplitedStr(
				"addr_test1qzf762fxqdyc79d3zzjplc57z6dpnrkygq5960tjguh683n3evd0dmxh9k7yzdxvqv9279nmkkwhx4m5wkj006a44nyscj7w9r",
				40,
			),
			"transactions": []map[string]interface{}{
				{
					"address": getSplitedStr(
						"addr_test1wp9g0wy5f58ruvt3d8cf2v3hylna934p99y0pwv8a4pm2wcx9he4s",
						40,
					),
					"amount": 1100000,
				},
				{
					"address": getSplitedStr(
						"addr_test1qqpszngm7jx9seaw9pr6pql7hey62an4k8lk6uncmagfd6wtn8ktl44rmpwahjg9w349v2tcf9zvujxd442qr3j24fms3fr687",
						40,
					),
					"amount": 1000000,
				},
			},
			"type": "bridgingRequest",
		},
	}
	outputs := []cardanowallet.TxOutput{
		{
			Addr:   receiverAddr,
			Amount: cardanowallet.MinUTxODefaultValue,
		},
	}
	outputsSum := cardanowallet.GetOutputsSum(outputs)

	builder, err := cardanowallet.NewTxBuilder(cardanoCliBinary)
	if err != nil {
		return nil, "", err
	}

	defer builder.Dispose()

	if err := builder.SetProtocolParametersAndTTL(context.Background(), txProvider, 0); err != nil {
		return nil, "", err
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, "", err
	}

	multiSigInputs, err := cardanowallet.GetUTXOsForAmount(
		context.Background(), txProvider, multiSigAddr.String(),
		outputsSum, outputsSum+cardanowallet.MinUTxODefaultValue)
	if err != nil {
		return nil, "", err
	}

	multiSigFeeInputs, err := cardanowallet.GetUTXOsForAmount(
		context.Background(), txProvider, multiSigFeeAddr.String(),
		potentialFee, potentialFee+cardanowallet.MinUTxODefaultValue)
	if err != nil {
		return nil, "", err
	}

	builder.SetMetaData(metadataBytes).SetTestNetMagic(testNetMagic)
	builder.AddOutputs(outputs...)
	builder.AddInputsWithScript(policyScriptMultiSig, multiSigInputs.Inputs...)
	builder.AddInputsWithScript(policyScriptFeeMultiSig, multiSigFeeInputs.Inputs...)

	if change := multiSigInputs.Sum - outputsSum; change > 0 {
		builder.AddOutputs(cardanowallet.TxOutput{
			Addr: multiSigAddr.String(), Amount: change,
		})
	}

	builder.AddOutputs(cardanowallet.TxOutput{Addr: multiSigFeeAddr.String()})

	fee, err := builder.CalculateFee(0)
	if err != nil {
		return nil, "", err
	}

	builder.SetFee(fee)

	if change := multiSigFeeInputs.Sum - fee; change > 0 {
		builder.UpdateOutputAmount(-1, change)
	} else {
		builder.RemoveOutput(-1)
	}

	txRaw, txHash, err := builder.Build()
	if err != nil {
		return nil, "", err
	}

	witnesses := make([][]byte, len(signers)+len(feeSigners))
	for i, w := range signers {
		witnesses[i], err = cardanowallet.CreateTxWitness(txHash, w)
		if err != nil {
			return nil, "", err
		}

		if err := cardanowallet.VerifyWitness(txHash, witnesses[i]); err != nil {
			return nil, "", err
		}
	}

	for i, w := range feeSigners {
		witnesses[i+len(signers)], err = cardanowallet.CreateTxWitness(txHash, w)
		if err != nil {
			return nil, "", err
		}

		if err := cardanowallet.VerifyWitness(txHash, witnesses[i+len(signers)]); err != nil {
			return nil, "", err
		}
	}

	txSigned, err := builder.AssembleTxWitnesses(txRaw, witnesses)
	if err != nil {
		return nil, "", err
	}

	return txSigned, txHash, nil
}

func createProvider(name string, cardanoCliBinary string) (cardanowallet.ITxProvider, error) {
	switch name {
	case "blockfrost":
		return cardanowallet.NewTxProviderBlockFrost(blockfrostUrl, blockfrostProjectApiKey), nil
	case "ogmios":
		return cardanowallet.NewTxProviderOgmios(ogmiosUrl), nil
	default:
		return cardanowallet.NewTxProviderCli(testNetMagic, socketPath, cardanoCliBinary)
	}
}

func loadWallets() ([]cardanowallet.IWallet, error) {
	verificationKeys := []string{
		"582068fc463c29900b00122423c7e6a39469987786314e07a5e7f5eae76a5fe671bf",
		"58209a9cefaa636d75dffa3a3a5ab446a191beac92b09ac82da513640e8e35935202",
		"5820839c3bd7397f35bf55d63c0bcb3880c95ffd91e8c3bfc405a60f6c605a7a40f2",
		"582063e95162d952d2fbc5240457750e1c13bfb4a5e3d9a96bf048b90bfe08b13de6",
		"5820030083fd0293fc6ed8d76faf02365617066f37ad6a6d6047b801e2865914d900",
		"5820ad5a1761213fb82a859333d78d66cf0d9dc56e413a26fe3108b5f21bac1d5fa4",
	}
	signingKeys := []string{
		"58201825bce09711e1563fc1702587da6892d1d869894386323bd4378ea5e3d6cba0",
		"5820ccdae0d1cd3fa9be16a497941acff33b9aa20bdbf2f9aa5715942d152988e083",
		"582094bfc7d65a5d936e7b527c93ea6bf75de51029290b1ef8c8877bffe070398b40",
		"58204cd84bf321e70ab223fbdbfe5eba249a5249bd9becbeb82109d45e56c9c610a9",
		"58208fcc8cac6b7fedf4c30aed170633df487642cb22f7e8615684e2b98e367fcaa3",
		"582058fb35da120c65855ad691dadf5681a2e4fc62e9dcda0d0774ff6fdc463a679a",
	}

	wallets := make([]cardanowallet.IWallet, len(verificationKeys))
	for i := range verificationKeys {
		signingKey, err := cardanowallet.GetKeyBytes(signingKeys[i])
		if err != nil {
			return nil, err
		}

		verificationKey, err := cardanowallet.GetKeyBytes(verificationKeys[i])
		if err != nil {
			return nil, err
		}

		wallets[i] = cardanowallet.NewWallet(verificationKey, signingKey)
	}

	return wallets, nil
}

func submitTx(
	ctx context.Context, txProvider cardanowallet.ITxProvider, txRaw []byte, txHash string, addr string,
) error {
	utxo, err := txProvider.GetUtxos(ctx, addr)
	if err != nil {
		return err
	}

	expectedAtLeast := cardanowallet.GetUtxosSum(utxo) + cardanowallet.MinUTxODefaultValue

	if err := txProvider.SubmitTx(context.Background(), txRaw); err != nil {
		return err
	}

	fmt.Println("transaction has been submitted. hash =", txHash)

	err = cardanowallet.WaitForAmount(ctx, txProvider, addr, func(val uint64) bool {
		return val >= expectedAtLeast
	}, 60, time.Second*5)
	if err != nil {
		return err
	}

	fmt.Println("transaction has been included in block. hash =", txHash)

	return nil
}

func main() {
	cardanoCliBinary := cardanowallet.ResolveCardanoCliBinary(cardanowallet.TestNetNetwork)

	wallets, err := loadWallets()
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	txProvider, err := createProvider(providerName, cardanoCliBinary)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	defer txProvider.Dispose()

	_, _ = txProvider.GetTip(context.Background())

	sigTx, txHash, err := createTx(
		cardanoCliBinary, txProvider, wallets[0], testNetMagic, potentialFee, receiverAddr)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	if err := submitTx(context.Background(), txProvider, sigTx, txHash, receiverAddr); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	multiSigTx, multiSigTxHash, err := createMultiSigTx(
		cardanoCliBinary, txProvider, wallets[:3], wallets[3:], testNetMagic, potentialFee, receiverMultisigAddr)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	if err := submitTx(context.Background(), txProvider, multiSigTx, multiSigTxHash, receiverMultisigAddr); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
