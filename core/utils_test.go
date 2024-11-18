package core

import (
	"context"
	"errors"
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

	res := GetUtxosSum([]Utxo{
		{Amount: 100}, {Amount: 200},
	})
	require.Equal(t, uint64(300), res)
}

func TestGetOutputsSum(t *testing.T) {
	t.Parallel()

	res := GetOutputsSum([]TxOutput{
		{Amount: 100}, {Amount: 200},
	})
	require.Equal(t, uint64(300), res)
}

type txRetrieverMock struct {
	getUtxosFn func(ctx context.Context, addr string) ([]Utxo, error)
}

func (m txRetrieverMock) GetUtxos(ctx context.Context, addr string) ([]Utxo, error) {
	return m.getUtxosFn(ctx, addr)
}
