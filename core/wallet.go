package core

import (
	"encoding/hex"
	"encoding/json"
	"os"

	"github.com/fxamacker/cbor/v2"
)

type Wallet struct {
	VerificationKey      []byte `json:"verificationKey"`
	SigningKey           []byte `json:"signingKey"`
	StakeVerificationKey []byte `json:"stakeVerificationKey"`
	StakeSigningKey      []byte `json:"stakeSigningKey"`
}

func NewWallet(verificationKey []byte, signingKey []byte) *Wallet {
	return &Wallet{
		VerificationKey: verificationKey,
		SigningKey:      signingKey,
	}
}

func NewStakeWallet(verificationKey []byte, signingKey []byte,
	stakeVerificationKey []byte, stakeSigningKey []byte) *Wallet {
	return &Wallet{
		StakeVerificationKey: stakeVerificationKey,
		StakeSigningKey:      stakeSigningKey,
		VerificationKey:      verificationKey,
		SigningKey:           signingKey,
	}
}

func (w Wallet) GetVerificationKey() []byte {
	return w.VerificationKey
}

func (w Wallet) GetSigningKey() []byte {
	return w.SigningKey
}

func (w Wallet) GetStakeVerificationKey() []byte {
	return w.StakeVerificationKey
}

func (w Wallet) GetStakeSigningKey() []byte {
	return w.StakeSigningKey
}

type Key struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Hex         string `json:"cborHex"`
}

func NewKey(filePath string) (Key, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return Key{}, err
	}

	var key Key

	if err := json.Unmarshal(bytes, &key); err != nil {
		return Key{}, err
	}

	return key, nil
}

func NewKeyFromBytes(keyType string, desc string, bytes []byte) (Key, error) {
	cborBytes, err := cbor.Marshal(bytes)
	if err != nil {
		return Key{}, err
	}

	return Key{
		Type:        keyType,
		Description: desc,
		Hex:         hex.EncodeToString(cborBytes),
	}, nil
}

func (k Key) GetKeyBytes() ([]byte, error) {
	bytes, err := hex.DecodeString(k.Hex)
	if err != nil {
		return nil, err
	}

	var result []byte

	if err := cbor.Unmarshal(bytes, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (k Key) WriteToFile(filePath string) error {
	bytes, err := json.Marshal(k)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filePath, bytes, FilePermission); err != nil {
		return err
	}

	return nil
}

func SaveKeyBytesToFile(keyBytes []byte, filePath string, isSigningKey bool, isStakeKey bool) error {
	var title, desc string

	if isStakeKey {
		if isSigningKey {
			title, desc = "StakeSigningKeyShelley_ed25519", "Stake Signing Key"
		} else {
			title, desc = "StakeVerificationKeyShelley_ed25519", "Stake Verification Key"
		}
	} else {
		if isSigningKey {
			title, desc = "PaymentSigningKeyShelley_ed25519", "Payment Signing Key"
		} else {
			title, desc = "PaymentVerificationKeyShelley_ed25519", "Payment Verification Key"
		}
	}

	key, err := NewKeyFromBytes(title, desc, keyBytes)
	if err != nil {
		return err
	}

	return key.WriteToFile(filePath)
}

func getKeyBytes(filePath string) ([]byte, error) {
	key, err := NewKey(filePath)
	if err != nil {
		return nil, err
	}

	return key.GetKeyBytes()
}
