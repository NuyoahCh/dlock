package dlock

import (
	"context"
	"time"
)

// Client 分布式锁客户端
type Client interface {
	NewLock(ctx context.Context, key string, expiration time.Duration) (Lock, error)
}

// Lock 锁相关操作
type Lock interface {
	// Lock 加锁
	Lock(ctx context.Context) error

	// Unlock 解锁
	Unlock(ctx context.Context) error

	// Refresh 续约， 延长过期时间
	Refresh(ctx context.Context) error
}
