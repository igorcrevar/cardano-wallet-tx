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
	socketPath   = "/home/bbs/Apps/card/node.socket"
	testNetMagic = uint(2) //1097911063
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
			Amount: cardanowallet.MinUtxoValue,
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
			Amount: cardanowallet.MinUtxoValue,
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

func submitTx(txSigned []byte, hash string, submitter cardanowallet.ITxSubmitter) error {
	if err := submitter.SubmitTx(txSigned); err != nil {
		return err
	}

	fmt.Println("transaction has been submitted", hash)

	return nil
}

func main() {
	txDataRetriever, err := cardanowallet.NewTxDataRetrieverCli(testNetMagic, socketPath)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	defer txDataRetriever.Dispose()

	multiSigTx, multiSigTxHash, err := createMultiSigTx(txDataRetriever, 3, 2)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	if err := submitTx(multiSigTx, multiSigTxHash, txDataRetriever); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	sigTx, txHash, err := createTx(txDataRetriever)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	if err := submitTx(sigTx, txHash, txDataRetriever); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
