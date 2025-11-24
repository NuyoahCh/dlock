package e2e

import (
	"context"
	_ "embed"
	"errors"
	"github.com/Nuyoahch/dlock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type LockTestSuite struct {
	suite.Suite
	client dlock.Client
}

func (s *LockTestSuite) TestLock() {
	testCases := []struct {
		name    string
		before  func(t *testing.T) dlock.Lock
		after   func(t *testing.T)
		wantErr error
	}{
		{
			name: "加锁成功",
			before: func(t *testing.T) dlock.Lock {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				lock, err := s.client.NewLock(ctx, "lock-success-key", time.Minute)
				require.NoError(t, err)
				return lock
			},
			after:   func(t *testing.T) {},
			wantErr: nil,
		},
		{
			name: "加锁失败，别人持有锁",
			before: func(t *testing.T) dlock.Lock {
				key := "lock-failed-key"
				ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel1()
				// 模拟别人加锁成功
				lock1, err := s.client.NewLock(ctx1, key, time.Minute)
				require.NoError(t, err)
				err = lock1.Lock(ctx1)
				require.NoError(t, err)

				ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel2()
				lock2, err := s.client.NewLock(ctx2, key, time.Minute)
				require.NoError(t, err)
				return lock2
			},
			after:   func(t *testing.T) {},
			wantErr: dlock.ErrLocked,
		},
		{
			name: "加锁成功，原持有人崩溃",
			before: func(t *testing.T) dlock.Lock {
				key := "lock-retry-success-key"
				ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel1()
				// 模拟别人加锁成功
				lock1, err := s.client.NewLock(ctx1, key, time.Second*2)
				require.NoError(t, err)
				err = lock1.Lock(ctx1)
				require.NoError(t, err)
				// 模拟到期未续约
				time.Sleep(time.Second * 4)

				ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel2()
				lock2, err := s.client.NewLock(ctx2, key, time.Second*10)
				require.NoError(t, err)
				return lock2
			},
			after:   func(t *testing.T) {},
			wantErr: nil,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			lock := tc.before(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			err := lock.Lock(ctx)
			cancel()
			assert.True(t, errors.Is(err, tc.wantErr))
			tc.after(t)
		})
	}
}

func (s *LockTestSuite) TestUnLock() {
	testCases := []struct {
		name    string
		before  func(t *testing.T) dlock.Lock
		after   func(t *testing.T)
		wantErr error
	}{
		{
			name: "解锁成功",
			before: func(t *testing.T) dlock.Lock {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				lock, err := s.client.NewLock(ctx, "unlock-success-key", time.Minute)
				require.NoError(t, err)
				err = lock.Lock(ctx)
				require.NoError(t, err)
				return lock
			},
			after:   func(t *testing.T) {},
			wantErr: nil,
		},
		{
			name: "解锁失败-不是自己的锁",
			before: func(t *testing.T) dlock.Lock {
				key := "unlock-failed-key"
				ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel1()
				lock1, err := s.client.NewLock(ctx1, key, time.Second*2)
				require.NoError(t, err)
				// 先拿到锁
				err = lock1.Lock(ctx1)
				require.NoError(t, err)

				// 锁到期未续约
				time.Sleep(time.Second * 4)

				ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel2()
				// 让另一个人持有锁
				lock2, err := s.client.NewLock(ctx2, key, time.Second*10)
				require.NoError(t, err)
				err = lock2.Lock(ctx2)
				require.NoError(t, err)

				return lock1
			},
			after:   func(t *testing.T) {},
			wantErr: dlock.ErrLockNotHold,
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			lock := tc.before(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			err := lock.Unlock(ctx)
			cancel()
			assert.True(t, errors.Is(err, tc.wantErr))
			tc.after(t)
		})
	}
}

func (s *LockTestSuite) TestRefresh() {
	testCases := []struct {
		name    string
		before  func(t *testing.T) dlock.Lock
		after   func(t *testing.T)
		wantErr error
	}{
		{
			name: "刷新成功",
			before: func(t *testing.T) dlock.Lock {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				lock, err := s.client.NewLock(ctx, "refresh-success-key", time.Minute)
				require.NoError(t, err)
				err = lock.Lock(ctx)
				require.NoError(t, err)
				return lock
			},
			after: func(t *testing.T) {},
		},
		{
			name: "刷新失败-不是自己的锁",
			before: func(t *testing.T) dlock.Lock {
				key := "refresh-failed-key"
				ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel1()
				lock1, err := s.client.NewLock(ctx1, key, time.Second*2)
				require.NoError(t, err)
				// 先加锁成功
				err = lock1.Lock(ctx1)
				require.NoError(t, err)

				// 锁到期未续约
				time.Sleep(time.Second * 4)

				ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel2()
				// 让另一个人持有锁
				lock2, err := s.client.NewLock(ctx2, key, time.Second*10)
				require.NoError(t, err)
				err = lock2.Lock(ctx2)
				require.NoError(t, err)

				return lock1
			},
			after:   func(t *testing.T) {},
			wantErr: dlock.ErrLockNotHold,
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			lock := tc.before(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			err := lock.Refresh(ctx)
			cancel()
			assert.True(t, errors.Is(err, tc.wantErr))
			tc.after(t)
		})
	}
}

func (s *LockTestSuite) TestAutoExpiration() {
	testCases := []struct {
		name    string
		before  func(t *testing.T) dlock.Lock
		after   func(t *testing.T)
		wantErr error
	}{
		{
			name: "测试自动过期",
			before: func(t *testing.T) dlock.Lock {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				lock, err := s.client.NewLock(ctx, "refresh-success-key", time.Second*4)
				require.NoError(t, err)
				err = lock.Lock(ctx)
				require.NoError(t, err)

				time.Sleep(time.Second * 5)
				return lock
			},
			after:   func(t *testing.T) {},
			wantErr: dlock.ErrLockNotHold,
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			lock := tc.before(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			err := lock.Refresh(ctx)
			cancel()
			assert.True(t, errors.Is(err, tc.wantErr))
			tc.after(t)
		})
	}
}
