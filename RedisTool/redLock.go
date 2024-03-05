package RedisTool

import (
	"context"
	"fmt"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/redis/go-redis/v9"
	"strconv"
	"sync"
	"time"
)

// NewRedLock 创建一个数量为num的redis集群
func NewRedLock(num int) *RedLock {
	var redisClients []*redisLockClient
	var wg sync.WaitGroup
	for i := 0; i < num; i++ {
		// 使用并行的方式完成
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			client := &redisLockClient{
				token: gofakeit.UUID(),
				rdb: redis.NewClient(&redis.Options{
					Addr:     "localhost:" + strconv.Itoa(idx+6379),
					Password: "123456",
				}),
			}
			redisClients = append(redisClients, client)
		}(i)
	}
	wg.Wait()
	return &RedLock{
		watchDog:      nil,
		redisClients:  redisClients,
		renewInterval: RENEW_INTERVAL,
	}
}

// startWatchDog 启动看门狗
func (rl *RedLock) startWatchDog(idxList []int, lop LockOptions, ctx context.Context) {
	if rl.watchDog == nil {
		rl.watchDog = make(chan struct{})
	}
	// 设定计时器
	ticker := time.NewTicker(rl.renewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// 每隔一段时间就续租
			for _, idx := range idxList {
				result, err := rl.redisClients[idx].rdb.Eval(ctx, EXPIRE_SCRIPT, []string{lop.Key, lop.Token}, lop.Expiration).Result()
				if err != nil || result == 0 {
					panic(ErrLockRent)
				}
			}
		case <-rl.watchDog:
			return
		}
	}
}

// Lock 对所有的节点上锁
func (rl *RedLock) Lock(ctx context.Context, lop LockOptions) error {
	// 遍历尝试对所有节点申请锁
	var successNum = 0 //成功上锁数量
	idxList := make([]int, 0)
	t1 := time.Now()
	var wg sync.WaitGroup
	for idx, lockClient := range rl.redisClients {
		// 并发对所有节点上锁
		lockClient := lockClient
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := lockClient.UnitTryLock(ctx, lop)
			if err == nil {
				// 如果上锁成功，计数++
				successNum++
				idxList = append(idxList, idx)
			}
		}(idx)
	}
	wg.Wait()
	t2 := time.Since(t1)
	//fmt.Println(t2)
	//fmt.Println(idxList)
	//fmt.Printf("%d:------------------", successNum)
	if len(idxList) != successNum {
		fmt.Printf("%d=====%d\n", len(idxList), successNum)
		panic(ErrDelLock)
	}
	// 需要保证过半节点申请成功并且剩余时间足够完成任务或者续租才行
	if successNum > len(rl.redisClients)/2 && t2 < ALL_NODE_TIMEOUT {
		go rl.startWatchDog(idxList, lop, ctx)
		return nil
	}
	//否则表示获取失败，需要对已经上锁的解锁
	for _, num := range idxList {
		err := rl.redisClients[num].UnitUnLock(ctx, lop)
		if err != nil {
			panic(err)
		}
	}
	idxList = nil
	return ErrGetLockFailed
}

// UnLock 解锁所有已上锁的节点
func (rl *RedLock) UnLock(ctx context.Context, lop LockOptions) error {
	var ok = false
	for _, lockClient := range rl.redisClients {
		err := lockClient.UnitUnLock(ctx, lop)
		if err != nil {
			ok = true
		}
	}
	close(rl.watchDog)
	if ok {
		return ErrDelLock
	}
	return nil
}

// UnitTryLock 尝试对某个节点申请锁
func (rlc *redisLockClient) UnitTryLock(ctx context.Context, lops LockOptions) error {
	result, err := rlc.rdb.SetNX(ctx, lops.Key, lops.Token, lops.Expiration).Result()
	//申请成功
	if err == nil && result {
		return nil
	}
	//申请失败
	if err != nil {
		return err
	}
	return ErrGetLockFailed
}

// UnitUnLock 对某个节点解锁
func (rlc *redisLockClient) UnitUnLock(ctx context.Context, lops LockOptions) error {
	result, err := rlc.rdb.Eval(ctx, UNLOCK_SCRIPT, []string{lops.Key}, lops.Token).Result()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrDelLock
	}
	return nil
}
