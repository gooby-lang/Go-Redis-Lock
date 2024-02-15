package RedisTool

import (
	"context"
	"errors"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisMutex struct {
	rdb            *redis.Client //redis客户端
	resource       string        //锁的资源
	token          string        //锁的标识
	expiration     time.Duration //锁的过期时间
	watchDog       chan struct{} //看门狗
	renewInterval  time.Duration //续租间隔
	reentrantCount int           //可重入计数器
}

func NewRedisMutex(key string, duration time.Duration, rdb *redis.Client) *RedisMutex {
	return &RedisMutex{
		rdb:            rdb,
		resource:       key,
		token:          gofakeit.UUID(),
		expiration:     duration,
		watchDog:       make(chan struct{}),
		renewInterval:  RENEW_INTERVAL,
		reentrantCount: 0,
	}
}

// Lock 阻塞式获取锁，支持续租和可重入策略
func (rm *RedisMutex) Lock(ctx context.Context) error {
	//尝试获取一次锁
	err := rm.TryLock(ctx)
	//获取成功
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrGetLockFailed) {
		return err
	}
	//阻塞式获取锁
	//启动续租计时器
	ticker := time.NewTicker(rm.renewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			//超时或者取消
			return ErrGetLockTimeout
		case <-ticker.C:
			//触发续租事件
			err := rm.TryLock(ctx)
			if err == nil {
				return nil
			}
			if !errors.Is(err, ErrGetLockFailed) {
				return err
			}
		}
	}
}

// startWatchDog 利用看门狗机制实行续租功能
func (rm *RedisMutex) startWatchDog(ctx context.Context) error {
	rm.watchDog = make(chan struct{})
	ticker := time.NewTicker(rm.renewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			//续租
			result, err := rm.rdb.Expire(ctx, rm.resource, rm.expiration).Result()
			if err != nil || !result {
				//续租失败
				return nil
			}
		case <-rm.watchDog:
			//已经解锁
			return nil
		}
	}
}

// TryLock 尝试获取一次锁
func (rm *RedisMutex) TryLock(ctx context.Context) error {
	//检查可重入计数器，是否已经拥有锁
	//if rm.reentrantCount > 0 {
	//	rm.reentrantCount++
	//	return nil
	//}
	//尝试获取锁
	result, err := rm.rdb.SetNX(ctx, rm.resource, rm.token, rm.expiration).Result()
	//语句执行错误
	if err != nil {
		return err
	}
	//获取锁失败
	if !result {
		return ErrGetLockFailed
	}
	//获取锁成功，启动看门狗机制
	go rm.startWatchDog(ctx)
	return nil
}

// Unlock 释放锁
func (rm *RedisMutex) Unlock(ctx context.Context) error {
	//释放锁
	result, err := rm.rdb.Eval(ctx, UNLOCK_SCRIPT, []string{rm.resource}, rm.token).Result()
	close(rm.watchDog)
	if err != nil || result == 0 {
		return ErrDelLock
	}
	return nil
}
