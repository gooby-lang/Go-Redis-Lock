package RedisTool

import (
	"context"
	"errors"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/redis/go-redis/v9"
	"time"
)

// NewRedisMutex 新建一个redis锁
func NewRedisMutex(key string, duration time.Duration, rdb *redis.Client) *RedisMutex {
	return &RedisMutex{
		rdb: rdb,
		LockOptions: LockOptions{
			Key:        key,
			Token:      gofakeit.UUID(),
			Expiration: duration,
		},
		watchDog:       nil,
		renewInterval:  RENEW_INTERVAL,
		reentrantCount: 0,
	}
}

// Lock 阻塞式获取锁，支持续租
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
	if rm.watchDog == nil {
		rm.watchDog = make(chan struct{})
	}
	// 设定计时器
	ticker := time.NewTicker(rm.renewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// 每过一段时间就续租
			result, err := rm.rdb.Expire(ctx, rm.Key, rm.Expiration).Result()
			if err != nil || !result {
				//续租失败
				panic(ErrLockRent)
			}
		case <-rm.watchDog:
			// 已经解锁
			return nil
		}
	}
}

// TryLock 尝试获取一次锁
func (rm *RedisMutex) TryLock(ctx context.Context) error {
	//尝试获取锁
	result, err := rm.rdb.SetNX(ctx, rm.Key, rm.Token, rm.Expiration).Result()
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
	// 释放锁
	result, err := rm.rdb.Eval(ctx, UNLOCK_SCRIPT, []string{rm.Key}, rm.Token).Result()
	// 释放锁之后关闭watch dog
	if _, ok := <-rm.watchDog; !ok {
		close(rm.watchDog)
	}
	if err != nil || result == 0 {
		return ErrDelLock
	}
	return nil
}
