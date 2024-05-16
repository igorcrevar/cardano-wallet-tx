package core

import (
	"context"
	"fmt"
)

const defaultTimeToLiveInc = 200

func (b *TxBuilder) SetProtocolParametersAndTTL(
	ctx context.Context, retriever ITxDataRetriever, timeToLiveInc uint64,
) error {
	if timeToLiveInc == 0 {
		timeToLiveInc = defaultTimeToLiveInc
	}

	protocolParams, err := retriever.GetProtocolParameters(ctx)
	if err != nil {
		return err
	}

	tip, err := retriever.GetTip(ctx)
	if err != nil {
		return err
	}

	b.SetProtocolParameters(protocolParams).SetTimeToLive(tip.Slot + timeToLiveInc)

	return nil
}

type TxInputs struct {
	Inputs []TxInput
	Sum    uint64
}

func GetUTXOsForAmount(
	ctx context.Context, retriever IUTxORetriever, addr string, desired uint64, minUtxo uint64,
) (TxInputs, error) {
	utxos, err := retriever.GetUtxos(ctx, addr)
	if err != nil {
		return TxInputs{}, err
	}

	// Loop through utxos to find first input with enough tokens
	// If we don't have this UTXO we need to use more of them
	//nolint:prealloc
	var (
		amountSum   = uint64(0)
		chosenUTXOs []TxInput
	)

	for _, utxo := range utxos {
		amountSum += utxo.Amount
		chosenUTXOs = append(chosenUTXOs, TxInput{
			Hash:  utxo.Hash,
			Index: utxo.Index,
		})

		if amountSum == desired || amountSum >= desired+minUtxo {
			return TxInputs{
				Inputs: chosenUTXOs,
				Sum:    amountSum,
			}, nil
		}
	}

	return TxInputs{}, fmt.Errorf(
		"not enough funds to generate the transaction: %d available vs %d required", amountSum, desired)
}

func GetOutputsSum(outputs []TxOutput) (receiversSum uint64) {
	for _, x := range outputs {
		receiversSum += x.Amount
	}

	return receiversSum
}
