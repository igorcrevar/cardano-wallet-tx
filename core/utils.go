package core

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
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

var ErrInvalidWitness = errors.New("invalid witness")

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

// WaitForTransaction waits for transaction to be included in block
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

// VerifyWitness verifies if txHash is signed by witness
func VerifyWitness(txHash string, witness []byte) error {
	txHashBytes, err := hex.DecodeString(txHash)
	if err != nil {
		return err
	}

	signature, vKey, err := TxWitnessRaw(witness).GetSignatureAndVKey()
	if err != nil {
		return err
	}

	return VerifyMessage(txHashBytes, vKey, signature)
}

// Sign signs message. This method can panic, use with caution!
func SignMessage(signingKey, verificationKey, message []byte) (result []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error: %v", r)
		}
	}()

	privateKey := make([]byte, len(signingKey)+len(verificationKey))

	copy(privateKey, signingKey)
	copy(privateKey[32:], verificationKey)

	result = ed25519.Sign(privateKey, message)

	return
}

// VerifyMessage verifies message with verificationKey and signature
func VerifyMessage(message, verificationKey, signature []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error: %v", r)
		}
	}()

	if !ed25519.Verify(verificationKey, message, signature) {
		err = ErrInvalidWitness
	}

	return
}

func GetVerificationKeyFromSigningKey(signingKey []byte) []byte {
	return ed25519.NewKeyFromSeed(signingKey).Public().(ed25519.PublicKey)
}
