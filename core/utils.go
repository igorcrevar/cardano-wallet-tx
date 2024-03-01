package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type AddressType string

const (
	AddressTypeStake      AddressType = "stake1"
	AddressTypeBase       AddressType = "addr1"
	AddressTypeTestStake  AddressType = "stake_test1"
	AddressTypeTestBase   AddressType = "addr_test1"
	AddressTypeAny        AddressType = ""
	AddressTypeAnyTest    AddressType = "test"
	AddressTypeAnyMainnet AddressType = "mainnet"
)

type AddressInfo struct {
	Address  string      `json:"address"`
	Base16   string      `json:"base16"`
	Encoding string      `json:"encoding"`
	Era      string      `json:"era"`
	Type     AddressType `json:"type"`
	IsValid  bool        `json:"-"`
	ErrorMsg string      `json:"-"`
}

// isValidCardanoAddress checks if the given string is a valid Cardano address.
func GetAddressInfo(address string, addressType AddressType) AddressInfo {
	res, err := runCommand(resolveCardanoCliBinary(), []string{
		"address", "info", "--address", address,
	})
	if err != nil {
		return AddressInfo{
			IsValid:  false,
			ErrorMsg: err.Error(),
		}
	}

	var ai AddressInfo

	if err := json.Unmarshal([]byte(strings.Trim(res, "\n")), &ai); err != nil {
		return AddressInfo{
			IsValid:  false,
			ErrorMsg: err.Error(),
		}
	}

	// Check if the address starts with correct prefix for mainnet and testnet respectively
	switch addressType {
	case AddressTypeAny:
		ai.IsValid = true
	case AddressTypeAnyMainnet:
		ai.IsValid = strings.HasPrefix(ai.Address, string(AddressTypeBase)) || strings.HasPrefix(ai.Address, string(AddressTypeStake))
	case AddressTypeAnyTest:
		ai.IsValid = strings.HasPrefix(ai.Address, string(AddressTypeTestBase)) || strings.HasPrefix(ai.Address, string(AddressTypeTestStake))
	default:
		ai.IsValid = strings.HasPrefix(ai.Address, string(addressType))
	}

	return ai
}

func WaitForTransaction(ctx context.Context, txRetriever ITxRetriever,
	hash string, numRetries int, waitTime time.Duration) (map[string]interface{}, error) {
	for count := 0; count < numRetries; count++ {
		result, err := txRetriever.GetTxByHash(hash)
		if err != nil {
			return nil, err
		} else if result != nil {
			return result, nil
		}

		select {
		case <-time.After(waitTime):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("timeout while waiting for transaction %s to be processed", hash)
}
