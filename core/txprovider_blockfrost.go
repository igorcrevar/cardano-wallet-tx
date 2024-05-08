package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type TxProviderBlockFrost struct {
	url       string
	projectID string
}

var _ ITxProvider = (*TxProviderBlockFrost)(nil)

func NewTxProviderBlockFrost(url string, projectID string) *TxProviderBlockFrost {
	return &TxProviderBlockFrost{
		projectID: projectID,
		url:       url,
	}
}

func (b *TxProviderBlockFrost) Dispose() {
}

func (b *TxProviderBlockFrost) GetProtocolParameters(ctx context.Context) ([]byte, error) {
	// Create a request with the JSON payload
	req, err := http.NewRequestWithContext(ctx, "GET", b.url+"/epochs/latest/parameters", nil)
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
		return nil, getErrorFromResponse(resp)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return convertProtocolParameters(bytes)
}

func (b *TxProviderBlockFrost) GetUtxos(ctx context.Context, addr string) ([]Utxo, error) {
	// Create a request with the JSON payload
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/addresses/%s/utxos", b.url, addr), nil)
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
		return nil, getErrorFromResponse(resp)
	}

	var responseData []map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, err
	}

	response := make([]Utxo, len(responseData))

	for i, v := range responseData {
		amount := uint64(0)

		//nolint:forcetypeassert
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

		//nolint:forcetypeassert
		response[i] = Utxo{
			Hash:   v["tx_hash"].(string),
			Index:  uint32(v["output_index"].(float64)),
			Amount: amount,
		}
	}

	return response, nil
}

func (b *TxProviderBlockFrost) GetTip(ctx context.Context) (QueryTipData, error) {
	// Create a request with the JSON payload
	req, err := http.NewRequestWithContext(ctx, "GET", b.url+"/blocks/latest", nil)
	if err != nil {
		return QueryTipData{}, err
	}

	// Set the Content-Type header to application/json
	req.Header.Set("Content-Type", "application/cbor")
	req.Header.Set("project_id", b.projectID)

	// Make the HTTP request
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return QueryTipData{}, err
	}

	defer resp.Body.Close()

	// Check the HTTP status code
	if resp.StatusCode != http.StatusOK {
		return QueryTipData{}, getErrorFromResponse(resp)
	}

	var responseData map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return QueryTipData{}, err
	}

	//nolint:forcetypeassert
	return QueryTipData{
		Slot:        uint64(responseData["slot"].(float64)),
		Block:       uint64(responseData["height"].(float64)),
		Epoch:       uint64(responseData["epoch"].(float64)),
		SlotInEpoch: uint64(responseData["epoch_slot"].(float64)),
		Hash:        responseData["hash"].(string),
	}, nil
}

func (b *TxProviderBlockFrost) SubmitTx(ctx context.Context, txSigned []byte) error {
	// Create a request with the JSON payload
	req, err := http.NewRequestWithContext(ctx, "POST", b.url+"/tx/submit", bytes.NewBuffer(txSigned))
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
		return getErrorFromResponse(resp)
	}

	return nil
}

func (b *TxProviderBlockFrost) GetTxByHash(ctx context.Context, hash string) (map[string]interface{}, error) {
	// Create a request with the JSON payload
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/txs/%s", b.url, hash), nil)
	if err != nil {
		return nil, err
	}

	// Set the Content-Type header to application/json
	req.Header.Set("project_id", b.projectID)

	// Make the HTTP request
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // tx not included in block (yet)
	} else if resp.StatusCode != http.StatusOK {
		return nil, getErrorFromResponse(resp)
	}

	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, err
	}

	return responseData, nil
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

	//nolint:forcetypeassert
	resultJSON := map[string]interface{}{
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

	//nolint
	// TODO: "costModels": "PlutusV1" ...

	return json.Marshal(resultJSON)
}

func getErrorFromResponse(resp *http.Response) error {
	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}

	return fmt.Errorf("status code %d: %s", resp.StatusCode, responseData["message"])
}
