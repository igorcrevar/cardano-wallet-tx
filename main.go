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

func createTx(dataRetriever cardanowallet.ITxDataRetriever) ([]byte, string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, "", err
	}

	wallet := cardanowallet.NewWallet(path.Join(currentUser.HomeDir, "cardano_wallet"), testNetMagic)

	err = wallet.Create(false)
	if err != nil {
		return nil, "", err
	}

	err = wallet.Load()
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

func createMultiSigTx(dataRetriever cardanowallet.ITxDataRetriever, cnt int, atLeast int) ([]byte, string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, "", err
	}

	wallets := make([]*cardanowallet.Wallet, cnt)

	for i := 0; i < cnt; i++ {
		wallets[i] = cardanowallet.NewWallet(path.Join(currentUser.HomeDir, fmt.Sprintf("cardano_wallet_%d", i+1)), testNetMagic)

		err := wallets[i].Create(false)
		if err != nil {
			return nil, "", err
		}

		err = wallets[i].Load()
		if err != nil {
			return nil, "", err
		}
	}

	keyHashes := make([]string, len(wallets))
	for i, w := range wallets {
		keyHashes[i] = w.GetKeyHash()
	}

	multisigAddr := cardanowallet.NewMultiSigAddress(path.Join(currentUser.HomeDir, "cardano_multisig"), keyHashes, testNetMagic, atLeast)
	err = multisigAddr.Create(false)
	if err != nil {
		return nil, "", err
	}

	err = multisigAddr.Load()
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
	txProviderBF, err := createProvider("blockfrost")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	defer txProviderBF.Dispose()

	multiSigTx, multiSigTxHash, err := createMultiSigTx(txProviderBF, 3, 2)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	if err := txProviderBF.SubmitTx(multiSigTx); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("transaction has been submitted", multiSigTxHash)

	sigTx, txHash, err := createTx(txProviderBF)
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
