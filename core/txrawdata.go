package core

import (
	"encoding/hex"
	"encoding/json"

	"github.com/fxamacker/cbor/v2"
)

var (
	witnessJSONType       = "TxWitness BabbageEra"
	witnessJSONDesc       = "Key Witness ShelleyEra"
	txUnwitnessedJSONType = "Unwitnessed Tx BabbageEra"
	txUnwitnessedJSONDesc = "Ledger Cddl Format"
	txWitnessedJSONType   = "Witnessed Tx BabbageEra"
	txWitnessedJSONDesc   = "Ledger Cddl Format"
)

type transactionUnwitnessedRaw []byte

func newTransactionUnwitnessedRawFromJSON(bytes []byte) (transactionUnwitnessedRaw, error) {
	var data map[string]interface{}

	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}

	// a little hack so we have always correct witness key and description for json
	// (cardano-cli can return error otherwise)
	txUnwitnessedJSONType = data["type"].(string)        //nolint:forcetypeassert
	txUnwitnessedJSONDesc = data["description"].(string) //nolint:forcetypeassert

	return hex.DecodeString(data["cborHex"].(string)) //nolint:forcetypeassert
}

func (tx transactionUnwitnessedRaw) ToJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":        txUnwitnessedJSONType,
		"description": txUnwitnessedJSONDesc,
		"cborHex":     hex.EncodeToString(tx),
	})
}

type transactionWitnessedRaw []byte

func newTransactionWitnessedRawFromJSON(bytes []byte) (transactionWitnessedRaw, error) {
	var data map[string]interface{}

	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}

	// a little hack so we have always correct witness key and description for json
	// (cardano-cli can return error otherwise)
	txWitnessedJSONType = data["type"].(string)        //nolint:forcetypeassert
	txWitnessedJSONDesc = data["description"].(string) //nolint:forcetypeassert

	return hex.DecodeString(data["cborHex"].(string)) //nolint:forcetypeassert
}

func (tx transactionWitnessedRaw) ToJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":        txWitnessedJSONType,
		"description": txWitnessedJSONDesc,
		"cborHex":     hex.EncodeToString(tx),
	})
}

type txWitnessRaw []byte // cbor slice of bytes

func (w txWitnessRaw) ToJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":        witnessJSONType,
		"description": witnessJSONDesc,
		"cborHex":     hex.EncodeToString(w),
	})
}

func (w txWitnessRaw) GetSignatureAndVKey() ([]byte, []byte, error) {
	var signatureWitness [2][]byte // Use the appropriate type for your CBOR structure

	if err := cbor.Unmarshal(w, &signatureWitness); err != nil {
		return nil, nil, err
	}

	return signatureWitness[1], signatureWitness[0], nil
}
