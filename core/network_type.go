package core

import "strings"

type CardanoNetworkType byte

const (
	TestNetProtocolMagic = uint(1097911063)
	MainNetProtocolMagic = uint(764824073)

	PrimeTestNetProtocolMagic  = uint(3311)
	PrimeMainNetProtocolMagic  = uint(764824073)
	VectorTestNetProtocolMagic = uint(1127)
	VectorMainNetProtocolMagic = uint(3327)

	VectorMainNetNetwork CardanoNetworkType = 3
	VectorTestNetNetwork CardanoNetworkType = 2
	MainNetNetwork       CardanoNetworkType = 1
	TestNetNetwork       CardanoNetworkType = 0
)

func (n CardanoNetworkType) GetPrefix() string {
	switch n {
	case VectorTestNetNetwork:
		return "vector_test"
	case VectorMainNetNetwork:
		return "vector"
	case MainNetNetwork:
		return "addr"
	case TestNetNetwork:
		return "addr_test"
	default:
		return "" // not handled but dont raise an error
	}
}

func (n CardanoNetworkType) GetStakePrefix() string {
	switch n {
	case MainNetNetwork, VectorMainNetNetwork:
		return "stake"
	case TestNetNetwork, VectorTestNetNetwork:
		return "stake_test"
	default:
		return "" // not handled but dont raise an error
	}
}

func (n CardanoNetworkType) IsMainNet() bool {
	return n == MainNetNetwork
}

func IsAddressWithValidPrefix(addr string) bool {
	return strings.HasPrefix(addr, "addr") ||
		strings.HasPrefix(addr, "vector") ||
		strings.HasPrefix(addr, "stake")
}
