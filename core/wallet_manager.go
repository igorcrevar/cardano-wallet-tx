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

type WalletManager struct {
}

func NewWalletManager() *WalletManager {
	return &WalletManager{}
}

func (w *WalletManager) Create(directory string, forceCreate bool) (IWallet, error) {
	dir := walletManagerDirectory(directory)

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

func (w *WalletManager) Load(directory string) (IWallet, error) {
	dir := walletManagerDirectory(directory)

	verificationKeyBytes, err := getKeyBytes(dir.GetVerificationKeyPath())
	if err != nil {
		return nil, err
	}

	signingKeyBytes, err := getKeyBytes(dir.GetSigningKeyPath())
	if err != nil {
		return nil, err
	}

	keyHash, err := GetKeyHashCli(dir.GetVerificationKeyPath())
	if err != nil {
		return nil, err
	}

	return NewWallet(verificationKeyBytes, signingKeyBytes, keyHash), nil
}

type StakeWalletManager struct {
}

func NewStakeWalletManager() *StakeWalletManager {
	return &StakeWalletManager{}
}

func (w *StakeWalletManager) Create(directory string, forceCreate bool) (IWallet, error) {
	dir := walletManagerDirectory(directory)

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

func (w *StakeWalletManager) Load(directory string) (IWallet, error) {
	dir := walletManagerDirectory(directory)

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

	keyHash, err := GetKeyHashCli(dir.GetVerificationKeyPath())
	if err != nil {
		return nil, err
	}

	return NewStakeWallet(verificationKeyBytes, signingKeyBytes, keyHash,
		stakeVerificationKeyBytes, stakeSigningKeyBytes), nil
}

type walletManagerDirectory string

func (w walletManagerDirectory) GetSigningKeyPath() string {
	return path.Join(string(w), signingKeyFile)
}

func (w walletManagerDirectory) GetVerificationKeyPath() string {
	return path.Join(string(w), verificationKeyFile)
}

func (w walletManagerDirectory) GetStakeSigningKeyPath() string {
	return path.Join(string(w), stakeSigningKeyFile)
}

func (w walletManagerDirectory) GetStakeVerificationKeyPath() string {
	return path.Join(string(w), stakeVerificationKeyFile)
}

func (w walletManagerDirectory) ArePaymentFilesExist() bool {
	return isFileOrDirExists(w.GetVerificationKeyPath()) && isFileOrDirExists(w.GetSigningKeyPath())
}

func (w walletManagerDirectory) AreStakeFilesExist() bool {
	return isFileOrDirExists(w.GetStakeVerificationKeyPath()) && isFileOrDirExists(w.GetStakeSigningKeyPath())
}

func (w walletManagerDirectory) CreateDirectoryIfNotExists() error {
	if _, err := os.Stat(string(w)); os.IsNotExist(err) {
		// If the directory doesn't exist, create it
		return os.MkdirAll(string(w), FilePermission)
	}

	return nil
}

// GetWalletAddress returns address and stake address for wallet (if wallet is stake wallet)
func GetWalletAddress(wallet IWallet, testNetMagic uint) (addr string, stakeAddr string, err error) {
	baseDirectory, err := os.MkdirTemp("", "get-address")
	if err != nil {
		return "", "", err
	}

	defer func() {
		os.RemoveAll(baseDirectory)
		os.Remove(baseDirectory)
	}()

	dir := walletManagerDirectory(baseDirectory)

	err = SaveKeyBytesToFile(wallet.GetVerificationKey(), dir.GetVerificationKeyPath(), false, false)
	if err != nil {
		return "", "", nil
	}

	if len(wallet.GetStakeVerificationKey()) == 0 {
		addr, err = runCommand(resolveCardanoCliBinary(), append([]string{
			"address", "build",
			"--payment-verification-key-file", dir.GetVerificationKeyPath(),
		}, getTestNetMagicArgs(testNetMagic)...))
	} else {
		err = SaveKeyBytesToFile(wallet.GetStakeVerificationKey(), dir.GetStakeVerificationKeyPath(), false, true)
		if err != nil {
			return "", "", nil
		}

		addr, err = runCommand(resolveCardanoCliBinary(), append([]string{
			"address", "build",
			"--payment-verification-key-file", dir.GetVerificationKeyPath(),
			"--stake-verification-key-file", dir.GetStakeVerificationKeyPath(),
		}, getTestNetMagicArgs(testNetMagic)...))
		if err != nil {
			return "", "", err
		}

		stakeAddr, err = runCommand(resolveCardanoCliBinary(), append([]string{
			"stake-address", "build",
			"--stake-verification-key-file", dir.GetStakeVerificationKeyPath(),
		}, getTestNetMagicArgs(testNetMagic)...))
	}

	return strings.Trim(addr, "\n"), strings.Trim(stakeAddr, "\n"), err
}

func GetKeyHashCli(verificationKeyPath string) (string, error) {
	resultKeyHash, err := runCommand(resolveCardanoCliBinary(), []string{
		"address", "key-hash",
		"--payment-verification-key-file", verificationKeyPath,
	})
	if err != nil {
		return "", err
	}

	return strings.Trim(resultKeyHash, "\n"), nil
}
