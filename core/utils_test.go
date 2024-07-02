package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWaitForTransaction(t *testing.T) {
	t.Parallel()

	var (
		errWait = errors.New("hello wait")
		txInfo  = map[string]interface{}{"block": "0x1001"}
	)

	mock := &txRetrieverMock{
		getTxByHashFn: func(_ context.Context, hash string) (map[string]interface{}, error) {
			switch hash {
			case "a":
				return nil, errWait
			case "b":
				return txInfo, nil
			default:
				return nil, nil
			}
		},
	}

	_, err := WaitForTransaction(context.Background(), mock, "a", 10, time.Second)
	require.ErrorIs(t, err, errWait)

	_, err = WaitForTransaction(context.Background(), mock, "not_exist", 10, time.Millisecond*5)
	require.ErrorIs(t, err, ErrWaitForTransactionTimeout)

	data, err := WaitForTransaction(context.Background(), mock, "b", 10, time.Millisecond*5)
	require.NoError(t, err)
	require.Equal(t, txInfo, data)

	ctx, cncl := context.WithCancel(context.Background())
	go func() {
		cncl()
	}()

	_, err = WaitForTransaction(ctx, mock, "not_exist", 10, time.Millisecond*5)
	require.ErrorIs(t, err, ctx.Err())

	_, err = WaitForTransaction(context.Background(), mock, "a",
		10, time.Millisecond*10, func(err error) bool { return errors.Is(err, errWait) })
	require.ErrorIs(t, err, ErrWaitForTransactionTimeout)
}

func TestWaitForAmount(t *testing.T) {
	t.Parallel()

	var (
		errWait = errors.New("hello wait")
		txInfo1 = []Utxo{
			{Amount: 10},
		}
		txInfo2 = []Utxo{
			{Amount: 10},
			{Amount: 20},
		}
	)

	mock := &txRetrieverMock{
		getUtxosFn: func(_ context.Context, addr string) ([]Utxo, error) {
			switch addr {
			case "a":
				return nil, errWait
			case "b":
				return txInfo1, nil
			case "c":
				return txInfo2, nil
			default:
				return nil, nil
			}
		},
	}

	cmpHandler := func(val uint64) bool {
		return val >= 30
	}

	err := WaitForAmount(context.Background(), mock, "a", cmpHandler, 10, time.Millisecond*10)
	require.ErrorIs(t, err, errWait)

	err = WaitForAmount(context.Background(), mock, "b", cmpHandler, 10, time.Millisecond*10)
	require.ErrorIs(t, err, ErrWaitForTransactionTimeout)

	err = WaitForAmount(context.Background(), mock, "c", cmpHandler, 10, time.Millisecond*10)
	require.NoError(t, err)

	ctx, cncl := context.WithCancel(context.Background())
	go func() {
		cncl()
	}()

	err = WaitForAmount(ctx, mock, "not_exists", cmpHandler, 1000, time.Millisecond*10)
	require.ErrorIs(t, err, ctx.Err())

	err = WaitForAmount(context.Background(), mock, "a", cmpHandler,
		10, time.Millisecond*10, func(err error) bool { return true })
	require.ErrorIs(t, err, ErrWaitForTransactionTimeout)
}

func TestWaitForTxHashInUtxos(t *testing.T) {
	t.Parallel()

	var (
		errWait = errors.New("hello wait")
		txInfo1 = []Utxo{
			{Hash: "0x1"},
		}
		txInfo2 = []Utxo{
			{Hash: "0x1"},
			{Hash: "0x3"},
		}
	)

	mock := &txRetrieverMock{
		getUtxosFn: func(_ context.Context, addr string) ([]Utxo, error) {
			switch addr {
			case "a":
				return nil, errWait
			case "b":
				return txInfo1, nil
			case "c":
				return txInfo2, nil
			default:
				return nil, nil
			}
		},
	}

	err := WaitForTxHashInUtxos(context.Background(), mock, "a", "0x1", 10, time.Millisecond*10)
	require.ErrorIs(t, err, errWait)

	err = WaitForTxHashInUtxos(context.Background(), mock, "b", "0x2", 10, time.Millisecond*10)
	require.ErrorIs(t, err, ErrWaitForTransactionTimeout)

	err = WaitForTxHashInUtxos(context.Background(), mock, "c", "0x3", 10, time.Millisecond*10)
	require.NoError(t, err)

	ctx, cncl := context.WithCancel(context.Background())
	go func() {
		cncl()
	}()

	err = WaitForTxHashInUtxos(ctx, mock, "not_exists", "0x3", 1000, time.Millisecond*10)
	require.ErrorIs(t, err, ctx.Err())

	err = WaitForTxHashInUtxos(context.Background(), mock, "a", "0x1",
		10, time.Millisecond*10, func(err error) bool { return true })
	require.ErrorIs(t, err, ErrWaitForTransactionTimeout)
}

type txRetrieverMock struct {
	getTxByHashFn func(ctx context.Context, hash string) (map[string]interface{}, error)
	getUtxosFn    func(ctx context.Context, addr string) ([]Utxo, error)
}

func (m txRetrieverMock) GetTxByHash(ctx context.Context, hash string) (map[string]interface{}, error) {
	return m.getTxByHashFn(ctx, hash)
}

func (m txRetrieverMock) GetUtxos(ctx context.Context, addr string) ([]Utxo, error) {
	return m.getUtxosFn(ctx, addr)
}
