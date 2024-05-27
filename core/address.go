package core

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/igorcrevar/cardano-wallet-tx/core/bech32"
)

// code mainly from https://github.com/fivebinaries/go-cardano-serialization/blob/master/address/address.go
var (
	ErrUnsupportedAddress = errors.New("invalid/unsupported address type")
	ErrInvalidData        = errors.New("invalid data")
)

func NewAddress(raw string) (addr CardanoAddress, err error) {
	var data []byte

	if strings.HasPrefix(raw, "addr") || strings.HasPrefix(raw, "stake") {
		_, data, err = bech32.DecodeToBase256(raw)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, ErrUnsupportedAddress // byron not supported data = base58.Decode(raw)
	}

	return NewAddressFromBytes(data)
}

func NewAddressFromBytes(data []byte) (addr CardanoAddress, err error) {
	if len(data) == 0 {
		return nil, ErrInvalidData
	}

	header := data[0]
	netID := CardanoNetworkType(header & 0x0F)

	switch (header & 0xF0) >> 4 {
	// 1000: byron address
	case 0b1000:
		return nil, ErrUnsupportedAddress

	// 0000: base address: keyhash28,keyhash28
	// 0001: base address: scripthash28,keyhash28
	// 0010: base address: keyhash28,scripthash28
	// 0011: base address: scripthash28,scripthash28
	case 0b0000, 0b0001, 0b0010, 0b0011:
		if len(data) != 1+KeyHashSize*2 {
			return nil, fmt.Errorf("%w: expect %d got %d", ErrInvalidData, 1+KeyHashSize*2, len(data))
		}

		payment, err := NewStakeCredential(data[1:], header&(1<<4) > 0)
		if err != nil {
			return nil, err
		}

		stake, err := NewStakeCredential(data[1+KeyHashSize:], header&(1<<5) > 0)
		if err != nil {
			return nil, err
		}

		return &BaseAddress{
			Network: netID,
			Payment: payment,
			Stake:   stake,
		}, nil

	// 0100: pointer address: keyhash28, 3 variable length uint
	// 0101: pointer address: scripthash28, 3 variable length uint
	case 0b0100, 0b0101:
		if len(data) < 1+KeyHashSize+1+1+1 { // header + payment + at least one byte for all three pointer parts
			return nil, fmt.Errorf("%w: expect at least %d got %d", ErrInvalidData, 1+KeyHashSize+1+1+1, len(data))
		}

		payment, err := NewStakeCredential(data[1:], header&(1<<4) > 0)
		if err != nil {
			return nil, err
		}

		pointer, err := GetStakePointer(data[29:])
		if err != nil {
			return nil, err
		}

		return &PointerAddress{
			Network:      netID,
			Payment:      payment,
			StakePointer: pointer,
		}, nil

	// 0110: enterprise address: keyhash28
	// 0111: enterprise address: scripthash28
	case 0b0110, 0b0111:
		if len(data) != KeyHashSize+1 {
			return nil, fmt.Errorf("%w: expect %d got %d", ErrInvalidData, 1+KeyHashSize, len(data))
		}

		payment, err := NewStakeCredential(data[1:], header&(1<<4) > 0)
		if err != nil {
			return nil, err
		}

		return &EnterpriseAddress{
			Network: netID,
			Payment: payment,
		}, nil

	case 0b1110, 0b1111:
		if len(data) != KeyHashSize+1 {
			return nil, fmt.Errorf("%w: expect %d got %d", ErrInvalidData, 1+KeyHashSize, len(data))
		}

		stake, err := NewStakeCredential(data[1:], header&(1<<4) > 0)
		if err != nil {
			return nil, err
		}

		return &RewardAddress{
			Network: netID,
			Stake:   stake,
		}, nil

	default:
		return nil, ErrUnsupportedAddress
	}
}

func NewBaseAddress(
	network CardanoNetworkType, paymentVerificationKey, stakeVerificationKey []byte,
) (*BaseAddress, error) {
	paymentHash, err := GetKeyHashBytes(paymentVerificationKey)
	if err != nil {
		return nil, err
	}

	stakeHash, err := GetKeyHashBytes(stakeVerificationKey)
	if err != nil {
		return nil, err
	}

	payment, err := NewStakeCredential(paymentHash, false)
	if err != nil {
		return nil, err
	}

	stake, err := NewStakeCredential(stakeHash, false)
	if err != nil {
		return nil, err
	}

	return &BaseAddress{
		Network: network,
		Payment: payment,
		Stake:   stake,
	}, nil
}

func NewEnterpriseAddress(
	network CardanoNetworkType, verificationKey []byte,
) (*EnterpriseAddress, error) {
	paymentHash, err := GetKeyHashBytes(verificationKey)
	if err != nil {
		return nil, err
	}

	payment, err := NewStakeCredential(paymentHash, false)
	if err != nil {
		return nil, err
	}

	return &EnterpriseAddress{
		Network: network,
		Payment: payment,
	}, nil
}

func NewEnterpriseAddressFromPolicyScript(
	network CardanoNetworkType, ps *PolicyScript,
) (*EnterpriseAddress, error) {
	policyID, err := ps.GetPolicyID()
	if err != nil {
		return nil, err
	}

	scriptKeyHashBytes, err := hex.DecodeString(policyID)
	if err != nil {
		return nil, err
	}

	payment, err := NewStakeCredential(scriptKeyHashBytes, true)
	if err != nil {
		return nil, err
	}

	return &EnterpriseAddress{
		Network: network,
		Payment: payment,
	}, nil
}

func NewRewardAddress(
	network CardanoNetworkType, verificationKey []byte,
) (*RewardAddress, error) {
	stakeHash, err := GetKeyHashBytes(verificationKey)
	if err != nil {
		return nil, err
	}

	stake, err := NewStakeCredential(stakeHash, false)
	if err != nil {
		return nil, err
	}

	return &RewardAddress{
		Network: network,
		Stake:   stake,
	}, nil
}
