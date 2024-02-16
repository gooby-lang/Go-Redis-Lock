package RedisTool

import (
	"context"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/redis/go-redis/v9"
)

func NewRedLock(num int) *RedLock {
	var redisClients []*redisLockClient
	for i := 0; i < num; i++ {
		client := &redisLockClient{
			rdb: redis.NewClient(&redis.Options{
				Addr:     "localhost:6379",
				Password: "123456",
			}),
		}
		redisClients = append(redisClients, client)
	}
	return &RedLock{
		redisClient: redisClients,
		lockOptions: lockOptions{
			key:        "key",
			token:      gofakeit.UUID(),
			expiration: LOCK_TIMEOUT,
		},
	}
}

// Lock 对所有的节点上锁
func (rl *RedLock) Lock(ctx context.Context) error {
	//遍历尝试对所有节点申请锁
	successNum := 0 //成功上锁数量
	var listNum []int
	for idx, lockClient := range rl.redisClient {
		err := lockClient.UnitTryLock(ctx, rl.lockOptions)
		if err == nil {
			//如果上锁成功，计数++
			successNum++
			listNum = append(listNum, idx)
		}
	}
	if successNum > len(rl.redisClient)/2 {
		//过半了才表示获取成功
		return nil
	}
	//否则表示获取失败，需要对已经上锁的解锁
	for _, num := range listNum {
		err := rl.redisClient[num].UnitUnLock(ctx, rl.lockOptions)
		if err != nil {
			return err
		}
	}
	return ErrGetLockFailed
}

// UnLock 解锁所有已上锁的节点
func (rl *RedLock) UnLock(ctx context.Context) error {
	for _, lockClient := range rl.redisClient {
		err := lockClient.UnitUnLock(ctx, rl.lockOptions)
		if err != nil {
			return err
		}
	}
	return nil
}

// UnitTryLock 尝试对某个节点申请锁
func (rlc *redisLockClient) UnitTryLock(ctx context.Context, lops lockOptions) error {
	result, err := rlc.rdb.SetNX(ctx, lops.key, lops.token, lops.expiration).Result()
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
func (rlc *redisLockClient) UnitUnLock(ctx context.Context, lops lockOptions) error {
	result, err := rlc.rdb.Eval(ctx, UNLOCK_SCRIPT, []string{lops.key}, lops.token).Result()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrDelLock
	}
	return nil
}
