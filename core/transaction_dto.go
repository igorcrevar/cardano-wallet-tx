package core

import (
	"encoding/json"
	"fmt"
)

type GetUTXOsForAmountHandler func(utxos []Utxo, receiversSum uint64, potentialFee uint64, minUtxo uint64) ([]TxInput, uint64, error)

type TransactionDTO struct {
	FromAddress         string
	TestNetMagic        uint
	Outputs             []TxOutput
	SlotNumber          uint64
	Utxos               []Utxo
	ProtocolParameters  []byte
	MetaData            []byte
	Policy              []byte
	WitnessCount        int
	PotentialFee        uint64
	GetUTXOsForAmountFN GetUTXOsForAmountHandler
	TimeToLiveInc       uint64 // how much current slot number should be increment
}

func NewTransactionDTO(retriever ITxDataRetriever, addr string) (TransactionDTO, error) {
	protocolParams, err := retriever.GetProtocolParameters()
	if err != nil {
		return TransactionDTO{}, err
	}

	slot, err := retriever.GetSlot()
	if err != nil {
		return TransactionDTO{}, err
	}

	utxos, err := retriever.GetUtxos(addr)
	if err != nil {
		return TransactionDTO{}, err
	}

	return TransactionDTO{
		Utxos:              utxos,
		SlotNumber:         slot,
		ProtocolParameters: protocolParams,
		FromAddress:        addr,
		TimeToLiveInc:      200,
	}, nil
}

func (dto TransactionDTO) GetMinUtxoFromProtocolParameters() (uint64, error) {
	var mp map[string]interface{}

	if err := json.Unmarshal(dto.ProtocolParameters, &mp); err != nil {
		return 0, err
	}

	if v := mp["minUTxOValue"]; v != nil {
		return uint64(v.(float64)), nil
	}

	return MinUTxODefaultValue, nil
}

func (b TxBuilder) BuildWithDTO(dto TransactionDTO) ([]byte, string, error) {
	minUtxo, err := dto.GetMinUtxoFromProtocolParameters()
	if err != nil {
		return nil, "", err
	}

	receiversSum := uint64(0)
	for _, x := range dto.Outputs {
		receiversSum += x.Amount
	}

	fn := dto.GetUTXOsForAmountFN
	if fn == nil {
		fn = GetUTXOsForAmount
	}

	inputs, utxosSum, err := fn(dto.Utxos, receiversSum, dto.PotentialFee, minUtxo)
	if err != nil {
		return nil, "", err
	}

	b.SetTestNetMagic(dto.TestNetMagic).SetPolicy(dto.Policy, dto.WitnessCount).SetMetaData(dto.MetaData)
	b.SetProtocolParameters(dto.ProtocolParameters).SetTimeToLive(dto.SlotNumber + dto.TimeToLiveInc)
	b.AddInputs(inputs...).AddOutputs(dto.Outputs...).AddOutputs(TxOutput{
		Addr: dto.FromAddress,
	})

	fee, err := b.CalculateFee()
	if err != nil {
		return nil, "", err
	}

	b.SetFee(fee).UpdateLastOutputAmount(utxosSum - fee - receiversSum)

	txRaw, err := b.Build()
	if err != nil {
		return nil, "", err
	}

	hash, err := b.GetTxHash(txRaw)
	if err != nil {
		return nil, "", err
	}

	return txRaw, hash, nil
}

func GetUTXOsForAmount(utxos []Utxo, receiversSum uint64, potentialFee uint64, minUtxo uint64) ([]TxInput, uint64, error) {
	// Loop through utxos to find first input with enough tokens
	// If we don't have this UTXO we need to use more of them
	var (
		amountSum   = uint64(0)
		chosenUTXOs []TxInput
		desired     = receiversSum + minUtxo + potentialFee
	)

	for _, utxo := range utxos {
		if utxo.Amount >= desired {
			return []TxInput{{
				Hash:  utxo.Hash,
				Index: utxo.Index,
			}}, utxo.Amount, nil
		}

		amountSum += utxo.Amount
		chosenUTXOs = append(chosenUTXOs, TxInput{
			Hash:  utxo.Hash,
			Index: utxo.Index,
		})

		if amountSum >= desired {
			return chosenUTXOs, amountSum, nil
		}
	}

	return nil, 0, fmt.Errorf("not enough funds to generate the transaction: %d available vs %d required", amountSum, desired)
}
