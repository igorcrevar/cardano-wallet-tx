package core

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"

	"golang.org/x/crypto/blake2b"
)

const FilePermission = 0750

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

// WaitForAmount waits for address to have amount specified by cmpHandler
func WaitForAmount(ctx context.Context, txRetriever IUTxORetriever,
	addr string, cmpHandler func(*big.Int) bool, numRetries int, waitTime time.Duration) error {
	for count := 0; count < numRetries; count++ {
		uxtos, err := txRetriever.GetUtxos(ctx, addr)
		if err != nil {
			return err
		} else {
			sum := big.NewInt(0)
			for _, utxo := range uxtos {
				sum.Add(sum, new(big.Int).SetUint64(utxo.Amount))
			}

			if cmpHandler(sum) {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
	}

	return ErrWaitForTransactionTimeout
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
	return ed25519.NewKeyFromSeed(signingKey).Public().(ed25519.PublicKey) //nolint:forcetypeassert
}

// GenerateKeyPair generates ed25519 (signing key, verifying) key pair
func GenerateKeyPair() ([]byte, []byte, error) {
	seed := make([]byte, ed25519.SeedSize)
	if _, err := io.ReadFull(rand.Reader, seed); err != nil {
		return nil, nil, err
	}

	return seed, GetVerificationKeyFromSigningKey(seed), nil
}

// GetKeyHash gets Cardano key hash from verification key
func GetKeyHash(verificationKey []byte) (string, error) {
	hasher, err := blake2b.New(28, nil)
	if err != nil {
		return "", err
	}

	if _, err := hasher.Write(verificationKey); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
