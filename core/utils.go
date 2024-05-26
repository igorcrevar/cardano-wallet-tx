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

type IsRecoverableErrorFn func(err error) bool

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

// GetUtxosSum returns big.Int sum of all utxos
func GetUtxosSum(utxos []Utxo) *big.Int {
	sum := big.NewInt(0)
	for _, utxo := range utxos {
		sum.Add(sum, new(big.Int).SetUint64(utxo.Amount))
	}

	return sum
}

// WaitForAmount waits for address to have amount specified by cmpHandler
func WaitForAmount(ctx context.Context, txRetriever IUTxORetriever,
	addr string, cmpHandler func(*big.Int) bool, numRetries int, waitTime time.Duration,
	isRecoverableError ...IsRecoverableErrorFn,
) error {
	return ExecuteWithRetry(ctx, numRetries, waitTime, func() (bool, error) {
		utxos, err := txRetriever.GetUtxos(ctx, addr)

		return err == nil && cmpHandler(GetUtxosSum(utxos)), err
	}, isRecoverableError...)
}

// WaitForTxHashInUtxos waits until tx with txHash occurs in addr utxos
func WaitForTxHashInUtxos(ctx context.Context, txRetriever IUTxORetriever,
	addr string, txHash string, numRetries int, waitTime time.Duration,
	isRecoverableError ...IsRecoverableErrorFn,
) error {
	return ExecuteWithRetry(ctx, numRetries, waitTime, func() (bool, error) {
		utxos, err := txRetriever.GetUtxos(ctx, addr)
		if err != nil {
			return false, err
		}

		for _, x := range utxos {
			if x.Hash == txHash {
				return true, nil
			}
		}

		return false, nil
	}, isRecoverableError...)
}

// WaitForTransaction waits for transaction to be included in block
func WaitForTransaction(ctx context.Context, txRetriever ITxRetriever,
	hash string, numRetries int, waitTime time.Duration,
	isRecoverableError ...IsRecoverableErrorFn,
) (res map[string]interface{}, err error) {
	err = ExecuteWithRetry(ctx, numRetries, waitTime, func() (bool, error) {
		res, err = txRetriever.GetTxByHash(ctx, hash)

		return err == nil && res != nil, err
	}, isRecoverableError...)

	return res, err
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

// GetKeyHashBytes gets Cardano key hash from verification key
func GetKeyHashBytes(verificationKey []byte) ([]byte, error) {
	hasher, err := blake2b.New(28, nil)
	if err != nil {
		return nil, err
	}

	if _, err := hasher.Write(verificationKey); err != nil {
		return nil, err
	}

	return hasher.Sum(nil), nil
}

// GetKeyHash gets Cardano key hash string from verification key
func GetKeyHash(verificationKey []byte) (string, error) {
	bytes, err := GetKeyHashBytes(verificationKey)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// ExecuteWithRetry executes custom executeFn with retries if needed
func ExecuteWithRetry(ctx context.Context,
	numRetries int, waitTime time.Duration,
	executeFn func() (bool, error),
	isRecoverableError ...IsRecoverableErrorFn,
) error {
	for count := 0; count < numRetries; count++ {
		stop, err := executeFn()
		if err != nil {
			if len(isRecoverableError) == 0 || !isRecoverableError[0](err) {
				return err
			}
		} else if stop {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
	}

	return ErrWaitForTransactionTimeout
}
