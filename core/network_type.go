package core

type CardanoNetworkType byte

const (
	MainNetNetwork CardanoNetworkType = 1
	TestNetNetwork CardanoNetworkType = 0
)

func (n CardanoNetworkType) GetPrefix() string {
	switch n {
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
	case MainNetNetwork:
		return "stake"
	case TestNetNetwork:
		return "stake_test"
	default:
		return "" // not handled but dont raise an error
	}
}

func (n CardanoNetworkType) IsMainNet() bool {
	return n == MainNetNetwork
}
