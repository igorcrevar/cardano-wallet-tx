package core

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPolicyScript(t *testing.T) {
	t.Parallel()

	var (
		err       error
		wallets   = [6]*Wallet{}
		keyHashes = [6]string{}
	)

	for i := range wallets {
		wallets[i], err = GenerateWallet(true)
		require.NoError(t, err)

		keyHashes[i], err = GetKeyHash(wallets[i].VerificationKey)
		require.NoError(t, err)
	}

	cliUtils := NewCliUtils(ResolveCardanoCliBinary(MainNetNetwork))

	ps := NewPolicyScript(keyHashes[:4], 4)
	psStake := NewPolicyScript(keyHashes[4:], 1)
	psDifferentOrder := NewPolicyScript(append([]string{}, keyHashes[1], keyHashes[0], keyHashes[3], keyHashes[2]), 4)

	policyID, err := cliUtils.GetPolicyID(ps)
	require.NoError(t, err)

	policyIDDifferentOrder, err := cliUtils.GetPolicyID(psDifferentOrder)
	require.NoError(t, err)

	require.Equal(t, policyID, policyIDDifferentOrder)

	policyIDStake, err := cliUtils.GetPolicyID(psStake)
	require.NoError(t, err)

	cliAddrStake, err := cliUtils.GetPolicyScriptAddress(MainNetProtocolMagic, ps, psStake)
	require.NoError(t, err)

	addrStake, err := NewPolicyScriptAddress(MainNetNetwork, policyID, policyIDStake)
	require.NoError(t, err)

	require.Equal(t, cliAddrStake, addrStake.String())

	cliAddr, err := cliUtils.GetPolicyScriptAddress(MainNetProtocolMagic, ps)
	require.NoError(t, err)

	addr, err := NewPolicyScriptAddress(MainNetNetwork, policyID)
	require.NoError(t, err)

	require.Equal(t, cliAddr, addr.String())
}

func TestPolicyScript_SpecificKeysAllPermutations(t *testing.T) {
	t.Parallel()

	var (
		allPublicKeys = []string{
			"5d767d06a9426bafd31eae25122b586fb6cac32efcee60c94bf8f43faddb8f5b",
			"f5c69c8a0bb63016068d4683dab19af0b0833158ed7a5ed91bd328d0939f3173",
			"2983addc84a6032feeb8870a9f74308e4a5446779bf44a8a790be3fb266e1abd",
			"001c4e9e6493675a3f380d1749d203a5aabb92b217b28b918a1b0aea6b8981b0",
		}
		permute func(int, []int, []bool, func([]string))
	)

	// permute generates all permutations of the input slice and appends them to the result.
	permute = func(n int, current []int, used []bool, execute func(keys []string)) {
		if len(current) == n {
			keys := make([]string, n)
			for i, x := range current {
				keys[i] = allPublicKeys[x]
			}

			execute(keys)

			return
		}

		for i := 0; i < n; i++ {
			if used[i] {
				continue
			}
			// Mark the element as used
			used[i] = true
			current = append(current, i)

			// Recursively generate permutations
			permute(n, current, used, execute)

			// Unmark the element and remove it from the current permutation
			used[i] = false
			current = current[:len(current)-1]
		}
	}

	permute(len(allPublicKeys), []int{}, make([]bool, len(allPublicKeys)), func(keys []string) {
		cliUtils := NewCliUtils(ResolveCardanoCliBinary(MainNetNetwork))
		hashes := make([]string, len(keys))

		for i, k := range keys {
			bytes, err := hex.DecodeString(k)
			require.NoError(t, err)

			h, err := GetKeyHash(bytes)
			require.NoError(t, err)

			hashes[i] = h
		}

		ps := NewPolicyScript(hashes, (len(hashes)*2+2)/3)

		policyID, err := cliUtils.GetPolicyID(ps)
		require.NoError(t, err)

		cliAddr, err := cliUtils.GetPolicyScriptAddress(TestNetProtocolMagic, ps)
		require.NoError(t, err)

		addr, err := NewPolicyScriptAddress(TestNetNetwork, policyID)
		require.NoError(t, err)

		require.Equal(t, cliAddr, addr.String())
	})
}
