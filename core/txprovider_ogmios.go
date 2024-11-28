package core

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const ogmiosJSONRPCVersion = "2.0"

type TxProviderOgmios struct {
	url string
}

var _ ITxProvider = (*TxProviderOgmios)(nil)

func NewTxProviderOgmios(url string) *TxProviderOgmios {
	return &TxProviderOgmios{
		url: url,
	}
}

// Dispose implements ITxProvider.
func (o *TxProviderOgmios) Dispose() {}

// GetProtocolParameters implements ITxProvider.
func (o *TxProviderOgmios) GetProtocolParameters(ctx context.Context) ([]byte, error) {
	params, err := executeHTTPOgmios[ogmiosQueryProtocolParamsResponse](
		ctx, o.url, ogmiosQueryStateRequest{
			Jsonrpc: ogmiosJSONRPCVersion,
			Method:  "queryLedgerState/protocolParameters",
		}, false,
	)
	if err != nil {
		return nil, err
	}

	asFloat := func(s string) float64 {
		v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)

		return v
	}

	resultJSON := map[string]interface{}{
		"extraPraosEntropy": nil,
		"decentralization":  nil,
		"protocolVersion": map[string]interface{}{
			"major": params.Result.Version.Major,
			"minor": params.Result.Version.Minor,
		},
		"maxBlockHeaderSize":   params.Result.MaxBlockHeaderSize.Bytes,
		"maxBlockBodySize":     params.Result.MaxBlockBodySize.Bytes,
		"maxTxSize":            params.Result.MaxTransactionSize.Bytes,
		"txFeeFixed":           params.Result.MinFeeConstant.Ada.Lovelace,
		"txFeePerByte":         params.Result.MinFeeCoefficient,
		"stakeAddressDeposit":  params.Result.StakeCredentialDeposit.Ada.Lovelace,
		"stakePoolDeposit":     params.Result.StakePoolDeposit.Ada.Lovelace,
		"minPoolCost":          params.Result.MinStakePoolCost.Ada.Lovelace,
		"poolRetireMaxEpoch":   params.Result.StakePoolRetirementEpochBound,
		"stakePoolTargetNum":   params.Result.DesiredNumberOfStakePools,
		"poolPledgeInfluence":  asFloat(params.Result.StakePoolPledgeInfluence),
		"monetaryExpansion":    asFloat(params.Result.MonetaryExpansion),
		"treasuryCut":          asFloat(params.Result.TreasuryExpansion),
		"collateralPercentage": params.Result.CollateralPercentage,
		"executionUnitPrices": map[string]interface{}{
			"priceMemory": asFloat(params.Result.ScriptExecutionPrices.Memory),
			"priceSteps":  asFloat(params.Result.ScriptExecutionPrices.CPU),
		},
		"utxoCostPerByte": params.Result.MinUtxoDepositCoefficient, // coins_per_utxo_size ?
		"minUTxOValue":    nil,                                     // min_utxo? this was nil with cardano-cli
		"maxTxExecutionUnits": map[string]interface{}{
			"memory": params.Result.MaxExecutionUnitsPerTransaction.Memory,
			"steps":  params.Result.MaxExecutionUnitsPerTransaction.CPU,
		},
		"maxBlockExecutionUnits": map[string]interface{}{
			"memory": params.Result.MaxExecutionUnitsPerBlock.Memory,
			"steps":  params.Result.MaxExecutionUnitsPerBlock.CPU,
		},
		"maxCollateralInputs": params.Result.MaxCollateralInputs,
		"maxValueSize":        params.Result.MaxValueSize.Bytes,
	}

	//nolint
	// TODO: "costModels": "PlutusV1" ...

	return json.Marshal(resultJSON)
}

// GetSlot implements ITxProvider.
func (o *TxProviderOgmios) GetTip(ctx context.Context) (QueryTipData, error) {
	heightResponse, err := executeHTTPOgmios[ogmiosQueryNetworkBlockHeightResponse](
		ctx, o.url, ogmiosQueryStateRequest{
			Jsonrpc: ogmiosJSONRPCVersion,
			Method:  "queryNetwork/blockHeight",
		}, false,
	)
	if err != nil {
		return QueryTipData{}, err
	}

	tipResponse, err := executeHTTPOgmios[ogmiosQueryTipResponse](
		ctx, o.url, ogmiosQueryStateRequest{
			Jsonrpc: ogmiosJSONRPCVersion,
			Method:  "queryLedgerState/tip",
		}, false,
	)
	if err != nil {
		return QueryTipData{}, err
	}

	return QueryTipData{
		Block: heightResponse.Result,
		Hash:  tipResponse.Result.ID,
		Slot:  tipResponse.Result.Slot,
	}, nil
}

// GetUtxos implements ITxProvider.
func (o *TxProviderOgmios) GetUtxos(ctx context.Context, addr string) ([]Utxo, error) {
	responseData, err := executeHTTPOgmios[ogmiosQueryUtxoResponse](
		ctx, o.url, ogmiosQueryUtxoRequest{
			Jsonrpc: ogmiosJSONRPCVersion,
			Method:  "queryLedgerState/utxo",
			Params: ogmiosQueryUtxoRequestParams{
				Addresses: []string{addr},
			},
		}, true,
	)
	if err != nil {
		return nil, err
	}

	var retVal = make([]Utxo, len(responseData.Result))
	for i, utxo := range responseData.Result {
		var (
			adaValue uint64
			tokens   []TokenAmount
		)

		if len(utxo.Value) > 1 {
			tokens = make([]TokenAmount, 0, len(utxo.Value)-1)
		}

		for policyID, nameValueMap := range utxo.Value {
			if policyID == AdaTokenPolicyID {
				adaValue = nameValueMap[AdaTokenName]
			} else {
				for name, value := range nameValueMap {
					realName, err := hex.DecodeString(name)
					if err == nil {
						name = string(realName)
					}

					tokens = append(tokens, TokenAmount{
						PolicyID: policyID,
						Name:     name,
						Amount:   value,
					})
				}
			}
		}

		retVal[i] = Utxo{
			Hash:   utxo.Transaction.ID,
			Index:  utxo.Index,
			Amount: adaValue,
			Tokens: tokens,
		}
	}

	return retVal, nil
}

// Expects TxCborString
func (o *TxProviderOgmios) SubmitTx(ctx context.Context, txSigned []byte) error {
	response, err := executeHTTPOgmios[ogmiosSubmitTransactionResponse](
		ctx, o.url, ogmiosSubmitTransaction{
			Jsonrpc: ogmiosJSONRPCVersion,
			Method:  "submitTransaction",
			Params: ogmiosSubmitTransactionParams{
				Transaction: ogmiosSubmitTransactionParamsTransaction{
					CBOR: hex.EncodeToString(txSigned),
				},
			},
			ID: nil,
		}, false,
	)
	if err != nil {
		return err
	}

	if response.Error.Message != "" {
		return fmt.Errorf("ogmios submit tx error: %s", response.Error.Message)
	}

	return nil
}

func (o *TxProviderOgmios) GetTxByHash(ctx context.Context, hash string) (map[string]interface{}, error) {
	panic("not implemented") //nolint:gocritic
}

func executeHTTPOgmios[T any](
	ctx context.Context, url string, request any, notFoundIsNotError bool,
) (T, error) {
	var result T // Zero value for type T

	queryBytes, err := json.Marshal(request)
	if err != nil {
		return result, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(queryBytes))
	if err != nil {
		return result, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if notFoundIsNotError && resp.StatusCode == http.StatusNotFound {
		return result, nil // tx not included in block (yet)
	}

	if resp.StatusCode != http.StatusOK {
		return result, getErrorFromResponseOgmios(resp)
	}

	err = json.NewDecoder(resp.Body).Decode(&result)

	return result, err
}

func getErrorFromResponseOgmios(resp *http.Response) error {
	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}

	msg := responseData["error"].(map[string]interface{})["message"].(string) //nolint:forcetypeassert

	return fmt.Errorf("status code %d: %s", resp.StatusCode, msg)
}
