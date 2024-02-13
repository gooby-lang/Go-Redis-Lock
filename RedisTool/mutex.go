package RedisTool

import (
	"context"
	"errors"
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
	//阻塞式获取锁
	deadline := time.Now().Add(timeout)
	count := 0
	for err := rm.TryLock(ctx); err != nil; err = rm.TryLock(ctx) {
		count++
		if time.Now().After(deadline) {
			return ErrTimeout
		}
		if errors.Is(err, ErrLockFailed) {
			time.Sleep(timeInterval)
			continue
		}
		return err
	}
	//fmt.Printf("FUCK %d\n", count)
	return nil
}

func (rm *RedisMutex) TryLock(ctx context.Context) error {
	//获取一次锁
	result, err := rm.rdb.SetNX(ctx, rm.Key, rm.Value, rm.Duration).Result()
	if err != nil {
		return err
	}
	if !result {
		return ErrLockFailed
	}
	return nil
}

func (rm *RedisMutex) Unlock(ctx context.Context) error {
	//释放锁
	//script, err := os.ReadFile(scriptPath)
	//if err != nil {
	//	fmt.Printf("Error reading file: %v\n", err)
	//	fmt.Printf("Failed on Reading file\n")
	//	return nil
	//}
	script := `if redis.call("get", KEYS[1], ARGV[1]) == 1 then
    return redis.call("del", KEYS[1]);
end;
return 0;
				`
	result, err := rm.rdb.Eval(ctx, string(script), []string{rm.Key}, rm.Value).Result()
	if err != nil {
		return err
	}
	if result == int64(0) {
		return ErrDelLock
	}
	return nil
}
