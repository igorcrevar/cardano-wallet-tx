package core

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/fxamacker/cbor/v2"
)

type TxProviderBlockFrost struct {
	url       string
	projectID string
}

func NewTxProviderBlockFrost(url string, projectID string) (*TxProviderBlockFrost, error) {
	return &TxProviderBlockFrost{
		projectID: projectID,
		url:       url,
	}, nil
}

func (b *TxProviderBlockFrost) Dispose() {
}

func (b *TxProviderBlockFrost) GetProtocolParameters() ([]byte, error) {
	// Create a request with the JSON payload
	req, err := http.NewRequest("GET", b.url+"/epochs/latest/parameters", nil)
	if err != nil {
		return nil, err
	}

	// Set the Content-Type header to application/json
	req.Header.Set("Content-Type", "application/cbor")
	req.Header.Set("project_id", b.projectID)

	// Make the HTTP request
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// Check the HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code is %d", resp.StatusCode)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return convertProtocolParameters(bytes)
}

func (b *TxProviderBlockFrost) GetUtxos(addr string) ([]Utxo, error) {
	// Create a request with the JSON payload
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/addresses/%s/utxos", b.url, addr), nil)
	if err != nil {
		return nil, err
	}

	// Set the Content-Type header to application/json
	req.Header.Set("Content-Type", "application/cbor")
	req.Header.Set("project_id", b.projectID)

	// Make the HTTP request
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// Check the HTTP status code
	if resp.StatusCode == http.StatusNotFound {
		return []Utxo{}, nil // this address does not have any UTxOs
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code is %d", resp.StatusCode)
	}

	var responseData []map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, err
	}

	response := make([]Utxo, len(responseData))
	for i, v := range responseData {
		amount := uint64(0)

		for _, item := range v["amount"].([]interface{}) {
			itemMap := item.(map[string]interface{})
			if itemMap["unit"].(string) == "lovelace" {
				amount, err = strconv.ParseUint(itemMap["quantity"].(string), 10, 64)
				if err != nil {
					return nil, err
				}

				break
			}
		}

		response[i] = Utxo{
			Hash:   v["tx_hash"].(string),
			Index:  uint32(v["output_index"].(float64)),
			Amount: amount,
		}
	}

	return response, nil
}

func (b *TxProviderBlockFrost) GetSlot() (uint64, error) {
	// Create a request with the JSON payload
	req, err := http.NewRequest("GET", b.url+"/blocks/latest", nil)
	if err != nil {
		return 0, err
	}

	// Set the Content-Type header to application/json
	req.Header.Set("Content-Type", "application/cbor")
	req.Header.Set("project_id", b.projectID)

	// Make the HTTP request
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	// Check the HTTP status code
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("status code is %d", resp.StatusCode)
	}

	var responseData map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return 0, err
	}

	return uint64(responseData["slot"].(float64)), nil
}

func (b *TxProviderBlockFrost) SubmitTx(tx []byte) error {
	if bytes.Contains(tx, []byte("cborHex")) {
		var responseData map[string]interface{}
		if err := json.Unmarshal(tx, &responseData); err != nil {
			return err
		}

		bytes, err := hex.DecodeString(responseData["cborHex"].(string))
		if err != nil {
			return err
		}

		tx = bytes
	} else {
		serializedTxBor, err := cbor.Marshal(tx)
		if err != nil {
			return err
		}

		tx = serializedTxBor
	}

	// Create a request with the JSON payload
	req, err := http.NewRequest("POST", b.url+"/tx/submit", bytes.NewBuffer(tx))
	if err != nil {
		return err
	}

	// Set the Content-Type header to application/json
	req.Header.Set("Content-Type", "application/cbor")
	req.Header.Set("project_id", b.projectID)

	// Make the HTTP request
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// Check the HTTP status code
	if resp.StatusCode != http.StatusOK {
		var responseData map[string]interface{}
		if err = json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
			return fmt.Errorf("status code %d", resp.StatusCode)
		}

		return fmt.Errorf("status code %d: %s", resp.StatusCode, responseData["message"])
	}

	return nil
}

func convertProtocolParameters(bytes []byte) ([]byte, error) {
	var jsonMap map[string]interface{}

	if err := json.Unmarshal(bytes, &jsonMap); err != nil {
		return nil, err
	}

	strToUInt64 := func(s string) uint64 {
		v, _ := strconv.ParseUint(s, 10, 64)
		return v
	}

	resultJson := map[string]interface{}{
		"extraPraosEntropy": nil,
		"decentralization":  nil,
		"protocolVersion": map[string]interface{}{
			"major": jsonMap["protocol_major_ver"],
			"minor": jsonMap["protocol_minor_ver"],
		},
		"maxBlockHeaderSize":   jsonMap["max_block_header_size"],
		"maxBlockBodySize":     jsonMap["max_block_size"],
		"maxTxSize":            jsonMap["max_tx_size"],
		"txFeeFixed":           jsonMap["min_fee_b"],
		"txFeePerByte":         jsonMap["min_fee_a"],
		"stakeAddressDeposit":  strToUInt64(jsonMap["key_deposit"].(string)),
		"stakePoolDeposit":     strToUInt64(jsonMap["pool_deposit"].(string)),
		"minPoolCost":          strToUInt64(jsonMap["min_pool_cost"].(string)),
		"poolRetireMaxEpoch":   jsonMap["e_max"],
		"stakePoolTargetNum":   jsonMap["n_opt"],
		"poolPledgeInfluence":  jsonMap["a0"],
		"monetaryExpansion":    jsonMap["rho"],
		"treasuryCut":          jsonMap["tau"],
		"collateralPercentage": jsonMap["collateral_percent"],
		"executionUnitPrices": map[string]interface{}{
			"priceMemory": jsonMap["price_mem"],
			"priceSteps":  jsonMap["price_step"],
		},
		"utxoCostPerByte": strToUInt64(jsonMap["coins_per_utxo_word"].(string)), // coins_per_utxo_size ?
		"minUTxOValue":    nil,                                                  // min_utxo? this was nil with cardano-cli
		"maxTxExecutionUnits": map[string]interface{}{
			"memory": strToUInt64(jsonMap["max_tx_ex_mem"].(string)),
			"steps":  strToUInt64(jsonMap["max_tx_ex_steps"].(string)),
		},
		"maxBlockExecutionUnits": map[string]interface{}{
			"memory": strToUInt64(jsonMap["max_block_ex_mem"].(string)),
			"steps":  strToUInt64(jsonMap["max_block_ex_steps"].(string)),
		},
		"maxCollateralInputs": jsonMap["max_collateral_inputs"],
		"maxValueSize":        strToUInt64(jsonMap["max_val_size"].(string)),
	}

	// TODO: "costModels": "PlutusV1" ...

	return json.Marshal(resultJson)
}
