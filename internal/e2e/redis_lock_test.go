package e2e

import (
	"context"
	rlock "github.com/Nuyoahch/dlock/internal/redis"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

func TestRedisLock(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	err := rdb.Ping(context.Background()).Err()
	require.NoError(t, err)
	suite.Run(t, &RedisLockTestSuite{
		LockTestSuite: &LockTestSuite{
			client: rlock.NewClient(rdb),
		},
		rdb: rdb,
	})
}

type RedisLockTestSuite struct {
	*LockTestSuite
	rdb *redis.Client
}

func (s *RedisLockTestSuite) SetupSuite() {

}

func (s *RedisLockTestSuite) TearDownTest() {
	err := s.rdb.FlushDB(context.Background()).Err()
	require.NoError(s.T(), err)
}
