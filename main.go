package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path"

	cardanowallet "github.com/igorcrevar/cardano-wallet-tx/core"
	cardanowallethelper "github.com/igorcrevar/cardano-wallet-tx/helper"
)

const (
	socketPath              = "/home/bbs/Apps/card/node.socket"
	testNetMagic            = uint(2)
	blockfrostUrl           = "https://cardano-preview.blockfrost.io/api/v0"
	blockfrostProjectApiKey = "YOUR_PROJECT_ID"
)

func createTx(dataRetriever cardanowallet.ITxDataRetriever, keyDirectory string) ([]byte, string, error) {
	walletBuilder := cardanowallet.NewWalletBuilder(path.Join(keyDirectory, "cardano_wallet"), testNetMagic)

	if err := walletBuilder.Create(false); err != nil {
		return nil, "", err
	}

	wallet, err := walletBuilder.Load()
	if err != nil {
		return nil, "", err
	}

	fmt.Println("Address =", wallet.GetAddress())

	metadata, err := json.Marshal(map[string]interface{}{
		"0": map[string]interface{}{
			"type": "single",
		},
	})
	if err != nil {
		return nil, "", err
	}

	return cardanowallethelper.PrepareSignedTx(dataRetriever, wallet, testNetMagic, []cardanowallet.TxOutput{
		{
			Addr:   "addr_test1vqjysa7p4mhu0l25qknwznvj0kghtr29ud7zp732ezwtzec0w8g3u",
			Amount: cardanowallet.MinUTxODefaultValue,
		},
	}, metadata)
}

func createMultiSigTx(dataRetriever cardanowallet.ITxDataRetriever, cnt int, atLeast int, keyDirectory string) ([]byte, string, error) {
	wallets := make([]*cardanowallet.Wallet, cnt)
	keyHashes := make([]string, len(wallets))

	for i := 0; i < cnt; i++ {
		walletBuilder := cardanowallet.NewWalletBuilder(path.Join(keyDirectory, fmt.Sprintf("cardano_wallet_%d", i+1)), testNetMagic)

		err := walletBuilder.Create(false)
		if err != nil {
			return nil, "", err
		}

		wallets[i], err = walletBuilder.Load()
		if err != nil {
			return nil, "", err
		}

		keyHashes[i] = wallets[i].GetKeyHash()
	}

	policyScript, err := cardanowallet.NewPolicyScript(keyHashes, atLeast)
	if err != nil {
		return nil, "", err
	}

	multisigAddr, err := policyScript.CreateMultiSigAddress(testNetMagic)
	if err != nil {
		return nil, "", err
	}

	fmt.Println("Multi-address =", multisigAddr.GetAddress())

	metadata, err := json.Marshal(map[string]interface{}{
		"0": map[string]interface{}{
			"type":    "multi",
			"atleast": atLeast,
			"max":     cnt,
		},
	})
	if err != nil {
		return nil, "", err
	}

	txRaw, hash, err := cardanowallethelper.PrepareMultiSigTx(dataRetriever, multisigAddr, testNetMagic, []cardanowallet.TxOutput{
		{
			Addr:   "addr_test1vqjysa7p4mhu0l25qknwznvj0kghtr29ud7zp732ezwtzec0w8g3u",
			Amount: cardanowallet.MinUTxODefaultValue,
		},
	}, metadata)
	if err != nil {
		return nil, "", err
	}

	txSigned, err := cardanowallethelper.AssemblyAllWitnesses(txRaw, wallets[:atLeast])
	if err != nil {
		return nil, "", err
	}

	return txSigned, hash, nil
}

func createProvider(name string) (cardanowallet.ITxProvider, error) {
	switch name {
	case "blockfrost":
		return cardanowallet.NewTxProviderBlockFrost(blockfrostUrl, blockfrostProjectApiKey)
	default:
		return cardanowallet.NewTxProviderCli(testNetMagic, socketPath)
	}
}

func main() {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	txProviderBF, err := createProvider("blockfrost")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	defer txProviderBF.Dispose()

	multiSigTx, multiSigTxHash, err := createMultiSigTx(txProviderBF, 3, 2, currentUser.HomeDir)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	if err := txProviderBF.SubmitTx(multiSigTx); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("transaction has been submitted", multiSigTxHash)

	sigTx, txHash, err := createTx(txProviderBF, currentUser.HomeDir)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	if err := txProviderBF.SubmitTx(sigTx); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("transaction has been submitted", txHash)
}
