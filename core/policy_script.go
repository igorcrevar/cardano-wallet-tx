package core

import (
	"encoding/json"
	"os"
	"path"
	"strings"
)

type keyHashSig struct {
	Type    string `json:"type"`
	KeyHash string `json:"keyHash"`
}

type PolicyScript struct {
	PolicyScript []byte `json:"ps"`
	Count        int    `json:"cnt"`
}

func NewPolicyScript(keyHashes []string, atLeastSignersCount int) (*PolicyScript, error) {
	policyScript, err := createPolicyScript(keyHashes, atLeastSignersCount)
	if err != nil {
		return nil, err
	}

	return &PolicyScript{
		PolicyScript: policyScript,
		Count:        len(keyHashes),
	}, nil
}

func NewPolicyScriptFromKeyHash(keyHash string) (*PolicyScript, error) {
	policyScript, err := json.MarshalIndent(keyHashSig{
		Type:    "sig",
		KeyHash: keyHash,
	}, "", "  ")
	if err != nil {
		return nil, err
	}

	return &PolicyScript{
		PolicyScript: policyScript,
		Count:        1,
	}, nil
}

func (ps PolicyScript) CreateMultiSigAddress(testNetMagic uint) (string, error) {
	baseDirectory, err := os.MkdirTemp("", "cardano-multisig-addr")
	if err != nil {
		return "", err
	}

	defer func() {
		os.RemoveAll(baseDirectory)
		os.Remove(baseDirectory)
	}()

	policyScriptFilePath := path.Join(baseDirectory, "policy-script.json")
	if err := os.WriteFile(policyScriptFilePath, ps.PolicyScript, FilePermission); err != nil {
		return "", err
	}

	response, err := runCommand(resolveCardanoCliBinary(), append([]string{
		"address", "build",
		"--payment-script-file", policyScriptFilePath,
	}, getTestNetMagicArgs(testNetMagic)...))
	if err != nil {
		return "", err
	}

	return strings.Trim(response, "\n"), nil
}

// GetPolicyID returns policy id (ScriptStakeCredentialType hash)
func (ps PolicyScript) GetPolicyID() (string, error) {
	baseDirectory, err := os.MkdirTemp("", "cardano-policy-id")
	if err != nil {
		return "", err
	}

	defer func() {
		os.RemoveAll(baseDirectory)
		os.Remove(baseDirectory)
	}()

	policyScriptFilePath := path.Join(baseDirectory, "policy-script.json")
	if err := os.WriteFile(policyScriptFilePath, ps.PolicyScript, FilePermission); err != nil {
		return "", err
	}

	return getPolicyID(policyScriptFilePath)
}

func (ps PolicyScript) GetPolicyScript() []byte {
	return ps.PolicyScript
}

func (ps PolicyScript) GetCount() int {
	return ps.Count
}

func createPolicyScript(keyHashes []string, atLeastSignersCount int) ([]byte, error) {
	type policyScript struct {
		Type     string       `json:"type"`
		Required int          `json:"required"`
		Scripts  []keyHashSig `json:"scripts"`
	}

	p := policyScript{
		Type:     "atLeast",
		Required: atLeastSignersCount,
	}

	for _, keyHash := range keyHashes {
		p.Scripts = append(p.Scripts, keyHashSig{
			Type:    "sig",
			KeyHash: keyHash,
		})
	}

	return json.MarshalIndent(p, "", "  ")
}

func getPolicyID(policyScriptFilePath string) (string, error) {
	response, err := runCommand(resolveCardanoCliBinary(), []string{
		"transaction", "policyid", "--script-file", policyScriptFilePath,
	})
	if err != nil {
		return "", err
	}

	return strings.Trim(response, "\n"), nil
}
