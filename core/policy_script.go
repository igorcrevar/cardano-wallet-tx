package core

import (
	"encoding/json"
	"os"
	"path"
	"strings"
)

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
	if err := os.WriteFile(policyScriptFilePath, ps.PolicyScript, 0755); err != nil {
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

func (ps PolicyScript) GetPolicyScript() []byte {
	return ps.PolicyScript
}

func (ps PolicyScript) GetCount() int {
	return ps.Count
}

func createPolicyScript(keyHashes []string, atLeastSignersCount int) ([]byte, error) {
	type keyHashSig struct {
		Type    string `json:"type"`
		KeyHash string `json:"keyHash"`
	}

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
