package RedisTool

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisMutex struct {
	Key      string
	Value    string
	Duration time.Duration
	rdb      *redis.Client
}

func NewRedisMutex(key string, value string, duration time.Duration, rdb *redis.Client) *RedisMutex {
	return &RedisMutex{
		Key:      key,
		Value:    value,
		Duration: duration,
		rdb:      rdb,
	}
}

func (rm *RedisMutex) Lock(ctx context.Context) error {
	//非阻塞式上锁
	result, err := rm.rdb.SetNX(ctx, rm.Key, rm.Value, rm.Duration).Result()
	if err != nil {
		return err
	}
	if !result {
		return fmt.Errorf("Failed to get mutex %s\n", rm.Key)
	}
	return nil
}

func (rm *RedisMutex) Unlock(ctx context.Context) error {
	//释放锁
	result, err := rm.rdb.Del(ctx, rm.Key).Result()
	if err != nil {
		return err
	}
	if result == 0 {
		return fmt.Errorf("Failed to unlock mutex %s\n", rm.Key)
	}
	return nil
}

func (rm *RedisMutex) BlockLock(ctx context.Context) error {
	//阻塞式上锁
	err := rm.Lock(ctx)
	for err != nil {
		err = rm.Lock(ctx)
	}
	return err
}
