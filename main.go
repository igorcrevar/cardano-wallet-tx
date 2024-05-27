package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/user"
	"path"
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

func createTx(
	txProvider cardanowallet.ITxProvider, wallet cardanowallet.IWallet, testNetMagic uint, potentialFee uint64,
	receiverAddr string,
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

	builder, err := cardanowallet.NewTxBuilder()
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
		context.Background(), txProvider, address, outputsSum+potentialFee, cardanowallet.MinUTxODefaultValue)
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
	txProvider cardanowallet.ITxProvider, signers []cardanowallet.IWallet,
	feeSigners []cardanowallet.IWallet, testNetMagic uint, potentialFee uint64,
	receiverAddr string,
) ([]byte, string, error) {
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

	builder, err := cardanowallet.NewTxBuilder()
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
		context.Background(), txProvider, multiSigAddr,
		outputsSum, cardanowallet.MinUTxODefaultValue)
	if err != nil {
		return nil, "", err
	}

	multiSigFeeInputs, err := cardanowallet.GetUTXOsForAmount(
		context.Background(), txProvider, multiSigFeeAddr,
		potentialFee, cardanowallet.MinUTxODefaultValue)
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
		return cardanowallet.NewTxProviderBlockFrost(blockfrostUrl, blockfrostProjectApiKey), nil
	case "ogmios":
		return cardanowallet.NewTxProviderOgmios(ogmiosUrl), nil
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

func submitTx(
	ctx context.Context, txProvider cardanowallet.ITxProvider, txRaw []byte, txHash string, addr string,
) error {
	utxo, err := txProvider.GetUtxos(ctx, addr)
	if err != nil {
		return err
	}

	prev := cardanowallet.GetUtxosSum(utxo)

	if err := txProvider.SubmitTx(context.Background(), txRaw); err != nil {
		return err
	}

	fmt.Println("transaction has been submitted. hash =", txHash)

	err = cardanowallet.WaitForAmount(ctx, txProvider, addr, func(val *big.Int) bool {
		return prev.Cmp(val) < 0
	}, 60, time.Second*5)
	if err != nil {
		return err
	}

	fmt.Println("transaction has been included in block. hash =", txHash)

	return nil
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

	txProvider, err := createProvider(providerName)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	defer txProvider.Dispose()

	_, _ = txProvider.GetTip(context.Background())

	receiverAddr, _, err := cardanowallet.GetWalletAddressCli(wallets[1], testNetMagic)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	multiSigTx, multiSigTxHash, err := createMultiSigTx(
		txProvider, wallets[:3], wallets[3:], testNetMagic, potentialFee, receiverAddr)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	if err := submitTx(context.Background(), txProvider, multiSigTx, multiSigTxHash, receiverAddr); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	sigTx, txHash, err := createTx(
		txProvider, wallets[0], testNetMagic, potentialFee, receiverAddr)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	if err := submitTx(context.Background(), txProvider, sigTx, txHash, receiverAddr); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
