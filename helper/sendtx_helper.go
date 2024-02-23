package helper

import "github.com/igorcrevar/cardano-wallet-tx/core"

func SendTx(
	wallet *core.Wallet,
	testNetMagic uint,
	socketPath string,
	outputs []core.TxOutput,
	metadata []byte) (string, error) {
	txDataRetriever, err := core.NewTxDataRetriever(testNetMagic, socketPath)
	if err != nil {
		return "", err
	}

	defer txDataRetriever.Dispose()

	protocolParams, err := txDataRetriever.GetProtocolParameters()
	if err != nil {
		return "", err
	}

	slot, err := txDataRetriever.GetSlot()
	if err != nil {
		return "", err
	}

	utxos, err := txDataRetriever.GetUtxos(wallet.GetAddress())
	if err != nil {
		return "", err
	}

	builder, err := core.NewTxBuilder()
	if err != nil {
		return "", err
	}

	defer builder.Dispose()

	txRaw, hash, err := builder.BuildWithDto(core.TransactionDTO{
		FromAddress:       wallet.GetAddress(),
		TestNetMagic:      testNetMagic,
		Outputs:           outputs,
		SlotNumber:        slot,
		Utxos:             utxos,
		ProtocolParamters: protocolParams,
		MetaData:          metadata,
		PotentialFee:      200_000,
	})
	if err != nil {
		return "", err
	}

	txSigned, err := builder.Sign(txRaw, wallet.GetSigningKeyPath())
	if err != nil {
		return "", err
	}

	return hash, txDataRetriever.SubmitTx(txSigned)
}

func SendMultiSigTx(
	multisigAddr *core.MultisigAddress,
	wallets []*core.Wallet,
	testNetMagic uint,
	socketPath string,
	outputs []core.TxOutput,
	metadata []byte) (string, error) {
	txDataRetriever, err := core.NewTxDataRetriever(testNetMagic, socketPath)
	if err != nil {
		return "", err
	}

	defer txDataRetriever.Dispose()

	protocolParams, err := txDataRetriever.GetProtocolParameters()
	if err != nil {
		return "", err
	}

	slot, err := txDataRetriever.GetSlot()
	if err != nil {
		return "", err
	}

	utxos, err := txDataRetriever.GetUtxos(multisigAddr.GetAddress())
	if err != nil {
		return "", err
	}

	builder, err := core.NewTxBuilder()
	if err != nil {
		return "", err
	}

	defer builder.Dispose()

	policy, err := multisigAddr.GetPolicyScript()
	if err != nil {
		return "", err
	}

	txRaw, hash, err := builder.BuildWithDto(core.TransactionDTO{
		FromAddress:       multisigAddr.GetAddress(),
		TestNetMagic:      testNetMagic,
		Outputs:           outputs,
		SlotNumber:        slot,
		Utxos:             utxos,
		ProtocolParamters: protocolParams,
		MetaData:          metadata,
		Policy:            policy,
		WitnessCount:      multisigAddr.GetCount(),
		PotentialFee:      200_000,
	})
	if err != nil {
		return "", err
	}

	witnesses := make([][]byte, len(wallets))

	for i, x := range wallets {
		witness, err := builder.AddWitness(txRaw, x.GetSigningKeyPath())
		if err != nil {
			return "", err
		}

		witnesses[i] = witness
	}

	txSigned, err := builder.AssembleWitnesses(txRaw, witnesses)
	if err != nil {
		return "", err
	}

	return hash, txDataRetriever.SubmitTx(txSigned)
}
