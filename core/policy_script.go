package core

import (
	"encoding/hex"
	"encoding/json"
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

func (ps PolicyScript) GetCount() (cnt int) {
	switch ps.Type {
	case PolicyScriptSigType:
		cnt = 1
	case "any":
		for _, x := range ps.Scripts {
			if subCnt := x.GetCount(); cnt < subCnt {
				cnt = subCnt
			}
		}
	case "all", PolicyScriptAtLeastType:
		for _, x := range ps.Scripts {
			cnt += x.GetCount()
		}
	}

	return cnt
}

// GetAddress returns address for this policy script
func NewPolicyScriptAddress(
	networkID CardanoNetworkType, policyID string, policyIDStake ...string,
) (*CardanoAddress, error) {
	policyIDBytes, err := hex.DecodeString(policyID)
	if err != nil {
		return nil, err
	}

	if len(policyIDStake) == 0 {
		return CardanoAddressInfo{
			AddressType: EnterpriseAddress,
			Network:     networkID,
			Payment: &CardanoAddressPayload{
				Payload:  [KeyHashSize]byte(policyIDBytes),
				IsScript: true,
			},
		}.ToCardanoAddress()
	}

	policyIDStakeBytes, err := hex.DecodeString(policyIDStake[0])
	if err != nil {
		return nil, err
	}

	return CardanoAddressInfo{
		AddressType: BaseAddress,
		Network:     networkID,
		Payment: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(policyIDBytes),
			IsScript: true,
		},
		Stake: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(policyIDStakeBytes),
			IsScript: true,
		},
	}.ToCardanoAddress()
}
