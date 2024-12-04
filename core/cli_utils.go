package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/igorcrevar/go-cardano-tx/core/bech32"
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
		return ai, errors.Join(ErrInvalidAddressData, err)
	}

	if err := json.Unmarshal([]byte(strings.Trim(res, "\n")), &ai); err != nil {
		return ai, errors.Join(ErrInvalidAddressData, err)
	}

	return ai, nil
}

// GetWalletAddress returns address and stake address for wallet (if wallet is stake wallet)
func (cu CliUtils) GetWalletAddress(
	verificationKey, stakeVerificationKey []byte, testNetMagic uint,
) (addr string, stakeAddr string, err error) {
	bech32String, err := getBech32Key(verificationKey, "addr_vk")
	if err != nil {
		return "", "", err
	}

	// enterprise address
	if len(stakeVerificationKey) == 0 {
		addr, err = runCommand(cu.cardanoCliBinary, append([]string{
			"address", "build",
			"--payment-verification-key", bech32String,
		}, getTestNetMagicArgs(testNetMagic)...))

		return strings.Trim(addr, "\n"), strings.Trim(stakeAddr, "\n"), err
	}

	bech32StakeString, err := getBech32Key(stakeVerificationKey, "stake_vk")
	if err != nil {
		return "", "", err
	}

	addr, err = runCommand(cu.cardanoCliBinary, append([]string{
		"address", "build",
		"--payment-verification-key", bech32String,
		"--stake-verification-key", bech32StakeString,
	}, getTestNetMagicArgs(testNetMagic)...))
	if err != nil {
		return "", "", err
	}

	stakeAddr, err = runCommand(cu.cardanoCliBinary, append([]string{
		"stake-address", "build",
		"--stake-verification-key", bech32StakeString,
	}, getTestNetMagicArgs(testNetMagic)...))

	return strings.Trim(addr, "\n"), strings.Trim(stakeAddr, "\n"), err
}

func (cu CliUtils) GetKeyHash(key []byte) (string, error) {
	bech32String, err := getBech32Key(key, "addr_vk")
	if err != nil {
		return "", err
	}

	resultKeyHash, err := runCommand(cu.cardanoCliBinary, []string{
		"address", "key-hash",
		"--payment-verification-key", bech32String,
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

	txBytes, err := transactionUnwitnessedRaw(txRaw).ToJSON()
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

func getBech32Key(key []byte, prefix string) (string, error) {
	converted, err := bech32.ConvertBits(key, 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("error converting bits: %w", err)
	}

	bech32String, err := bech32.Encode(prefix, converted)
	if err != nil {
		return "", fmt.Errorf("error encoding to Bech32: %w", err)
	}

	return bech32String, nil
}
