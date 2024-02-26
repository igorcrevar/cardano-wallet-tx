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
	addressFile         = "payment.addr"
)

type Wallet struct {
	directory    string
	testNetMagic uint

	address         string
	verificationKey []byte
	signingKey      []byte
	keyHash         string
}

func NewWallet(directory string, testNetMagic uint) *Wallet {
	return &Wallet{
		directory:    directory,
		testNetMagic: testNetMagic,
	}
}

func (w *Wallet) Create(forceCreate bool) error {
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
	if err != nil {
		return err
	}

	_, err = runCommand(resolveCardanoCliBinary(), append([]string{
		"address", "build",
		"--payment-verification-key-file", w.GetVerificationKeyPath(),
		"--out-file", w.GetAddressFilePath(),
	}, getTestNetMagicArgs(w.testNetMagic)...))

	return err
}

func (w *Wallet) Load() error {
	verificationKey, err := NewKey(w.GetVerificationKeyPath())
	if err != nil {
		return err
	}

	w.verificationKey, err = verificationKey.GetKeyBytes()
	if err != nil {
		return err
	}

	signingKey, err := NewKey(w.GetSigningKeyPath())
	if err != nil {
		return err
	}

	w.signingKey, err = signingKey.GetKeyBytes()
	if err != nil {
		return err
	}

	bytes, err := os.ReadFile(w.GetAddressFilePath())
	if err != nil {
		return err
	}

	w.address = string(bytes)

	result, err := runCommand(resolveCardanoCliBinary(), []string{
		"address", "key-hash",
		"--payment-verification-key-file", w.GetVerificationKeyPath(),
	})
	if err != nil {
		return err
	}

	w.keyHash = strings.Trim(result, "\n")

	return nil
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

func (w Wallet) GetSigningKeyPath() string {
	return path.Join(w.directory, signingKeyFile)
}

func (w Wallet) GetVerificationKeyPath() string {
	return path.Join(w.directory, verificationKeyFile)
}

func (w Wallet) GetAddressFilePath() string {
	return path.Join(w.directory, addressFile)
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
