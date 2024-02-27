package core

import (
	"fmt"
	"os"
	"path"
	"strings"
)

const (
	verificationKeyFile      = "payment.vkey"
	signingKeyFile           = "payment.skey"
	stakeVerificationKeyFile = "stake.vkey"
	stakeSigningKeyFile      = "stake.skey"
)

type WalletBuilder struct {
	testNetMagic uint
}

func NewWalletBuilder(testNetMagic uint) *WalletBuilder {
	return &WalletBuilder{
		testNetMagic: testNetMagic,
	}
}

func (w *WalletBuilder) Create(directory string, forceCreate bool) (IWallet, error) {
	dir := walletBuilderDirectory(directory)

	if !forceCreate && dir.ArePaymentFilesExist() {
		return w.Load(directory)
	}

	if err := dir.CreateDirectoryIfNotExists(); err != nil {
		return nil, err
	}

	_, err := runCommand(resolveCardanoCliBinary(), []string{
		"address", "key-gen",
		"--verification-key-file", dir.GetVerificationKeyPath(),
		"--signing-key-file", dir.GetSigningKeyPath(),
	})
	if err != nil {
		return nil, err
	}

	return w.Load(directory)
}

func (w *WalletBuilder) Load(directory string) (*Wallet, error) {
	dir := walletBuilderDirectory(directory)

	verificationKeyBytes, err := getKeyBytes(dir.GetVerificationKeyPath())
	if err != nil {
		return nil, err
	}

	signingKeyBytes, err := getKeyBytes(dir.GetSigningKeyPath())
	if err != nil {
		return nil, err
	}

	resultAddress, err := runCommand(resolveCardanoCliBinary(), append([]string{
		"address", "build",
		"--payment-verification-key-file", dir.GetVerificationKeyPath(),
	}, getTestNetMagicArgs(w.testNetMagic)...))
	if err != nil {
		return nil, err
	}

	resultKeyHash, err := runCommand(resolveCardanoCliBinary(), []string{
		"address", "key-hash",
		"--payment-verification-key-file", dir.GetVerificationKeyPath(),
	})
	if err != nil {
		return nil, err
	}

	address := strings.Trim(resultAddress, "\n")
	keyHash := strings.Trim(resultKeyHash, "\n")

	return NewWallet(address, verificationKeyBytes, signingKeyBytes, keyHash), nil
}

type StakeWalletBuilder struct {
	testNetMagic uint
}

func NewStakeWalletBuilder(testNetMagic uint) *StakeWalletBuilder {
	return &StakeWalletBuilder{
		testNetMagic: testNetMagic,
	}
}

func (w *StakeWalletBuilder) Create(directory string, forceCreate bool) (IWallet, error) {
	dir := walletBuilderDirectory(directory)

	if !forceCreate && dir.ArePaymentFilesExist() {
		if dir.AreStakeFilesExist() {
			return w.Load(directory)
		}

		return nil, fmt.Errorf("directory %s contains only payment key pair", directory)
	}

	if err := dir.CreateDirectoryIfNotExists(); err != nil {
		return nil, err
	}

	_, err := runCommand(resolveCardanoCliBinary(), []string{
		"address", "key-gen",
		"--verification-key-file", dir.GetVerificationKeyPath(),
		"--signing-key-file", dir.GetSigningKeyPath(),
	})
	if err != nil {
		return nil, err
	}

	_, err = runCommand(resolveCardanoCliBinary(), []string{
		"stake-address", "key-gen",
		"--verification-key-file", dir.GetStakeVerificationKeyPath(),
		"--signing-key-file", dir.GetStakeSigningKeyPath(),
	})
	if err != nil {
		return nil, err
	}

	return w.Load(directory)
}

func (w *StakeWalletBuilder) Load(directory string) (*StakeWallet, error) {
	dir := walletBuilderDirectory(directory)

	verificationKeyBytes, err := getKeyBytes(dir.GetVerificationKeyPath())
	if err != nil {
		return nil, err
	}

	signingKeyBytes, err := getKeyBytes(dir.GetSigningKeyPath())
	if err != nil {
		return nil, err
	}

	stakeVerificationKeyBytes, err := getKeyBytes(dir.GetStakeVerificationKeyPath())
	if err != nil {
		return nil, err
	}

	stakeSigningKeyBytes, err := getKeyBytes(dir.GetStakeSigningKeyPath())
	if err != nil {
		return nil, err
	}

	resultAddress, err := runCommand(resolveCardanoCliBinary(), append([]string{
		"address", "build",
		"--payment-verification-key-file", dir.GetVerificationKeyPath(),
		"--stake-verification-key-file", dir.GetStakeVerificationKeyPath(),
	}, getTestNetMagicArgs(w.testNetMagic)...))
	if err != nil {
		return nil, err
	}

	resultStakeAddress, err := runCommand(resolveCardanoCliBinary(), append([]string{
		"stake-address", "build",
		"--stake-verification-key-file", dir.GetStakeVerificationKeyPath(),
	}, getTestNetMagicArgs(w.testNetMagic)...))
	if err != nil {
		return nil, err
	}

	resultKeyHash, err := runCommand(resolveCardanoCliBinary(), []string{
		"address", "key-hash",
		"--payment-verification-key-file", dir.GetVerificationKeyPath(),
	})
	if err != nil {
		return nil, err
	}

	address := strings.Trim(resultAddress, "\n")
	stakeAddress := strings.Trim(resultStakeAddress, "\n")
	keyHash := strings.Trim(resultKeyHash, "\n")

	return NewStakeWallet(address, verificationKeyBytes, signingKeyBytes, keyHash,
		stakeAddress, stakeVerificationKeyBytes, stakeSigningKeyBytes), nil
}

type walletBuilderDirectory string

func (w walletBuilderDirectory) GetSigningKeyPath() string {
	return path.Join(string(w), signingKeyFile)
}

func (w walletBuilderDirectory) GetVerificationKeyPath() string {
	return path.Join(string(w), verificationKeyFile)
}

func (w walletBuilderDirectory) GetStakeSigningKeyPath() string {
	return path.Join(string(w), stakeSigningKeyFile)
}

func (w walletBuilderDirectory) GetStakeVerificationKeyPath() string {
	return path.Join(string(w), stakeVerificationKeyFile)
}

func (w walletBuilderDirectory) ArePaymentFilesExist() bool {
	return isFileOrDirExists(w.GetVerificationKeyPath()) && isFileOrDirExists(w.GetSigningKeyPath())
}

func (w walletBuilderDirectory) AreStakeFilesExist() bool {
	return isFileOrDirExists(w.GetStakeVerificationKeyPath()) && isFileOrDirExists(w.GetStakeSigningKeyPath())
}

func (w walletBuilderDirectory) CreateDirectoryIfNotExists() error {
	if _, err := os.Stat(string(w)); os.IsNotExist(err) {
		// If the directory doesn't exist, create it
		return os.MkdirAll(string(w), 0755)
	}

	return nil
}
