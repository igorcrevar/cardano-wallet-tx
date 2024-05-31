package core

import (
	"encoding/hex"
	"encoding/json"

	"github.com/fxamacker/cbor/v2"
)

const (
	verificationKeyName      = "vkey"
	signingKeyName           = "skey"
	stakeVerificationKeyName = "stake.vkey"
	stakeSigningKeyName      = "stake.skey"
)

func GenerateWallet(isStake bool) (*Wallet, error) {
	signingKey, verificationKey, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	if !isStake {
		return NewWallet(verificationKey, signingKey), nil
	}

	stakeSigningKey, stakeVerificationKey, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	return NewStakeWallet(verificationKey, signingKey, stakeVerificationKey, stakeSigningKey), nil
}

func (w Wallet) MarshalJSON() ([]byte, error) {
	keyBytesToString := func(bytes []byte) (string, error) {
		cborBytes, err := cbor.Marshal(bytes)
		if err != nil {
			return "", err
		}

		return hex.EncodeToString(cborBytes), nil
	}

	verificationKeyStr, err := keyBytesToString(w.VerificationKey)
	if err != nil {
		return nil, err
	}

	signingKeyStr, err := keyBytesToString(w.SigningKey)
	if err != nil {
		return nil, err
	}

	result := map[string]string{
		verificationKeyName: verificationKeyStr,
		signingKeyName:      signingKeyStr,
	}

	if len(w.StakeVerificationKey) > 0 && len(w.StakeSigningKey) > 0 {
		stakeVerificationKeyStr, err := keyBytesToString(w.StakeVerificationKey)
		if err != nil {
			return nil, err
		}

		stakeSigningKeyStr, err := keyBytesToString(w.StakeSigningKey)
		if err != nil {
			return nil, err
		}

		result[stakeVerificationKeyName] = stakeVerificationKeyStr
		result[stakeSigningKeyName] = stakeSigningKeyStr
	}

	return json.Marshal(result)
}

func (w *Wallet) UnmarshalJSON(data []byte) error {
	keyBytesFromString := func(str string) ([]byte, error) {
		bytes, err := hex.DecodeString(str)
		if err != nil {
			return nil, err
		}

		var result []byte

		if err := cbor.Unmarshal(bytes, &result); err != nil {
			return nil, err
		}

		return result, nil
	}

	mp := map[string]string{}

	if err := json.Unmarshal(data, &mp); err != nil {
		return err
	}

	vkey, err := keyBytesFromString(mp[verificationKeyName])
	if err != nil {
		return err
	}

	skey, err := keyBytesFromString(mp[signingKeyName])
	if err != nil {
		return err
	}

	if len(mp[stakeVerificationKeyName]) > 0 && len(mp[stakeSigningKeyName]) > 0 {
		stakeVerificationKeyBytes, err := keyBytesFromString(mp[stakeVerificationKeyName])
		if err != nil {
			return err
		}

		stakeSigningKeyBytes, err := keyBytesFromString(mp[stakeSigningKeyName])
		if err != nil {
			return err
		}

		w.StakeVerificationKey = stakeVerificationKeyBytes
		w.StakeSigningKey = stakeSigningKeyBytes
	}

	w.VerificationKey = vkey
	w.SigningKey = skey

	return nil
}
