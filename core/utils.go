package core

import (
	"context"
	"errors"
	"time"
)

type IsRecoverableErrorFn func(err error) bool

var ErrWaitForTransactionTimeout = errors.New("timeout while waiting for transaction")

// GetUtxosSum returns sum of all utxos
func GetUtxosSum(utxos []Utxo) (sum uint64) {
	for _, utxo := range utxos {
		sum += utxo.Amount
	}

	return sum
}

// GetOutputsSum returns sum of tx outputs
func GetOutputsSum(outputs []TxOutput) (receiversSum uint64) {
	for _, x := range outputs {
		receiversSum += x.Amount
	}

	return receiversSum
}

// WaitForAmount waits for address to have amount specified by cmpHandler
func WaitForAmount(ctx context.Context, txRetriever IUTxORetriever,
	addr string, cmpHandler func(uint64) bool, numRetries int, waitTime time.Duration,
	isRecoverableError ...IsRecoverableErrorFn,
) error {
	return ExecuteWithRetry(ctx, numRetries, waitTime, func() (bool, error) {
		utxos, err := txRetriever.GetUtxos(ctx, addr)

		return err == nil && cmpHandler(GetUtxosSum(utxos)), err
	}, isRecoverableError...)
}

// WaitForTxHashInUtxos waits until tx with txHash occurs in addr utxos
func WaitForTxHashInUtxos(ctx context.Context, txRetriever IUTxORetriever,
	addr string, txHash string, numRetries int, waitTime time.Duration,
	isRecoverableError ...IsRecoverableErrorFn,
) error {
	return ExecuteWithRetry(ctx, numRetries, waitTime, func() (bool, error) {
		utxos, err := txRetriever.GetUtxos(ctx, addr)
		if err != nil {
			return false, err
		}

		for _, x := range utxos {
			if x.Hash == txHash {
				return true, nil
			}
		}

		return false, nil
	}, isRecoverableError...)
}

// WaitForTransaction waits for transaction to be included in block
func WaitForTransaction(ctx context.Context, txRetriever ITxRetriever,
	hash string, numRetries int, waitTime time.Duration,
	isRecoverableError ...IsRecoverableErrorFn,
) (res map[string]interface{}, err error) {
	err = ExecuteWithRetry(ctx, numRetries, waitTime, func() (bool, error) {
		res, err = txRetriever.GetTxByHash(ctx, hash)

		return err == nil && res != nil, err
	}, isRecoverableError...)

	return res, err
}

// ExecuteWithRetry attempts to execute the provided executeFn function multiple times
// if the call fails with a recoverable error. It retries up to numRetries times.
func ExecuteWithRetry(ctx context.Context,
	numRetries int, waitTime time.Duration,
	executeFn func() (bool, error),
	isRecoverableError ...IsRecoverableErrorFn,
) error {
	for count := 0; count < numRetries; count++ {
		stop, err := executeFn()
		if err != nil {
			if len(isRecoverableError) == 0 || !isRecoverableError[0](err) {
				return err
			}
		} else if stop {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
	}

	return ErrWaitForTransactionTimeout
}
