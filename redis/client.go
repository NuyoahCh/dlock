package redis

import (
	rlock "github.com/Nuyoahch/dlock/internal/redis"
	"github.com/redis/go-redis/v9"
)

// NewClient 初始化客户端
func NewClient(rdb redis.Cmdable) *rlock.Client {
	return rlock.NewClient(rdb)
}
