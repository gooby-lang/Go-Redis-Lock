package RedisTool

import (
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

var rm *RedisMutex

func initMutex() {
	//初始化锁的设置，包括key-value，过期时间
	rm = NewRedisMutex("key", "value", 100, redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "123456",
	}))
}

func MutexBlockSet(ctx context.Context, key string, value any, duration time.Duration) (string, error) {
	//阻塞式的set
	err := rm.BlockLock(ctx)
	if err != nil {
		return "false", err
	}
	defer rm.Unlock(ctx)
	_, err = rm.rdb.Set(ctx, key, value, duration).Result()
	result, err := rm.rdb.Set(ctx, key, value, duration).Result()
	return result, err
}
func MutexSet(ctx context.Context, key string, value any, duration time.Duration) (string, error) {
	//非阻塞式的set
	err := rm.Lock(ctx)
	if err != nil {
		return "false", err
	}
	defer rm.Unlock(ctx)
	result, err := rm.rdb.Set(ctx, key, value, duration).Result()
	return result, err
}
func MutexBlockGet(ctx context.Context, key string) (any, error) {
	//阻塞式的get
	err := rm.BlockLock(ctx)
	if err != nil {
		return false, err
	}
	defer rm.Unlock(ctx)
	result, err := rm.rdb.Get(ctx, key).Result()
	return result, err
}
func MutexGet(ctx context.Context, key string) (any, error) {
	//非阻塞式的get
	err := rm.BlockLock(ctx)
	if err != nil {
		return false, err
	}
	defer rm.Unlock(ctx)
	result, err := rm.rdb.Get(ctx, key).Result()
	return result, err
}
