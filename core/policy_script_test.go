package core

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPolicyScript(t *testing.T) {
	var (
		err       error
		wallets   = [6]IWallet{}
		keyHashes = [6]string{}
	)

	for i := range wallets {
		wallets[i], err = GenerateWallet(true)
		require.NoError(t, err)

		keyHashes[i], err = GetKeyHash(wallets[i].GetVerificationKey())
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
