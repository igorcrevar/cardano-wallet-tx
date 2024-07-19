package core

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

const (
	PolicyScriptAtLeastType = "atLeast"
	PolicyScriptSigType     = "sig"
)

type PolicyScript struct {
	Type string `json:"type"`

	Required int            `json:"required,omitempty"`
	Scripts  []PolicyScript `json:"scripts,omitempty"`

	KeyHash string `json:"keyHash,omitempty"`
	Slot    uint64 `json:"slot,omitempty"`
}

func NewPolicyScript(keyHashes []string, atLeastSignersCount int) *PolicyScript {
	scripts := make([]PolicyScript, len(keyHashes))
	for i, keyHash := range keyHashes {
		scripts[i] = PolicyScript{
			Type:    PolicyScriptSigType,
			KeyHash: keyHash,
		}
	}

	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].KeyHash < scripts[j].KeyHash
	})

	return &PolicyScript{
		Type:     PolicyScriptAtLeastType,
		Required: atLeastSignersCount,
		Scripts:  scripts,
	}
}

func (ps PolicyScript) GetPolicyScriptJSON() ([]byte, error) {
	return json.MarshalIndent(ps, "", "  ")
}

func (ps PolicyScript) GetCount() int {
	cnt := 0

	for _, x := range ps.Scripts {
		if x.Type == PolicyScriptSigType {
			cnt++
		}
	}

	return cnt
}

// GetAddress returns address for this policy script
func NewPolicyScriptAddress(
	networkID CardanoNetworkType, policyID string, policyIDStake ...string,
) (CardanoAddress, error) {
	getStakeCredential := func(pid string) (StakeCredential, error) {
		keyHash, err := hex.DecodeString(pid)
		if err != nil {
			return StakeCredential{}, fmt.Errorf("failed to decode policy id: %w", err)
		}

		return NewStakeCredential(keyHash, true)
	}

	payment, err := getStakeCredential(policyID)
	if err != nil {
		return nil, err
	}

	if len(policyIDStake) == 0 {
		return &EnterpriseAddress{
			Network: networkID,
			Payment: payment,
		}, nil
	}

	stake, err := getStakeCredential(policyIDStake[0])
	if err != nil {
		return nil, err
	}

	return &BaseAddress{
		Network: networkID,
		Payment: payment,
		Stake:   stake,
	}, nil
}
