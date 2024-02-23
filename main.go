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

func sendTx() error {
	currentUser, err := user.Current()
	if err != nil {
		return err
	}

	wallet := cardanowallet.NewWallet(path.Join(currentUser.HomeDir, "cardano_wallet"), testNetMagic)

	err = wallet.Create(false)
	if err != nil {
		return err
	}

	err = wallet.Load()
	if err != nil {
		return err
	}

	fmt.Println(wallet.GetAddress())
	fmt.Println(wallet.GetSigningKey())
	fmt.Println(wallet.GetVerificationKey())
	fmt.Println(wallet.GetKeyHash())

	metadata, err := json.Marshal(map[string]interface{}{
		"0": map[string]interface{}{
			"who":    "pera",
			"what":   "taba-kera",
			"health": 20,
		},
	})
	if err != nil {
		return err
	}

	hash, err := cardanowallethelper.SendTx(wallet, testNetMagic, socketPath, []cardanowallet.TxOutput{
		{
			Addr:   "addr_test1vqjysa7p4mhu0l25qknwznvj0kghtr29ud7zp732ezwtzec0w8g3u",
			Amount: cardanowallet.MinUtxoValue,
		},
	}, metadata)
	if err != nil {
		return err
	}

	fmt.Println("transaction has been submitted", hash)

	return nil
}

func sendMultiSigTx(cnt int, atLeast int) error {
	currentUser, err := user.Current()
	if err != nil {
		return err
	}

	wallets := make([]*cardanowallet.Wallet, cnt)

	for i := 0; i < cnt; i++ {
		wallets[i] = cardanowallet.NewWallet(path.Join(currentUser.HomeDir, fmt.Sprintf("cardano_wallet_%d", i+1)), testNetMagic)

		err := wallets[i].Create(false)
		if err != nil {
			return err
		}

		err = wallets[i].Load()
		if err != nil {
			return err
		}
	}

	keyHashes := make([]string, len(wallets))
	for i, w := range wallets {
		keyHashes[i] = w.GetKeyHash()
	}

	multisigAddr := cardanowallet.NewMultiSigAddress(path.Join(currentUser.HomeDir, "cardano_multisig"), keyHashes, testNetMagic, atLeast)
	err = multisigAddr.Create(false)
	if err != nil {
		return err
	}

	err = multisigAddr.Load()
	if err != nil {
		return err
	}

	fmt.Println(multisigAddr.GetAddress())
	fmt.Println(multisigAddr.GetCount())

	metadata, err := json.Marshal(map[string]interface{}{
		"0": map[string]interface{}{
			"who":    "pera",
			"what":   "taba-kera",
			"health": 20,
		},
	})
	if err != nil {
		return err
	}

	hash, err := cardanowallethelper.SendMultiSigTx(multisigAddr, wallets, testNetMagic, socketPath, []cardanowallet.TxOutput{
		{
			Addr:   "addr_test1vqjysa7p4mhu0l25qknwznvj0kghtr29ud7zp732ezwtzec0w8g3u",
			Amount: cardanowallet.MinUtxoValue,
		},
	}, metadata)
	if err != nil {
		return err
	}

	fmt.Println("transaction has been submitted", hash)

	return nil
}

func main() {
	if err := sendMultiSigTx(3, 2); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	if err := sendTx(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
