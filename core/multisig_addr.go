package core

import (
	"encoding/json"
	"os"
	"path"
	"strings"
)

type MultisigAddress struct {
	policyScript []byte
	address      string
	count        int
}

func NewMultiSigAddress(address string, policyScript []byte, count int) *MultisigAddress {
	return &MultisigAddress{
		address:      address,
		policyScript: policyScript,
		count:        count,
	}
}

func (ma MultisigAddress) GetAddress() string {
	return ma.address
}

func (ma MultisigAddress) GetCount() int {
	return ma.count
}

func (ma MultisigAddress) GetPolicyScript() []byte {
	return ma.policyScript
}

type PolicyScript struct {
	policyScript []byte
	count        int
}

func NewPolicyScript(keyHashes []string, atLeastSignersCount int) (*PolicyScript, error) {
	policyScript, err := createPolicyScript(keyHashes, atLeastSignersCount)
	if err != nil {
		return nil, err
	}

	return &PolicyScript{
		policyScript: policyScript,
		count:        len(keyHashes),
	}, nil
}

func (ps PolicyScript) CreateMultiSigAddress(testNetMagic uint) (*MultisigAddress, error) {
	baseDirectory, err := os.MkdirTemp("", "cardano-multisig-addr")
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(baseDirectory)

	policyScriptFilePath := path.Join(baseDirectory, "policy-script.json")
	if err := os.WriteFile(policyScriptFilePath, ps.policyScript, 0755); err != nil {
		return nil, err
	}

	response, err := runCommand(resolveCardanoCliBinary(), append([]string{
		"address", "build",
		"--payment-script-file", policyScriptFilePath,
	}, getTestNetMagicArgs(testNetMagic)...))
	if err != nil {
		return nil, err
	}

	return NewMultiSigAddress(strings.Trim(response, "\n"), ps.policyScript, ps.count), nil
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
