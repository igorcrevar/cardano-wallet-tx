package core

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWalletExtended(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-cardano-wallet")
	require.NoError(t, err)

	defer func() {
		os.RemoveAll(testDir)
		os.Remove(testDir)
	}()

	type walletContainer struct {
		Wallet      *Wallet `json:"wallet"`
		WalletStake *Wallet `json:"walletStake"`
	}

	wallet, err := GenerateWallet(false)
	require.NoError(t, err)

	walletStake, err := GenerateWallet(true)
	require.NoError(t, err)

	wc := &walletContainer{
		Wallet:      wallet,
		WalletStake: walletStake,
	}

	bytes, err := json.Marshal(wc)
	require.NoError(t, err)

	var wc2 *walletContainer

	require.NoError(t, json.Unmarshal(bytes, &wc2))

	require.Equal(t, wc, wc2)

	const (
		msg1 = "message number 1"
		msg2 = "some other message"
		msg3 = "third one"
	)

	signature, err := SignMessage(wc.Wallet.SigningKey, wc.Wallet.VerificationKey, []byte(msg1))
	require.NoError(t, err)

	signature2, err := SignMessage(wc.WalletStake.StakeSigningKey, wc.WalletStake.StakeVerificationKey, []byte(msg2))
	require.NoError(t, err)

	signature3, err := SignMessage(wc.WalletStake.SigningKey, wc.WalletStake.VerificationKey, []byte(msg3))
	require.NoError(t, err)

	require.NoError(t, VerifyMessage([]byte(msg1), wc.Wallet.VerificationKey, signature))
	require.NoError(t, VerifyMessage([]byte(msg2), wc.WalletStake.StakeVerificationKey, signature2))
	require.NoError(t, VerifyMessage([]byte(msg3), wc.WalletStake.VerificationKey, signature3))
	require.Error(t, VerifyMessage([]byte(msg3), wc.Wallet.VerificationKey, signature))
	require.Error(t, VerifyMessage([]byte(msg1), wc.WalletStake.StakeVerificationKey, signature2))
	require.Error(t, VerifyMessage([]byte(msg2), wc.WalletStake.VerificationKey, signature3))
}
