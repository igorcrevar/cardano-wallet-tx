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

var (
	ErrInvalidSignature          = errors.New("invalid signature")
	ErrInvalidAddressInfo        = errors.New("invalid address info")
	ErrWaitForTransactionTimeout = errors.New("timeout while waiting for transaction")
)

type AddressInfo struct {
	Address  string `json:"address"`
	Base16   string `json:"base16"`
	Encoding string `json:"encoding"`
	Era      string `json:"era"`
	Type     string `json:"type"`
}

// GetAddressInfo returns address info if string representation for address is valid or error
func GetAddressInfo(address string) (AddressInfo, error) {
	var ai AddressInfo

	res, err := runCommand(resolveCardanoCliBinary(), []string{
		"address", "info", "--address", address,
	})
	if err != nil {
		return ai, errors.Join(ErrInvalidAddressInfo, err)
	}

	if err := json.Unmarshal([]byte(strings.Trim(res, "\n")), &ai); err != nil {
		return ai, errors.Join(ErrInvalidAddressInfo, err)
	}

	return ai, nil
}

// WaitForTransaction waits for transaction to be included in block
func WaitForTransaction(ctx context.Context, txRetriever ITxRetriever,
	hash string, numRetries int, waitTime time.Duration) (map[string]interface{}, error) {
	for count := 0; count < numRetries; count++ {
		result, err := txRetriever.GetTxByHash(ctx, hash)
		if err != nil {
			return nil, err
		} else if result != nil {
			return result, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(waitTime):
		}
	}

	return nil, ErrWaitForTransactionTimeout
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

// SignMessage signs message
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
		err = ErrInvalidSignature
	}

	return
}

// GetVerificationKeyFromSigningKey retrieves verification/public key from signing/private key
func GetVerificationKeyFromSigningKey(signingKey []byte) []byte {
	return ed25519.NewKeyFromSeed(signingKey).Public().(ed25519.PublicKey)
}
