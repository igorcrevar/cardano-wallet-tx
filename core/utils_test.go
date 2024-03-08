package core

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidCardanoAddress(t *testing.T) {
	addresses := []string{
		"addr_test1vp4l5ka8jaqe32kygjemg6g745lxrn0mem7fxvuarrazmesdyntms",
		"addr_test1wpkr0wd9ggr3zmfs7a2pg845jld95nvjjzg4mnr0ewqmzmsf689u8",
		"addr1qy4h5rr93jh4x83448tk8y3whpddtyfq7g6pwr4tr3fuwcet0gxxtr902v0rt2whvwfzawz66kgjpu35zu82k8znca3sk9t664",
		"addr1qy4h5rr93jh4x83448tk8y3whpddtyfq7g6pwr4tr3fuwcet0gxxtr902v0rt2whvwfzawz66kgjpu35zu82k8znca3sk9t664",
		"stake1uy4h5rr93jh4x83448tk8y3whpddtyfq7g6pwr4tr3fuwccdjgq9n",
		"addr1wpkr0wd9ggr3zmfs7a2pg845jld95nvjjzg4mnr0ewqmzmsf689u8",
		"addr_test1wpkr0wd9ggr3zmfs7a2pg845jld95nvjjzg4mnr0ewqmzmef689u8",
		"addr1qy4h5rr93jh4x83448tk8y3whpddtyfq7g6pwr4tr3fuwcet0gxxtr902v0rt2whvwfzawz66kgjpu35zu82k8znca3sket664",
		"addr1qy4h5rr93jh4x83448tk8y3whpddtyfq7g6pwr4tr3fuwcet0gxxtr902v0rt2whvwfzawz66kgjpu35zu82k8znca3sk9t664",
		"stake1uy4h5rr93jh4x83448tk8y3whpddtyfq7g6pwr4tr3fuwccdjgq9n",
		"addr_test1wpkr0wd9ggr3zmfs7a2pg845jld95nvjjzg4mnr0ewqmzmsf689u8",
	}
	testNetwork := []AddressType{
		AddressTypeTestBase,
		AddressTypeAnyTest,
		AddressTypeBase,
		AddressTypeAnyMainnet,
		AddressTypeStake,
		AddressTypeBase,
		AddressTypeTestBase,
		AddressTypeBase,
		AddressTypeTestBase,
		AddressTypeBase,
		AddressTypeTestStake,
	}
	results := []bool{
		true,
		true,
		true,
		true,
		true,
		false,
		false,
		false,
		false,
		false,
		false,
	}

	for i, addr := range addresses {
		ai := GetAddressInfo(addr, testNetwork[i])
		assert.Equal(t, results[i], ai.IsValid)
	}
}

func TestVerifyWitness(t *testing.T) {
	var (
		txHash         = "7e8b59e41d2ba71888272a14cff401268fa01dceb19014f5dda7763334b8f221"
		signingKey, _  = hex.DecodeString("1217236ac24d8ac12684b308cf9468f68ef5283096896dc1c5c3caf8351e2847")                                                                                                                                           // nolint
		witnessCbor, _ = hex.DecodeString("8258203e9d3a6f792c9820ab4423e41256e4b6e2ae1f456318f9d936fc70e0eafdc76f58402992d7fbc6fb155b7cc83223c80bf9b0ddbfe24ff260600897a06e8050f6596a76defeea6a86048605f8f7c27ef53da318aa02838532ea1876aac876b2491a01") // nolint
		txHashBytes, _ = hex.DecodeString(txHash)                                                                                                                                                                                                       // noline
	)

	verificationKey := GetVerificationKeyFromSigningKey(signingKey)

	signature, vKeyWitness, err := TxWitnessRaw(witnessCbor).GetSignatureAndVKey()
	require.NoError(t, err)

	require.NoError(t, err)
	assert.Equal(t, verificationKey, vKeyWitness)

	signedMessageCbor, err := SignMessage(signingKey, verificationKey, txHashBytes)
	require.NoError(t, err)
	assert.Equal(t, signature, signedMessageCbor)

	dummySignature, err := SignMessage(signingKey, verificationKey, append([]byte{255}, txHash[1:]...))
	require.NoError(t, err)

	dummyWitness, err := cbor.Marshal([][]byte{verificationKey, dummySignature})
	require.NoError(t, err)

	assert.NoError(t, VerifyWitness(txHash, witnessCbor))
	assert.ErrorIs(t, VerifyWitness(strings.Replace(txHash, "7e", "7f", 1), witnessCbor), ErrInvalidWitness)
	assert.ErrorIs(t, VerifyWitness(txHash, dummyWitness), ErrInvalidWitness)
}
