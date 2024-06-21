package core

import (
	"encoding/hex"

	"github.com/igorcrevar/cardano-wallet-tx/core/bech32"
)

// code mainly from https://github.com/fivebinaries/go-cardano-serialization/blob/master/address/address.go
type StakeCredentialType byte

const (
	KeyStakeCredentialType StakeCredentialType = iota
	ScriptStakeCredentialType
	EmptyStakeCredentialType

	KeyHashSize int = 28
)

type CardanoAddress interface {
	GetPayment() StakeCredential
	GetStake() StakeCredential
	GetNetwork() CardanoNetworkType
	GetStakePointer() StakePointer
	Bytes() []byte
	String() string
}

type StakeCredential struct {
	Kind    StakeCredentialType `cbor:"0,keyasint,omitempty"`
	Payload [KeyHashSize]byte   `cbor:"1,keyasint,omitempty"`
}

func (sc StakeCredential) String() string {
	return hex.EncodeToString(sc.Payload[:])
}

func NewStakeCredential(data []byte, isScript bool) (StakeCredential, error) {
	if len(data) < KeyHashSize {
		return StakeCredential{}, ErrInvalidData
	}

	var hashBytes [KeyHashSize]byte

	copy(hashBytes[:], data[:KeyHashSize])

	if isScript {
		return StakeCredential{
			Kind:    ScriptStakeCredentialType,
			Payload: hashBytes,
		}, nil
	}

	return StakeCredential{
		Kind:    KeyStakeCredentialType,
		Payload: hashBytes,
	}, nil
}

// BaseAddress contains information of the base address.
// A base address directly specifies the staking key that should control the stake for that address
// but can be used for transactions without registering the staking key in advance.
type BaseAddress struct {
	Network CardanoNetworkType
	Payment StakeCredential
	Stake   StakeCredential
	Extra   []byte
}

func (a BaseAddress) GetPayment() StakeCredential {
	return a.Payment
}

func (a BaseAddress) GetStake() StakeCredential {
	return a.Stake
}

func (a BaseAddress) GetNetwork() CardanoNetworkType {
	return a.Network
}

func (a BaseAddress) GetStakePointer() StakePointer {
	return StakePointer{}
}

func (a BaseAddress) Bytes() []byte {
	bytes := make([]byte, KeyHashSize*2+1+len(a.Extra))
	bytes[0] = (byte(a.Payment.Kind) << 4) | (byte(a.Stake.Kind) << 5) | (byte(a.Network) & 0xf)

	copy(bytes[1:KeyHashSize+1], a.Payment.Payload[:])
	copy(bytes[KeyHashSize+1:], a.Stake.Payload[:])
	copy(bytes[KeyHashSize*2+1:], a.Extra)

	return bytes
}

func (a BaseAddress) String() string {
	str, _ := bech32.EncodeFromBase256(a.Network.GetPrefix(), a.Bytes())

	return str
}

// EnterpriseAddress contains content for enterprise addresses.
// Enterprise addresses carry no stake rights, so using these addresses
// means that you are opting out of participation in the proof-of-stake protocol.
type EnterpriseAddress struct {
	Network CardanoNetworkType
	Payment StakeCredential
}

func (a EnterpriseAddress) GetPayment() StakeCredential {
	return a.Payment
}

func (a EnterpriseAddress) GetStake() StakeCredential {
	return StakeCredential{Kind: EmptyStakeCredentialType}
}

func (a EnterpriseAddress) GetNetwork() CardanoNetworkType {
	return a.Network
}

func (a EnterpriseAddress) Bytes() []byte {
	bytes := [KeyHashSize + 1]byte{}
	bytes[0] = 0b01100000 | (byte(a.Payment.Kind) << 4) | (byte(a.Network) & 0xf)

	copy(bytes[1:], a.Payment.Payload[:])

	return bytes[:]
}

func (a EnterpriseAddress) GetStakePointer() StakePointer {
	return StakePointer{}
}

func (a EnterpriseAddress) String() string {
	str, _ := bech32.EncodeFromBase256(a.Network.GetPrefix(), a.Bytes())

	return str
}

// RewardAddress contains content of the reward/staking address.
// Reward account addresses are used to distribute rewards for participating
// in the proof-of-stake protocol (either directly or via delegation).
type RewardAddress struct {
	Network CardanoNetworkType
	Stake   StakeCredential
}

func (a RewardAddress) GetPayment() StakeCredential {
	return StakeCredential{Kind: EmptyStakeCredentialType}
}

func (a RewardAddress) GetStake() StakeCredential {
	return a.Stake
}

func (a RewardAddress) GetNetwork() CardanoNetworkType {
	return a.Network
}

func (a RewardAddress) GetStakePointer() StakePointer {
	return StakePointer{}
}

func (a RewardAddress) Bytes() []byte {
	data := [KeyHashSize + 1]byte{}
	data[0] = 0b1110_0000 | (byte(a.Stake.Kind) << 4) | (byte(a.Network) & 0xf)

	copy(data[1:], a.Stake.Payload[:])

	return data[:]
}

func (a RewardAddress) String() string {
	str, _ := bech32.EncodeFromBase256(a.Network.GetStakePrefix(), a.Bytes())

	return str
}

type StakePointer struct {
	Slot      uint64
	TxIndex   uint64
	CertIndex uint64
}

// A pointer address indirectly specifies the staking key that should control the stake for the address.
type PointerAddress struct {
	Network      CardanoNetworkType
	Payment      StakeCredential
	StakePointer StakePointer
}

func (a PointerAddress) GetPayment() StakeCredential {
	return a.Payment
}

func (a PointerAddress) GetStake() StakeCredential {
	return StakeCredential{Kind: EmptyStakeCredentialType}
}

func (a PointerAddress) GetNetwork() CardanoNetworkType {
	return a.Network
}

func (a PointerAddress) GetStakePointer() StakePointer {
	return a.StakePointer
}

func (a PointerAddress) Bytes() (bytes []byte) {
	variableEncode := func(num uint64) []byte {
		var output []byte

		output = append(output, byte(num)&0x7F)
		num /= 128

		for num > 0 {
			output = append(output, byte(num)&0x7F|0x80)
			num /= 128
		}

		for i, j := 0, len(output)-1; i < j; i, j = i+1, j-1 {
			output[i], output[j] = output[j], output[i]
		}

		return output
	}

	buf := make([]byte, 0, KeyHashSize+1+3)

	header := 0b0100_0000 | (byte(a.Payment.Kind) << 4) | (byte(a.Network) & 0xF)
	buf = append(buf, header)
	buf = append(buf, a.Payment.Payload[:]...)
	buf = append(buf, variableEncode(a.StakePointer.Slot)...)
	buf = append(buf, variableEncode(a.StakePointer.TxIndex)...)

	return append(buf, variableEncode(a.StakePointer.CertIndex)...)
}

func (a PointerAddress) String() string {
	str, _ := bech32.EncodeFromBase256(a.Network.GetPrefix(), a.Bytes())

	return str
}

func GetStakePointer(raw []byte) (StakePointer, error) {
	readOne := func(raw []byte) (result uint64, bytesReadCnt int, err error) {
		for _, rbyte := range raw {
			result = (result << 7) | uint64(rbyte&0x7F)
			bytesReadCnt++

			if (rbyte & 0x80) == 0 {
				return result, bytesReadCnt, nil
			}
		}

		return 0, 0, ErrInvalidData
	}

	slot, bytesReadCnt, err := readOne(raw)
	if err != nil {
		return StakePointer{}, err
	}

	txIndex, bytesReadCnt2, err := readOne(raw[bytesReadCnt:])
	if err != nil {
		return StakePointer{}, err
	}

	certIndex, bytesReadCnt3, err := readOne(raw[bytesReadCnt+bytesReadCnt2:])
	if err != nil {
		return StakePointer{}, err
	}

	if bytesReadCnt+bytesReadCnt2+bytesReadCnt3 != len(raw) {
		return StakePointer{}, ErrInvalidData
	}

	return StakePointer{
		Slot:      slot,
		TxIndex:   txIndex,
		CertIndex: certIndex,
	}, nil
}
