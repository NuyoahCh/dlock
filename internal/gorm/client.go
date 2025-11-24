package gorm

import (
	"context"
	"github.com/Nuyoahch/dlock"
	"gorm.io/gorm"
	"time"
)

// Client 数据库分布式锁客户端
type Client struct {
	db *gorm.DB
	// 为了测试不得已而为之
	NewDLock func(db *gorm.DB, key string, expiration time.Duration) *Lock
}

func NewCASFirstClient(db *gorm.DB) *Client {
	return &Client{db: db, NewDLock: NewCASFirstLock}
}

func NewInsertFirstClient(db *gorm.DB) *Client {
	return &Client{db: db, NewDLock: NewInsertFirstLock}
}

func (c *Client) NewLock(ctx context.Context, key string, expiration time.Duration) (dlock.Lock, error) {
	return c.NewDLock(c.db, key, expiration), nil
}

func (c *Client) InitTable() error {
	return c.db.AutoMigrate(&DistributedLock{})
}
