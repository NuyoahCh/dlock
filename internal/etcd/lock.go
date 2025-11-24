package etcd

import (
	"errors"
	"fmt"
	"github.com/Nuyoahch/dlock"
	"github.com/Nuyoahch/dlock/internal/pkg"
	"github.com/ecodeclub/ekit/retry"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
	"strconv"
	"time"
)

var _ dlock.Lock = (*etcdLock)(nil)

// etcdLock 结构体
type etcdLock struct {
	client     *clientv3.Client
	leaseID    clientv3.LeaseID
	key        string
	expiration time.Duration
	_          pkg.NoCopy

	lockRetry   retry.Strategy
	lockTimeout time.Duration
}

// Lock 加锁，成功后才能进行 Unlock
func (l *etcdLock) Lock(ctx context.Context) (err error) {
	needRevoke := true
	c1, cancel1 := context.WithTimeout(ctx, l.lockTimeout)
	defer cancel1()

	// 这里可以引入重试机制
	leaseResp, err := l.client.Grant(c1, int64(l.expiration.Seconds()))
	if err != nil {
		return fmt.Errorf("创建租约失败: %w", err)
	}
	defer func() {
		c2, cancel2 := context.WithTimeout(context.Background(), l.lockTimeout)
		defer cancel2()
		// 这里个人认为其实可以不用控制，只要是检测没有加锁成功直接解除续约
		// 因为，返回err了以后，调用方都认为锁没有获取到，那么就应该是是需要重试。
		// 而哪怕是这里解除续约如果超时，最坏的结果也是锁没有释放，但是不会有其他业务获取到锁。
		if needRevoke {
			_, err1 := l.client.Revoke(c2, leaseResp.ID)
			if err1 != nil {
				err = fmt.Errorf("%w ,且解除租约失败 %w", err, err1)
			}
		}
	}()

	sid := strconv.FormatInt(int64(leaseResp.ID), 10)
	return retry.Retry(ctx, l.lockRetry, func() (err error) {
		withTimeoutCtx, cancel := context.WithTimeout(ctx, l.lockTimeout)
		defer cancel()

		// 如果用户另一个协程加锁后释放锁，这个地方还是能进去
		// 但是逻辑上锁已经释放了，可以重新加锁
		txn := l.client.Txn(withTimeoutCtx).
			If(clientv3.Compare(clientv3.CreateRevision(l.key), "=", 0)).
			Then(clientv3.OpPut(l.key, sid, clientv3.WithLease(leaseResp.ID)))

		txnResp, err := txn.Commit()
		// 超时连不上集群才会到这里
		if err != nil {
			return fmt.Errorf("etcd加锁失败: %w", err)
		}
		// 这里是竞争失败
		if !txnResp.Succeeded {
			return dlock.ErrLocked
		}

		l.leaseID = leaseResp.ID
		needRevoke = false
		return nil
	})
}

// Unlock 释放锁 只能保证当前 leaseID 释放对应的key
func (l *etcdLock) Unlock(ctx context.Context) error {
	withTimeoutCtx, cancel := context.WithTimeout(ctx, l.lockTimeout)
	defer cancel()
	_, err := l.client.Revoke(withTimeoutCtx, l.leaseID)
	if err != nil {
		if errors.Is(err, rpctypes.ErrLeaseNotFound) {
			return dlock.ErrLockNotHold
		}
		return err
	}
	return nil
}

// Refresh 续约， 延长过期时间
func (l *etcdLock) Refresh(ctx context.Context) error {
	withTimeoutCtx, cancel := context.WithTimeout(ctx, l.lockTimeout)
	defer cancel()
	_, err := l.client.KeepAliveOnce(withTimeoutCtx, l.leaseID)
	if err != nil {
		if errors.Is(err, rpctypes.ErrLeaseNotFound) {
			return dlock.ErrLockNotHold
		}
		return err
	}
	return nil
}
