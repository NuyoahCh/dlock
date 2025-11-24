package e2e

import (
	"context"
	"fmt"
	etcd_lock "github.com/Nuyoahch/dlock/internal/etcd"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"

	"testing"
)

// 针对不同存储后端的分布式锁实现进行端到端测试
func TestEtcdLock(t *testing.T) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})

	require.NoError(t, err)
	conn := cli.ActiveConnection()
	require.NotNil(t, conn)

	//err := rdb.Ping(context.Background()).Err()
	//require.NoError(t, err)
	suite.Run(t, &EtcdLockTestSuite{
		LockTestSuite: &LockTestSuite{
			client: etcd_lock.NewEtcdClient(cli),
		},
	})
}

type EtcdLockTestSuite struct {
	*LockTestSuite
	c *clientv3.Client
}

func (s *EtcdLockTestSuite) SetupSuite() {
	if s.c != nil {
		_, _ = s.c.Delete(context.Background(), "", clientv3.WithPrefix())
	}
}

func (s *EtcdLockTestSuite) TearDownTest() {
	if s.c != nil {
		_, _ = s.c.Delete(context.Background(), "", clientv3.WithPrefix())
	}
}

// TestGrant test etcd lease grant and need to revoke
func TestGrant(t *testing.T) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	// 创建租约(不关联任何键)
	lease := clientv3.NewLease(cli)
	grantResp, err := lease.Grant(context.TODO(), 50) // 初始TTL 10秒
	if err != nil {
		panic(err)
	}
	leaseID := grantResp.ID
	fmt.Printf("Created lease ID: %x (not attached to any key)\n", leaseID)

	// 启动续约
	//keepAliveCh, err := lease.KeepAlive(context.TODO(), leaseID)
	//if err != nil {
	//	panic(err)
	//}
	//
	//// 监控续约响应
	//go func() {
	//	for ka := range keepAliveCh {
	//		if ka == nil {
	//			fmt.Println("Lease expired or cancelled")
	//			return
	//		}
	//		fmt.Printf("Lease kept alive at %s: ID=%x, TTL=%d\n",
	//			time.Now().Format("15:04:05"), ka.ID, ka.TTL)
	//	}
	//}()

	// 定期检查租约状态
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for i := 0; i < 10; i++ {
		<-ticker.C
		resp, err := lease.TimeToLive(context.TODO(), leaseID)
		if err != nil {
			fmt.Println("Error checking lease:", err)
			continue
		}
		if resp.TTL == -1 {
			fmt.Println("Lease has expired")
		} else {
			fmt.Printf("Lease status: ID=%x, TTL=%d, GrantedTTL=%d\n",
				resp.ID, resp.TTL, resp.GrantedTTL)
		}
	}

	// 显式撤销租约(可选)
	_, err = lease.Revoke(context.TODO(), leaseID)
	if err != nil {
		fmt.Println("Error revoke lease:", err)
	}
	fmt.Println("Lease revoked")
	resp, err := lease.TimeToLive(context.TODO(), leaseID)
	if err != nil {
		fmt.Println("Error checking lease:", err)
	}

	if resp.TTL == -1 {
		fmt.Println("Lease has expired")
	} else {
		fmt.Printf("Lease status: ID=%x, TTL=%d, GrantedTTL=%d\n",
			resp.ID, resp.TTL, resp.GrantedTTL)
	}
}
