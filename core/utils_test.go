package core

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsTxInUtxos(t *testing.T) {
	t.Parallel()

	const existingHash = "0xfff03"

	var (
		ctx       = context.Background()
		errCustom = errors.New("custom error")
		utxos     = []Utxo{
			{Hash: existingHash},
		}
		mock = &txRetrieverMock{
			getUtxosFn: func(_ context.Context, addr string) ([]Utxo, error) {
				switch addr {
				case "a":
					return nil, errCustom
				default:
					return utxos, nil
				}
			},
		}
	)

	_, err := IsTxInUtxos(ctx, mock, "a", existingHash)
	require.ErrorIs(t, err, errCustom)

	res, err := IsTxInUtxos(ctx, mock, "b", "00dfg")
	require.NoError(t, err)
	require.False(t, res)

	res, err = IsTxInUtxos(ctx, mock, "b", existingHash)
	require.NoError(t, err)
	require.True(t, res)
}

func TestGetUtxosSum(t *testing.T) {
	t.Parallel()

	result := GetUtxosSum([]Utxo{
		{
			Amount: 200,
		},
		{
			Amount: 0,
			Tokens: []TokenAmount{
				NewTokenAmount("1", "1", 100),
				NewTokenAmount("2", "1", 400),
			},
		},
		{
			Amount: 300,
			Tokens: []TokenAmount{
				NewTokenAmount("3", "3", 20),
				NewTokenAmount("2", "1", 150),
			},
		},
	})

	require.Equal(t, 4, len(result))
	require.Equal(t, uint64(500), result[AdaTokenName])
	require.Equal(t, uint64(20), result["3.33"])
	require.Equal(t, uint64(550), result["2.31"])
	require.Equal(t, uint64(100), result["1.31"])
}

func TestGetOutputsSum(t *testing.T) {
	t.Parallel()

	const psHash = "29f8873beb52e126f207a2dfd50f7cff556806b5b4cba9834a7b26a8"

	// Kash_Token
	token1, err := NewTokenAmountWithFullName(fmt.Sprintf("%s.4b6173685f546f6b656e", psHash), 190, true)
	require.NoError(t, err)

	// Route3 token
	token2, err := NewTokenAmountWithFullName(fmt.Sprintf("%s.Route3", psHash), 720, false)
	require.NoError(t, err)

	result := GetOutputsSum([]TxOutput{
		NewTxOutput("", 200),
		NewTxOutput("", 300, NewTokenAmount("1", "2", 10), token1),
		NewTxOutput("", 100, token2),
		NewTxOutput("", 50, NewTokenAmount("1", "2", 30)),
	})

	require.Equal(t, 4, len(result))
	require.Equal(t, uint64(650), result[AdaTokenName])
	require.Equal(t, uint64(40), result["1.32"])
	require.Equal(t, uint64(720), result[fmt.Sprintf("%s.526f75746533", psHash)])
	require.Equal(t, uint64(190), result[fmt.Sprintf("%s.4b6173685f546f6b656e", psHash)])
}

func TestGetTokensFromSumMap(t *testing.T) {
	tokens := []TokenAmount{
		NewTokenAmount("29f8873beb52e126f207a2dfd50f7cff556806b5b4cba9834a7b26a8", "Route3", 54),
		NewTokenAmount("72f3d1e6c885e4d0bdcf5250513778dbaa851c0b4bfe3ed4e1bcceb0", "Kash_Token", 180),
	}
	sum := map[string]uint64{
		"72f3d1e6c885e4d0bdcf5250513778dbaa851c0b4bfe3ed4e1bcceb0.4b6173685f546f6b656e": 180,
		"29f8873beb52e126f207a2dfd50f7cff556806b5b4cba9834a7b26a8.526f75746533":         54,
		AdaTokenName: 1,
	}

	res, err := GetTokensFromSumMap(sum)

	sort.Slice(res, func(i, j int) bool {
		return res[i].TokenName() < res[j].TokenName()
	})

	require.NoError(t, err)
	require.Equal(t, tokens, res)

	res, err = GetTokensFromSumMap(sum, "72f3d1e6c885e4d0bdcf5250513778dbaa851c0b4bfe3ed4e1bcceb0.4b6173685f546f6b656e")

	require.NoError(t, err)
	require.Equal(t, tokens[:1], res)

	res, err = GetTokensFromSumMap(sum,
		"72f3d1e6c885e4d0bdcf5250513778dbaa851c0b4bfe3ed4e1bcceb0.4b6173685f546f6b656e",
		"29f8873beb52e126f207a2dfd50f7cff556806b5b4cba9834a7b26a8.526f75746533")

	require.NoError(t, err)
	require.Equal(t, tokens[:0], res)

	res, err = GetTokensFromSumMap(sum,
		"29f8873beb52e126f207a2dfd50f7cff556806b5b4cba9834a7b26a8.526f75746533")

	require.NoError(t, err)
	require.Equal(t, tokens[1:], res)
}

type txRetrieverMock struct {
	getUtxosFn func(ctx context.Context, addr string) ([]Utxo, error)
}

func (m txRetrieverMock) GetUtxos(ctx context.Context, addr string) ([]Utxo, error) {
	return m.getUtxosFn(ctx, addr)
}
