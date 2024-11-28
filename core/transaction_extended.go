package core

import (
	"context"
	"fmt"

	"github.com/igorcrevar/cardano-wallet-tx/common"
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
	Sum    map[string]uint64
}

func GetUTXOsForAmount(
	ctx context.Context,
	retriever IUTxORetriever,
	addr string,
	tokenNames []string,
	exactSum map[string]uint64,
	atLeastSum map[string]uint64,
) (TxInputs, error) {
	utxos, err := common.ExecuteWithRetry(ctx, func(ctx context.Context) ([]Utxo, error) {
		return retriever.GetUtxos(ctx, addr)
	})
	if err != nil {
		return TxInputs{}, err
	}

	// Loop through utxos to find first input with enough tokens
	// If we don't have this UTXO we need to use more of them
	//nolint:prealloc
	var (
		currentSum       = map[string]uint64{}
		chosenUTXOs      []TxInput
		notGoodTokenName string
	)

	for _, utxo := range utxos {
		currentSum[AdaTokenName] += utxo.Amount

		for _, token := range utxo.Tokens {
			currentSum[token.TokenName()] += token.Amount
		}

		chosenUTXOs = append(chosenUTXOs, TxInput{
			Hash:  utxo.Hash,
			Index: utxo.Index,
		})

		isOk := true

		for _, tokenName := range tokenNames {
			if currentSum[tokenName] != exactSum[tokenName] && currentSum[tokenName] < atLeastSum[tokenName] {
				isOk = false
				notGoodTokenName = tokenName

				break
			}
		}

		if isOk {
			return TxInputs{
				Inputs: chosenUTXOs,
				Sum:    currentSum,
			}, nil
		}
	}

	return TxInputs{}, fmt.Errorf("not enough funds for the transaction: (available, exact, at least) = (%d, %d, %d)",
		currentSum[notGoodTokenName], exactSum[notGoodTokenName], atLeastSum[notGoodTokenName])
}
