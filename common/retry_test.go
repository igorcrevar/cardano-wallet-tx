package common

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteWithRetry(t *testing.T) {
	t.Parallel()

	var (
		errWait = errors.New("hello wait")
		ctx     = context.Background()
	)

	options := []RetryConfigOption{
		WithRetryCount(10),
		WithRetryWaitTime(time.Millisecond * 5),
	}

	_, err := ExecuteWithRetry(ctx, func(_ context.Context) (int, error) {
		return 0, errWait
	}, options...)

	require.ErrorIs(t, err, errWait)

	i := 0

	_, err = ExecuteWithRetry(ctx, func(_ context.Context) (int, error) {
		i++
		if i&1 == 1 {
			return 0, &net.DNSError{IsTimeout: true}
		} else if i&3 == 0 {
			return 0, ErrRetryTryAgain
		}

		return 0, errors.New("status code 500")
	}, options...)

	require.ErrorIs(t, err, ErrRetryTimeout)
	require.Equal(t, 10, i)

	ctxWithCancel, cncl := context.WithCancel(ctx)
	go cncl()

	_, err = ExecuteWithRetry(ctxWithCancel, func(_ context.Context) (int, error) {
		return 0, errors.New("status code 500")
	}, options...)

	require.ErrorIs(t, err, ctxWithCancel.Err())

	result, err := ExecuteWithRetry(ctx, func(cnt context.Context) (int, error) {
		return 8930, nil
	}, options...)

	require.NoError(t, err)
	require.Equal(t, 8930, result)
}

func TestIsRetryableError(t *testing.T) {
	assert.False(t, IsRetryableError(nil))
	assert.False(t, IsRetryableError(context.Canceled))
	assert.False(t, IsRetryableError(context.DeadlineExceeded))
	assert.True(t, IsRetryableError(ErrRetryTryAgain))
	assert.False(t, IsRetryableError(net.ErrClosed))
	assert.True(t, IsRetryableError(&net.DNSError{IsTimeout: true}))
	assert.True(t, IsRetryableError(errors.New("status code 500")))
}
