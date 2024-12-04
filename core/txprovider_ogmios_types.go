package core

type ogmiosQueryStateRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	ID      interface{} `json:"id"`
}

type ogmiosQueryUtxoRequestParams struct {
	Addresses []string `json:"addresses"`
}

type ogmiosQueryUtxoRequest struct {
	Jsonrpc string                       `json:"jsonrpc"`
	Method  string                       `json:"method"`
	Params  ogmiosQueryUtxoRequestParams `json:"params"`
	ID      interface{}                  `json:"id"`
}

type ogmiosQueryUtxoResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Result  []struct {
		Transaction struct {
			ID string `json:"id"`
		} `json:"transaction"`
		Index   uint32                       `json:"index"`
		Address string                       `json:"address"`
		Value   map[string]map[string]uint64 `json:"value"`
	} `json:"result"`
	ID interface{} `json:"id"`
}

type ogmiosQueryProtocolParamsResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Result  struct {
		MinFeeCoefficient uint64 `json:"minFeeCoefficient"`
		MinFeeConstant    struct {
			Ada struct {
				Lovelace uint64 `json:"lovelace"`
			} `json:"ada"`
		} `json:"minFeeConstant"`
		MaxBlockBodySize struct {
			Bytes uint64 `json:"bytes"`
		} `json:"maxBlockBodySize"`
		MaxBlockHeaderSize struct {
			Bytes uint64 `json:"bytes"`
		} `json:"maxBlockHeaderSize"`
		MaxTransactionSize struct {
			Bytes uint64 `json:"bytes"`
		} `json:"maxTransactionSize"`
		StakeCredentialDeposit struct {
			Ada struct {
				Lovelace uint64 `json:"lovelace"`
			} `json:"ada"`
		} `json:"stakeCredentialDeposit"`
		StakePoolDeposit struct {
			Ada struct {
				Lovelace uint64 `json:"lovelace"`
			} `json:"ada"`
		} `json:"stakePoolDeposit"`
		StakePoolRetirementEpochBound uint64 `json:"stakePoolRetirementEpochBound"`
		DesiredNumberOfStakePools     uint64 `json:"desiredNumberOfStakePools"`
		StakePoolPledgeInfluence      string `json:"stakePoolPledgeInfluence"`
		MonetaryExpansion             string `json:"monetaryExpansion"`
		TreasuryExpansion             string `json:"treasuryExpansion"`
		MinStakePoolCost              struct {
			Ada struct {
				Lovelace uint64 `json:"lovelace"`
			} `json:"ada"`
		} `json:"minStakePoolCost"`
		MinUtxoDepositConstant struct {
			Ada struct {
				Lovelace uint64 `json:"lovelace"`
			} `json:"ada"`
		} `json:"minUtxoDepositConstant"`
		MinUtxoDepositCoefficient uint64             `json:"minUtxoDepositCoefficient"`
		PlutusCostModels          map[string][]int64 `json:"plutusCostModels"`
		ScriptExecutionPrices     struct {
			Memory string `json:"memory"`
			CPU    string `json:"cpu"`
		} `json:"scriptExecutionPrices"`
		MaxExecutionUnitsPerTransaction struct {
			Memory uint64 `json:"memory"`
			CPU    uint64 `json:"cpu"`
		} `json:"maxExecutionUnitsPerTransaction"`
		MaxExecutionUnitsPerBlock struct {
			Memory uint64 `json:"memory"`
			CPU    uint64 `json:"cpu"`
		} `json:"maxExecutionUnitsPerBlock"`
		MaxValueSize struct {
			Bytes uint64 `json:"bytes"`
		} `json:"maxValueSize"`
		CollateralPercentage uint64 `json:"collateralPercentage"`
		MaxCollateralInputs  uint64 `json:"maxCollateralInputs"`
		Version              struct {
			Major uint64 `json:"major"`
			Minor uint64 `json:"minor"`
		} `json:"version"`
	} `json:"result"`
	ID interface{} `json:"id"`
}

type ogmiosQueryTipResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Result  struct {
		Slot uint64 `json:"slot"`
		ID   string `json:"id"`
	} `json:"result"`
	ID interface{} `json:"id"`
}

type ogmiosSubmitTransactionParamsTransaction struct {
	CBOR string `json:"cbor"`
}

type ogmiosSubmitTransactionParams struct {
	Transaction ogmiosSubmitTransactionParamsTransaction `json:"transaction"`
}

type ogmiosSubmitTransaction struct {
	Jsonrpc string                        `json:"jsonrpc"`
	Method  string                        `json:"method"`
	Params  ogmiosSubmitTransactionParams `json:"params"`
	ID      interface{}                   `json:"id"`
}

// Define the types for the submit transaction response
type ogmiosSubmitTransactionResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Result  struct {
		Transaction struct {
			ID string `json:"id"`
		} `json:"transaction"`
	} `json:"result"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			MissingScripts []string `json:"missingScripts"`
		} `json:"data"`
	} `json:"error"`
	ID interface{} `json:"id"`
}

type ogmiosQueryNetworkBlockHeightResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Result  uint64      `json:"result"`
	ID      interface{} `json:"id"`
}
