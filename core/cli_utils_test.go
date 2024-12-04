package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidCardanoAddress(t *testing.T) {
	t.Parallel()

	addresses := []string{
		"addr_test1vp4l5ka8jaqe32kygjemg6g745lxrn0mem7fxvuarrazmesdyntms",
		"addr_test1wpkr0wd9ggr3zmfs7a2pg845jld95nvjjzg4mnr0ewqmzmsf689u8",
		"addr1qy4h5rr93jh4x83448tk8y3whpddtyfq7g6pwr4tr3fuwcet0gxxtr902v0rt2whvwfzawz66kgjpu35zu82k8znca3sk9t664",
		"stake1uy4h5rr93jh4x83448tk8y3whpddtyfq7g6pwr4tr3fuwccdjgq9n",
		"addr1wpkr0wd9ggr3zmfs7a2pg845jld95nvjjzg4mnr0ewqmzmsf689u8",
		"addr_test1wpkr0wd9ggr3zmfs7a2pg845jld95nvjjzg4mnr0ewqmzmef689u8",
		"addr1qy4h5rr93jh4x83448tk8y3whpddtyfq7g6pwr4tr3fuwcet0gxxtr902v0rt2whvwfzawz66kgjpu35zu82k8znca3sket664",
		"stake1uy4h5rr93jh4x83448tk8y3whpddtyfq7g6pwr4tr3fuwccdjgq9n",
	}
	results := []bool{
		true,
		true,
		true,
		true,
		false,
		false,
		false,
		true,
	}

	cliUtils := NewCliUtils(ResolveCardanoCliBinary(TestNetNetwork))

	for i, addr := range addresses {
		ai, err := cliUtils.GetAddressInfo(addr)
		if results[i] {
			if strings.Contains(ai.Address, "stake") {
				assert.Equal(t, ai.Type, "stake")
			} else {
				assert.Equal(t, ai.Type, "payment")
			}

			assert.NoError(t, err)
		} else {
			require.ErrorContains(t, err, ErrInvalidAddressData.Error())
		}
	}
}
