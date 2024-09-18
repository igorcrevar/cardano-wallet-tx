package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrInvalidAddressInfo = errors.New("invalid address info")
)

type AddressInfo struct {
	Address  string `json:"address"`
	Base16   string `json:"base16"`
	Encoding string `json:"encoding"`
	Era      string `json:"era"`
	Type     string `json:"type"`
}

type CliUtils struct {
	cardanoCliBinary string
}

func NewCliUtils(cardanoCliBinary string) CliUtils {
	return CliUtils{
		cardanoCliBinary: cardanoCliBinary,
	}
}

// GetPolicyScriptAddress get address for policy script
func (cu CliUtils) GetPolicyScriptAddress(
	testNetMagic uint, policyScript *PolicyScript, policyScriptStake ...*PolicyScript,
) (string, error) {
	baseDirectory, err := os.MkdirTemp("", "ps-multisig-addr")
	if err != nil {
		return "", err
	}

	defer os.RemoveAll(baseDirectory)

	policyScriptBytes, err := json.Marshal(policyScript)
	if err != nil {
		return "", err
	}

	policyScriptFilePath := filepath.Join(baseDirectory, "policy-script.json")
	if err := os.WriteFile(policyScriptFilePath, policyScriptBytes, FilePermission); err != nil {
		return "", err
	}

	args := []string{
		"address", "build",
		"--payment-script-file", policyScriptFilePath,
	}

	if len(policyScriptStake) > 0 {
		policyScriptStakeBytes, err := json.Marshal(policyScriptStake[0])
		if err != nil {
			return "", err
		}

		policyScriptStakeFilePath := filepath.Join(baseDirectory, "policy-script-stake.json")
		if err := os.WriteFile(policyScriptStakeFilePath, policyScriptStakeBytes, FilePermission); err != nil {
			return "", err
		}

		args = append(args, "--stake-script-file", policyScriptStakeFilePath)
	}

	response, err := runCommand(cu.cardanoCliBinary, append(args, getTestNetMagicArgs(testNetMagic)...))
	if err != nil {
		return "", err
	}

	return strings.Trim(response, "\n"), nil
}

// GetPolicyID returns policy id
func (cu CliUtils) GetPolicyID(policyScript any) (string, error) {
	baseDirectory, err := os.MkdirTemp("", "ps-policy-id")
	if err != nil {
		return "", err
	}

	defer os.RemoveAll(baseDirectory)

	policyScriptBytes, err := json.Marshal(policyScript)
	if err != nil {
		return "", err
	}

	policyScriptFilePath := filepath.Join(baseDirectory, "policy-script.json")
	if err := os.WriteFile(policyScriptFilePath, policyScriptBytes, FilePermission); err != nil {
		return "", err
	}

	response, err := runCommand(cu.cardanoCliBinary, []string{
		"transaction", "policyid", "--script-file", policyScriptFilePath,
	})
	if err != nil {
		return "", err
	}

	return strings.Trim(response, "\n"), nil
}

// GetAddressInfo returns address info if string representation for address is valid or error
func (cu CliUtils) GetAddressInfo(address string) (AddressInfo, error) {
	var ai AddressInfo

	res, err := runCommand(cu.cardanoCliBinary, []string{
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

// GetWalletAddress returns address and stake address for wallet (if wallet is stake wallet)
func (cu CliUtils) GetWalletAddress(wallet IWallet, testNetMagic uint) (addr string, stakeAddr string, err error) {
	baseDirectory, err := os.MkdirTemp("", "get-address")
	if err != nil {
		return "", "", err
	}

	defer os.RemoveAll(baseDirectory)

	key, err := NewKeyFromBytes(
		PaymentVerificationKeyShelley, PaymentVerificationKeyShelleyDesc, wallet.GetVerificationKey())
	if err != nil {
		return "", "", nil
	}

	verificationFilePath := filepath.Join(baseDirectory, "ver.key")
	stakeVerificationFilePath := filepath.Join(baseDirectory, "stake.key")

	if err = key.WriteToFile(verificationFilePath); err != nil {
		return "", "", nil
	}

	// enterprise address
	if len(wallet.GetStakeVerificationKey()) == 0 {
		addr, err = runCommand(cu.cardanoCliBinary, append([]string{
			"address", "build",
			"--payment-verification-key-file", verificationFilePath,
		}, getTestNetMagicArgs(testNetMagic)...))

		return strings.Trim(addr, "\n"), strings.Trim(stakeAddr, "\n"), err
	}

	stakeKey, err := NewKeyFromBytes(
		StakeVerificationKeyShelley, StakeVerificationKeyShelleyDesc, wallet.GetStakeVerificationKey())
	if err != nil {
		return "", "", nil
	}

	if err = stakeKey.WriteToFile(stakeVerificationFilePath); err != nil {
		return "", "", nil
	}

	addr, err = runCommand(cu.cardanoCliBinary, append([]string{
		"address", "build",
		"--payment-verification-key-file", verificationFilePath,
		"--stake-verification-key-file", stakeVerificationFilePath,
	}, getTestNetMagicArgs(testNetMagic)...))
	if err != nil {
		return "", "", err
	}

	stakeAddr, err = runCommand(cu.cardanoCliBinary, append([]string{
		"stake-address", "build",
		"--stake-verification-key-file", stakeVerificationFilePath,
	}, getTestNetMagicArgs(testNetMagic)...))

	return strings.Trim(addr, "\n"), strings.Trim(stakeAddr, "\n"), err
}

func (cu CliUtils) GetKeyHash(verificationKeyPath string) (string, error) {
	resultKeyHash, err := runCommand(cu.cardanoCliBinary, []string{
		"address", "key-hash",
		"--payment-verification-key-file", verificationKeyPath,
	})
	if err != nil {
		return "", err
	}

	return strings.Trim(resultKeyHash, "\n"), nil
}

// GetTxHash gets hash from transaction cbor slice
func (cu CliUtils) GetTxHash(txRaw []byte) (string, error) {
	baseDirectory, err := os.MkdirTemp("", "tx-hash-retriever")
	if err != nil {
		return "", err
	}

	defer os.RemoveAll(baseDirectory)

	return cu.getTxHash(txRaw, baseDirectory)
}

func (cu CliUtils) getTxHash(txRaw []byte, baseDirectory string) (string, error) {
	txFilePath := filepath.Join(baseDirectory, "tx.tmp")

	txBytes, err := TransactionUnwitnessedRaw(txRaw).ToJSON()
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(txFilePath, txBytes, FilePermission); err != nil {
		return "", err
	}

	args := []string{
		"transaction", "txid",
		"--tx-body-file", txFilePath}

	res, err := runCommand(cu.cardanoCliBinary, args)
	if err != nil {
		return "", err
	}

	return strings.Trim(res, "\n"), err
}
