package core

import (
	"fmt"
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
	verificationKeyBytes, err := getKeyBytes(w.GetVerificationKeyPath())
	if err != nil {
		return nil, err
	}

	signingKeyBytes, err := getKeyBytes(w.GetSigningKeyPath())
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

type StakeWalletBuilder struct {
	*WalletBuilder
}

func NewStakeWalletBuilder(directory string, testNetMagic uint) *StakeWalletBuilder {
	return &StakeWalletBuilder{
		WalletBuilder: NewWalletBuilder(directory, testNetMagic),
	}
}

func (w *StakeWalletBuilder) Create(forceCreate bool) error {
	if !forceCreate && isFileOrDirExists(w.GetVerificationKeyPath()) && isFileOrDirExists(w.GetSigningKeyPath()) {
		if isFileOrDirExists(w.GetStakeVerificationKeyPath()) && isFileOrDirExists(w.GetStakeSigningKeyPath()) {
			return nil
		}

		return fmt.Errorf("directory %s contains only payment key pair", w.WalletBuilder.directory)
	}

	if err := w.WalletBuilder.Create(forceCreate); err != nil {
		return err
	}

	_, err := runCommand(resolveCardanoCliBinary(), []string{
		"stake-address", "key-gen",
		"--verification-key-file", w.GetStakeVerificationKeyPath(),
		"--signing-key-file", w.GetStakeSigningKeyPath(),
	})
	return err
}

func (w *StakeWalletBuilder) Load() (*StakeWallet, error) {
	verificationKeyBytes, err := getKeyBytes(w.GetVerificationKeyPath())
	if err != nil {
		return nil, err
	}

	signingKeyBytes, err := getKeyBytes(w.GetSigningKeyPath())
	if err != nil {
		return nil, err
	}

	stakeVerificationKeyBytes, err := getKeyBytes(w.GetStakeVerificationKeyPath())
	if err != nil {
		return nil, err
	}

	stakeSigningKeyBytes, err := getKeyBytes(w.GetStakeSigningKeyPath())
	if err != nil {
		return nil, err
	}

	resultAddress, err := runCommand(resolveCardanoCliBinary(), append([]string{
		"address", "build",
		"--payment-verification-key-file", w.GetVerificationKeyPath(),
		"--stake-verification-key-file", w.GetStakeVerificationKeyPath(),
	}, getTestNetMagicArgs(w.testNetMagic)...))
	if err != nil {
		return nil, err
	}

	resultStakeAddress, err := runCommand(resolveCardanoCliBinary(), append([]string{
		"stake-address", "build",
		"--stake-verification-key-file", w.GetStakeVerificationKeyPath(),
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
	stakeAddress := strings.Trim(resultStakeAddress, "\n")
	keyHash := strings.Trim(resultKeyHash, "\n")

	return NewStakeWallet(address, verificationKeyBytes, signingKeyBytes, keyHash,
		stakeAddress, stakeVerificationKeyBytes, stakeSigningKeyBytes), nil
}

func (w StakeWalletBuilder) GetStakeSigningKeyPath() string {
	return path.Join(w.WalletBuilder.directory, stakeSigningKeyFile)
}

func (w StakeWalletBuilder) GetStakeVerificationKeyPath() string {
	return path.Join(w.WalletBuilder.directory, stakeVerificationKeyFile)
}
