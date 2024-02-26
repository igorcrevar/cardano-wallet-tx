package core

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path"
	"strings"

	"github.com/fxamacker/cbor/v2"
)

const (
	verificationKeyFile = "payment.vkey"
	signingKeyFile      = "payment.skey"
)

type Wallet struct {
	address         string
	verificationKey []byte
	signingKey      []byte
	keyHash         string
}

func NewWallet(address string, verificationKey []byte, signingKey []byte, keyHash string) *Wallet {
	return &Wallet{
		address:         address,
		verificationKey: verificationKey,
		signingKey:      signingKey,
		keyHash:         keyHash,
	}
}

func (w Wallet) GetAddress() string {
	return w.address
}

func (w Wallet) GetVerificationKey() []byte {
	return w.verificationKey
}

func (w Wallet) GetSigningKey() []byte {
	return w.signingKey
}

func (w Wallet) GetKeyHash() string {
	return w.keyHash
}

func (w Wallet) SaveVerificationKeyToFile(filePath string) error {
	verificationKey, err := NewKeyFromBytes("PaymentVerificationKeyShelley_ed25519", "Payment Verification Key", w.verificationKey)
	if err != nil {
		return err
	}

	return verificationKey.WriteToFile(filePath)
}

func (w Wallet) SaveSigningKeyToFile(filePath string) error {
	signingKey, err := NewKeyFromBytes("PaymentSigningKeyShelley_ed25519", "Payment Signing Key", w.signingKey)
	if err != nil {
		return err
	}

	return signingKey.WriteToFile(filePath)
}

type WalletBuilder struct {
	directory    string
	testNetMagic uint
}

func NewWalletBuilder(directory string, testNetMagic uint) *WalletBuilder {
	return &WalletBuilder{
		directory:    directory,
		testNetMagic: testNetMagic,
	}
}

func (w *WalletBuilder) Create(forceCreate bool) error {
	if !forceCreate && isFileOrDirExists(w.GetVerificationKeyPath()) && isFileOrDirExists(w.GetSigningKeyPath()) {
		return nil
	}

	if err := createDirectoryIfNotExists(w.directory); err != nil {
		return err
	}

	_, err := runCommand(resolveCardanoCliBinary(), []string{
		"address", "key-gen",
		"--verification-key-file", w.GetVerificationKeyPath(),
		"--signing-key-file", w.GetSigningKeyPath(),
	})
	return err
}

func (w *WalletBuilder) Load() (*Wallet, error) {
	verificationKey, err := NewKey(w.GetVerificationKeyPath())
	if err != nil {
		return nil, err
	}

	verificationKeyBytes, err := verificationKey.GetKeyBytes()
	if err != nil {
		return nil, err
	}

	signingKey, err := NewKey(w.GetSigningKeyPath())
	if err != nil {
		return nil, err
	}

	signingKeyBytes, err := signingKey.GetKeyBytes()
	if err != nil {
		return nil, err
	}

	resultAddress, err := runCommand(resolveCardanoCliBinary(), append([]string{
		"address", "build",
		"--payment-verification-key-file", w.GetVerificationKeyPath(),
	}, getTestNetMagicArgs(w.testNetMagic)...))
	if err != nil {
		return nil, err
	}

	resultKeyHash, err := runCommand(resolveCardanoCliBinary(), []string{
		"address", "key-hash",
		"--payment-verification-key-file", w.GetVerificationKeyPath(),
	})
	if err != nil {
		return nil, err
	}

	address := strings.Trim(resultAddress, "\n")
	keyHash := strings.Trim(resultKeyHash, "\n")

	return NewWallet(address, verificationKeyBytes, signingKeyBytes, keyHash), nil
}

func (w WalletBuilder) GetSigningKeyPath() string {
	return path.Join(w.directory, signingKeyFile)
}

func (w WalletBuilder) GetVerificationKeyPath() string {
	return path.Join(w.directory, verificationKeyFile)
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

	if err := os.WriteFile(filePath, bytes, 0755); err != nil {
		return err
	}

	return nil
}
