package core

import (
	"encoding/json"
	"os"
	"path"
)

type MultisigAddress struct {
	keyHashes           []string
	directory           string
	testNetMagic        uint
	atLeastSignersCount int
	address             string
}

func NewMultiSigAddress(directory string, keyHashes []string, testNetMagic uint, atLeastSignersCount int) *MultisigAddress {
	return &MultisigAddress{
		keyHashes:           keyHashes,
		directory:           directory,
		testNetMagic:        testNetMagic,
		atLeastSignersCount: atLeastSignersCount,
	}
}

func (ma MultisigAddress) GetPolicyScript() ([]byte, error) {
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
		Required: ma.atLeastSignersCount,
	}

	for _, keyHash := range ma.keyHashes {
		p.Scripts = append(p.Scripts, keyHashSig{
			Type:    "sig",
			KeyHash: keyHash,
		})
	}

	return json.MarshalIndent(p, "", "  ")
}

func (ma *MultisigAddress) Load() error {
	bytes, err := os.ReadFile(path.Join(ma.directory, addressFile))
	if err != nil {
		return err
	}

	ma.address = string(bytes)

	return nil
}

func (ma *MultisigAddress) Create(forceCreate bool) error {
	addressFilePath := path.Join(ma.directory, addressFile)
	policyFilePath := path.Join(ma.directory, "policy.json")

	if !forceCreate && isFileOrDirExists(addressFilePath) {
		return nil
	}

	if err := createDirectoryIfNotExists(ma.directory); err != nil {
		return err
	}

	content, err := ma.GetPolicyScript()
	if err != nil {
		return err
	}

	if err := os.WriteFile(policyFilePath, content, 0755); err != nil {
		return err
	}

	_, err = runCommand(resolveCardanoCliBinary(), append([]string{
		"address", "build",
		"--payment-script-file", policyFilePath,
		"--out-file", addressFilePath,
	}, getTestNetMagicArgs(ma.testNetMagic)...))

	return err
}

func (ma MultisigAddress) GetAddress() string {
	return ma.address
}

func (ma MultisigAddress) GetCount() int {
	return len(ma.keyHashes)
}
