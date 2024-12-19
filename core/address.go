package core

import (
	"errors"

	"github.com/akamensky/base58"
	"github.com/igorcrevar/go-cardano-tx/core/bech32"
)

var (
	ErrInvalidAddressData = errors.New("invalid address data")
)

type CardanoAddressType byte

const (
	UnsupportedAddress CardanoAddressType = 0
	ByronAddress       CardanoAddressType = 0b1000
	BaseAddress        CardanoAddressType = 1 // 0b0000, 0b0001, 0b0010, 0b0011
	PointerAddress     CardanoAddressType = 2 // 0b0100, 0b0101
	EnterpriseAddress  CardanoAddressType = 3 // 0b0110, 0b0111
	RewardAddress      CardanoAddressType = 4 // 0b1110, 0b1111
)

func GetAddressTypeFromHeader(header byte) CardanoAddressType {
	switch (header & 0xF0) >> 4 {
	case byte(ByronAddress):
		return ByronAddress
	case 0b0000, 0b0001, 0b0010, 0b0011:
		return BaseAddress
	case 0b0100, 0b0101:
		return PointerAddress
	case 0b0110, 0b0111:
		return EnterpriseAddress
	case 0b1110, 0b1111:
		return RewardAddress
	default:
		return UnsupportedAddress
	}
}

type CardanoAddress struct {
	raw []byte

	addressParser     cardanoAddressParser
	cachedAddressInfo CardanoAddressInfo
	cachedStr         string
}

func NewCardanoAddress(raw []byte) (*CardanoAddress, error) {
	if len(raw) == 0 {
		return nil, ErrInvalidAddressData
	}

	addressParser, err := getAddressParser(GetAddressTypeFromHeader(raw[0]))
	if err != nil {
		return nil, err
	}

	if err := addressParser.IsValid(raw); err != nil {
		return nil, err
	}

	return &CardanoAddress{
		raw:           raw,
		addressParser: addressParser,
	}, nil
}

func NewCardanoAddressFromString(raw string) (*CardanoAddress, error) {
	if !IsAddressWithValidPrefix(raw) {
		data, err := base58.Decode(raw)
		if err != nil {
			return nil, errors.Join(ErrInvalidAddressData, err)
		} else if len(data) == 0 || GetAddressTypeFromHeader(data[0]) != ByronAddress {
			return nil, ErrInvalidAddressData
		}

		return NewCardanoAddress(data)
	}

	_, data, err := bech32.DecodeToBase256(raw)
	if err != nil {
		return nil, err
	}

	addr, err := NewCardanoAddress(data)
	if err != nil {
		return nil, err
	}

	addr.cachedStr = raw // string representation should not be recalculated

	return addr, nil
}

func (a *CardanoAddress) GetInfo() CardanoAddressInfo {
	if a.cachedAddressInfo.AddressType == UnsupportedAddress {
		a.cachedAddressInfo = a.addressParser.ToCardanoAddressInfo(a.raw)
	}

	return a.cachedAddressInfo
}

func (a *CardanoAddress) GetBytes() []byte {
	return a.raw
}

func (a *CardanoAddress) String() string {
	if a.cachedStr == "" {
		a.cachedStr = a.addressParser.ToString(a.raw)
	}

	return a.cachedStr
}

type CardanoAddressInfo struct {
	AddressType  CardanoAddressType
	Network      CardanoNetworkType
	Payment      *CardanoAddressPayload
	Stake        *CardanoAddressPayload
	StakePointer *StakePointer
	Extra        []byte
}

func (cai CardanoAddressInfo) ToCardanoAddress() (*CardanoAddress, error) {
	parser, err := getAddressParser(cai.AddressType)
	if err != nil {
		return nil, err
	}

	return NewCardanoAddress(parser.FromCardanoAddressInfo(cai))
}

func NewBaseAddress(
	network CardanoNetworkType, paymentVerificationKey, stakeVerificationKey []byte,
) (*CardanoAddress, error) {
	paymentHash, err := GetKeyHashBytes(paymentVerificationKey)
	if err != nil {
		return nil, err
	}

	stakeHash, err := GetKeyHashBytes(stakeVerificationKey)
	if err != nil {
		return nil, err
	}

	return CardanoAddressInfo{
		AddressType: BaseAddress,
		Network:     network,
		Payment: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(paymentHash),
			IsScript: false,
		},
		Stake: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(stakeHash),
			IsScript: false,
		},
	}.ToCardanoAddress()
}

func NewEnterpriseAddress(
	network CardanoNetworkType, verificationKey []byte,
) (*CardanoAddress, error) {
	paymentHash, err := GetKeyHashBytes(verificationKey)
	if err != nil {
		return nil, err
	}

	return CardanoAddressInfo{
		AddressType: EnterpriseAddress,
		Network:     network,
		Payment: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(paymentHash),
			IsScript: false,
		},
	}.ToCardanoAddress()
}

func NewRewardAddress(
	network CardanoNetworkType, verificationKey []byte,
) (*CardanoAddress, error) {
	stakeHash, err := GetKeyHashBytes(verificationKey)
	if err != nil {
		return nil, err
	}

	return CardanoAddressInfo{
		AddressType: RewardAddress,
		Network:     network,
		Stake: &CardanoAddressPayload{
			Payload:  [KeyHashSize]byte(stakeHash),
			IsScript: false,
		},
	}.ToCardanoAddress()
}
