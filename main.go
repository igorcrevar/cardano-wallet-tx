package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path"
	"time"

	cardanowallet "github.com/igorcrevar/cardano-wallet-tx/core"
)

const (
	socketPath              = "/home/bbs/Apps/card/node.socket"
	testNetMagic            = uint(2)
	blockfrostUrl           = "https://cardano-preview.blockfrost.io/api/v0"
	blockfrostProjectApiKey = ""
	potentialFee            = uint64(300_000)
	providerName            = "blockfrost"
)

func getKeyHashes(wallets []cardanowallet.IWallet) []string {
	keyHashes := make([]string, len(wallets))
	for i, w := range wallets {
		keyHashes[i] = w.GetKeyHash()
	}

	return keyHashes
}

func createWallets(walletMngr cardanowallet.IWalletManager, keyDirectory string, cnt int) ([]cardanowallet.IWallet, error) {
	wallets := make([]cardanowallet.IWallet, cnt)

	for i := 0; i < cnt; i++ {
		fpath := fmt.Sprintf("%s%d", keyDirectory, i+1)

		wallet, err := walletMngr.Create(fpath, false)
		if err != nil {
			return nil, err
		}

		wallets[i] = wallet
	}

	return wallets, nil
}

func createTx(txProvider cardanowallet.ITxProvider,
	wallet cardanowallet.IWallet, testNetMagic uint, potentialFee uint64) ([]byte, string, error) {
	address, _, err := cardanowallet.GetWalletAddress(wallet, testNetMagic)
	if err != nil {
		return nil, "", err
	}

	fmt.Println("address =", address)

	metadata := map[string]interface{}{
		"0": map[string]interface{}{
			"type": "single",
		},
	}
	outputs := []cardanowallet.TxOutput{
		{
			Addr:   "addr_test1vqjysa7p4mhu0l25qknwznvj0kghtr29ud7zp732ezwtzec0w8g3u",
			Amount: cardanowallet.MinUTxODefaultValue,
		},
	}
	outputsSum := cardanowallet.GetOutputsSum(outputs)

	builder, err := cardanowallet.NewTxBuilder()
	if err != nil {
		return nil, "", err
	}

	defer builder.Dispose()

	if err := builder.SetProtocolParametersAndTTL(txProvider, 0); err != nil {
		return nil, "", err
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, "", err
	}

	inputs, err := cardanowallet.GetUTXOsForAmount(txProvider, address, outputsSum+potentialFee)
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

	txSigned, err := cardanowallet.SignTx(txRaw, txHash, wallet)
	if err != nil {
		return nil, "", err
	}

	return txSigned, txHash, nil
}

func createMultiSigTx(
	txProvider cardanowallet.ITxProvider,
	signers []cardanowallet.IWallet,
	feeSigners []cardanowallet.IWallet,
	testNetMagic uint,
	potentialFee uint64) ([]byte, string, error) {
	policyScriptMultiSig, err := cardanowallet.NewPolicyScript(getKeyHashes(signers), len(signers)*2/3+1)
	if err != nil {
		return nil, "", err
	}

	policyScriptFeeMultiSig, err := cardanowallet.NewPolicyScript(getKeyHashes(feeSigners), len(signers)*2/3+1)
	if err != nil {
		return nil, "", err
	}

	multiSigAddr, err := policyScriptMultiSig.CreateMultiSigAddress(testNetMagic)
	if err != nil {
		return nil, "", err
	}

	multiSigFeeAddr, err := policyScriptFeeMultiSig.CreateMultiSigAddress(testNetMagic)
	if err != nil {
		return nil, "", err
	}

	fmt.Println("multi-address sig =", multiSigAddr, " multi-address fee =", multiSigFeeAddr)

	metadata := map[string]interface{}{
		"0": map[string]interface{}{
			"type":       "multi",
			"signers":    len(signers),
			"feeSigners": len(feeSigners),
		},
	}
	outputs := []cardanowallet.TxOutput{
		{
			Addr:   "addr_test1vqjysa7p4mhu0l25qknwznvj0kghtr29ud7zp732ezwtzec0w8g3u",
			Amount: cardanowallet.MinUTxODefaultValue,
		},
	}
	outputsSum := cardanowallet.GetOutputsSum(outputs)

	builder, err := cardanowallet.NewTxBuilder()
	if err != nil {
		return nil, "", err
	}

	defer builder.Dispose()

	if err := builder.SetProtocolParametersAndTTL(txProvider, 0); err != nil {
		return nil, "", err
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, "", err
	}

	multiSigInputs, err := cardanowallet.GetUTXOsForAmount(txProvider, multiSigAddr, cardanowallet.MinUTxODefaultValue)
	if err != nil {
		return nil, "", err
	}

	multiSigFeeInputs, err := cardanowallet.GetUTXOsForAmount(txProvider, multiSigFeeAddr, potentialFee)
	if err != nil {
		return nil, "", err
	}

	builder.SetMetaData(metadataBytes).SetTestNetMagic(testNetMagic)
	builder.AddOutputs(outputs...).AddOutputs(cardanowallet.TxOutput{
		Addr: multiSigAddr,
	}).AddOutputs(cardanowallet.TxOutput{
		Addr: multiSigFeeAddr,
	})
	builder.AddInputsWithScript(policyScriptMultiSig, multiSigInputs.Inputs...)
	builder.AddInputsWithScript(policyScriptFeeMultiSig, multiSigFeeInputs.Inputs...)

	fee, err := builder.CalculateFee(0)
	if err != nil {
		return nil, "", err
	}

	builder.SetFee(fee)

	builder.UpdateOutputAmount(-2, multiSigInputs.Sum-outputsSum)
	builder.UpdateOutputAmount(-1, multiSigFeeInputs.Sum-fee)

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

	txSigned, err := cardanowallet.AssembleTxWitnesses(txRaw, witnesses)
	if err != nil {
		return nil, "", err
	}

	return txSigned, txHash, nil
}

func createProvider(name string) (cardanowallet.ITxProvider, error) {
	switch name {
	case "blockfrost":
		return cardanowallet.NewTxProviderBlockFrost(blockfrostUrl, blockfrostProjectApiKey)
	default:
		return cardanowallet.NewTxProviderCli(testNetMagic, socketPath)
	}
}

func createWalletMngr(isStake bool) cardanowallet.IWalletManager {
	if isStake {
		return cardanowallet.NewStakeWalletManager()
	}

	return cardanowallet.NewWalletManager()
}

func main() {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	wallets, err := createWallets(createWalletMngr(false), path.Join(currentUser.HomeDir, "cardano", "wallet_stake_"), 6)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	txProviderBF, err := createProvider(providerName)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	defer txProviderBF.Dispose()

	txRetriever := txProviderBF.(cardanowallet.ITxRetriever)

	multiSigTx, multiSigTxHash, err := createMultiSigTx(txProviderBF, wallets[:3], wallets[3:], testNetMagic, potentialFee)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	if err := txProviderBF.SubmitTx(multiSigTx); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	txData, err := cardanowallet.WaitForTransaction(context.Background(), txRetriever, multiSigTxHash, 100, time.Second*2)
	if err != nil {
		fmt.Printf("error waiting for multisig transaction: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("transaction has been submitted. hash =", multiSigTxHash, " block =", txData["block"])

	sigTx, txHash, err := createTx(txProviderBF, wallets[0], testNetMagic, potentialFee)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	if err := txProviderBF.SubmitTx(sigTx); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	txData, err = cardanowallet.WaitForTransaction(context.Background(), txRetriever, txHash, 100, time.Second*2)
	if err != nil {
		fmt.Printf("error waiting for transaction: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("transaction has been submitted. hash =", txHash, " block =", txData["block"])
}
