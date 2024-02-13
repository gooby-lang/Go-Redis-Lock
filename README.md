基于Go语言的Redis分布式锁
> 可重入，乐观锁，悲观锁，续租策略
> 基于RedLock实现分布式锁



1、自定义一个Mutex互斥锁
```go
type RedisMutex struct {
	Key      string
	Value    string
	Duration time.Duration
	rdb      *redis.Client
}
```
2、利用SETNX完成上锁
```go
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
```
```go
func (rm *RedisMutex) BlockLock(ctx context.Context) error {
	//阻塞式上锁
	err := rm.Lock(ctx)
	for err != nil {
		err = rm.Lock(ctx)
	}
	return err
}
```
3、重写GET，SET方法
```go
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
```

```go
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
```

注意点：

1、在执行事务之前，先上锁，获取到锁之后再执行过程，执行结束后释放锁。

2、为了防止事务执行过程中发生错误导致一直占有锁不放开，所以需要设置过期时间，从而防止死锁。