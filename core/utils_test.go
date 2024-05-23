package core

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
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

	for i, addr := range addresses {
		ai, err := GetAddressInfo(addr)
		if results[i] {
			if strings.Contains(ai.Address, "stake") {
				assert.Equal(t, ai.Type, "stake")
			} else {
				assert.Equal(t, ai.Type, "payment")
			}

			assert.NoError(t, err)
		} else {
			require.ErrorContains(t, err, ErrInvalidAddressInfo.Error())
		}
	}
}

func TestVerifyWitness(t *testing.T) {
	t.Parallel()

	var (
		txHash         = "7e8b59e41d2ba71888272a14cff401268fa01dceb19014f5dda7763334b8f221"
		signingKey, _  = hex.DecodeString("1217236ac24d8ac12684b308cf9468f68ef5283096896dc1c5c3caf8351e2847")
		witnessCbor, _ = hex.DecodeString("8258203e9d3a6f792c9820ab4423e41256e4b6e2ae1f456318f9d936fc70e0eafdc76f58402992d7fbc6fb155b7cc83223c80bf9b0ddbfe24ff260600897a06e8050f6596a76defeea6a86048605f8f7c27ef53da318aa02838532ea1876aac876b2491a01")
		txHashBytes, _ = hex.DecodeString(txHash)
	)

	verificationKey := GetVerificationKeyFromSigningKey(signingKey)

	signature, vKeyWitness, err := TxWitnessRaw(witnessCbor).GetSignatureAndVKey()
	require.NoError(t, err)

	require.NoError(t, err)
	assert.Equal(t, verificationKey, vKeyWitness)

	signedMessageCbor, err := SignMessage(signingKey, verificationKey, txHashBytes)
	require.NoError(t, err)
	assert.Equal(t, signature, signedMessageCbor)

	dummySignature, err := SignMessage(signingKey, verificationKey, append([]byte{255}, txHash[1:]...))
	require.NoError(t, err)

	dummyWitness, err := cbor.Marshal([][]byte{verificationKey, dummySignature})
	require.NoError(t, err)

	assert.NoError(t, VerifyWitness(txHash, witnessCbor))
	assert.ErrorIs(t, VerifyWitness(strings.Replace(txHash, "7e", "7f", 1), witnessCbor), ErrInvalidSignature)
	assert.ErrorIs(t, VerifyWitness(txHash, dummyWitness), ErrInvalidSignature)
}

func TestVerifyMessage(t *testing.T) {
	t.Parallel()

	const msg = "Hello World!"

	priv, pub, err := GenerateKeyPair()
	require.NoError(t, err)

	signature, err := SignMessage(priv, pub, []byte(msg))
	require.NoError(t, err)

	require.NoError(t, VerifyMessage([]byte(msg), pub, signature))
	require.ErrorIs(t, VerifyMessage([]byte("invalid msg"), pub, signature), ErrInvalidSignature)
}

func TestKeyHash(t *testing.T) {
	t.Parallel()

	baseDirectory, err := os.MkdirTemp("", "key-hash-test")
	require.NoError(t, err)

	defer func() {
		os.RemoveAll(baseDirectory)
		os.Remove(baseDirectory)
	}()

	const accountsNumber = 20

	walletManager := NewStakeWalletManager()

	for i := 0; i < accountsNumber; i++ {
		wallet, err := walletManager.Create(path.Join(baseDirectory, fmt.Sprintf("a_%d", i)), true)
		require.NoError(t, err)

		keyHash, err := GetKeyHash(wallet.GetVerificationKey())
		require.NoError(t, err)

		assert.Equal(t, wallet.GetKeyHash(), keyHash)
	}
}

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

	cmpHandler := func(val *big.Int) bool {
		return val.Cmp(new(big.Int).SetUint64(30)) >= 0
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
