package redis

import (
	"context"
	"github.com/Nuyoahch/dlock"
	"github.com/redis/go-redis/v9"
	"time"
)

// Client redis 客户端
type Client struct {
	rdb redis.Cmdable
}

// NewClient 初始化客户端
func NewClient(rdb redis.Cmdable) *Client {
	return &Client{rdb: rdb}
}

// NewLock 初始化 redis 分布式锁
func (c *Client) NewLock(ctx context.Context, key string, expiration time.Duration) (dlock.Lock, error) {
	return NewLock(c.rdb, key, expiration), nil
}
