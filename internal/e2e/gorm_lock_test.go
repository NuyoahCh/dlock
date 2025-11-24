package e2e

import (
	"context"
	glock "github.com/Nuyoahch/dlock/internal/gorm"
	"github.com/ecodeclub/ekit/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestGORMLock(t *testing.T) {
	db, err := gorm.Open(mysql.Open("root:root@tcp(localhost:13316)/dlock?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=True&loc=Local&timeout=1s&readTimeout=3s&writeTimeout=3s"))
	require.NoError(t, err)
	err = db.AutoMigrate(&glock.DistributedLock{})
	require.NoError(t, err)

	t.Run("cas first", func(t *testing.T) {
		suite.Run(t, &GORMLockTestSuite{
			LockTestSuite: &LockTestSuite{
				client: glock.NewCASFirstClient(db),
			},
			db: db,
		})
	})

	t.Run("insert first", func(t *testing.T) {
		suite.Run(t, &GORMLockTestSuite{
			LockTestSuite: &LockTestSuite{
				client: glock.NewInsertFirstClient(db),
			},
			db: db,
		})
	})
}

type GORMLockTestSuite struct {
	*LockTestSuite
	db *gorm.DB
}

func (s *GORMLockTestSuite) SetupSuite() {

}

func (s *GORMLockTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE distributed_locks").Error
	require.NoError(s.T(), err)
}

// TestGORMLock gorm 实现的独有特性测试
func (s *GORMLockTestSuite) TestLockExtra() {
	testCases := []struct {
		name string
		key  string
		// 用来测试的锁
		lock       func(t *testing.T) *glock.Lock
		expiration time.Duration
		before     func(t *testing.T)
		after      func(t *testing.T)
		wantErr    error
	}{
		{
			name: "insert 加锁成功，而后是 cas 尝试加锁失败，最后成功",
			key:  "gorm_key1",
			lock: func(t *testing.T) *glock.Lock {
				gl := glock.NewCASFirstLock(s.db, "gorm_key1", time.Minute)
				return gl
			},
			before: func(t *testing.T) {
				// 用 insert 模式加锁成功，5s 后过期
				gl := glock.NewInsertFirstLock(s.db, "gorm_key1", time.Second*5)
				gl.LockRetry, _ = retry.NewFixedIntervalRetryStrategy(time.Second, 10)
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
				defer cancel()
				err := gl.Lock(ctx)
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				// 如果 cas 成功，那么 version 会从 1 变为 2
				var res glock.DistributedLock
				err := s.db.WithContext(ctx).Where("`key` = ? AND version = ?", "gorm_key1", 2).First(&res).Error
				require.NoError(t, err)
				assert.True(s.T(), res.Expiration > time.Now().UnixMilli())
				assert.True(s.T(), res.Status == glock.StatusLocked)
			},
		},
		{
			name: "cas 尝试加锁成功，而后是 insert 加锁失败，最后成功",
			key:  "gorm_key2",
			lock: func(t *testing.T) *glock.Lock {
				gl := glock.NewInsertFirstLock(s.db, "gorm_key2", time.Minute)
				return gl
			},
			before: func(t *testing.T) {
				// 用 cas 模式加锁成功，5s 后过期
				gl := glock.NewCASFirstLock(s.db, "gorm_key2", time.Second*5)
				gl.LockRetry, _ = retry.NewFixedIntervalRetryStrategy(time.Second, 10)
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
				defer cancel()
				err := gl.Lock(ctx)
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				// 如果 insert 成功，那么 version 会从 1 变为 2
				var res glock.DistributedLock
				err := s.db.WithContext(ctx).Where("`key` = ? AND version = ?", "gorm_key2", 2).First(&res).Error
				require.NoError(t, err)
				assert.True(s.T(), res.Expiration > time.Now().UnixMilli())
				assert.True(s.T(), res.Status == glock.StatusLocked)
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
			defer cancel()
			lock := tc.lock(t)
			err := lock.Lock(ctx)
			assert.NoError(t, err)
		})
	}
}
