基于Go语言的Redis分布式锁

### 锁的初始化部分
```go
// lockOptions 锁的设置
type lockOptions struct {
    key        string        //资源名称
    token      string        //唯一标识符
    expiration time.Duration //锁的过期时间
}
// RedisMutex 互斥锁
type RedisMutex struct {
    lockOptions
    rdb            *redis.Client //redis客户端
    watchDog       chan struct{} //看门狗
    renewInterval  time.Duration //续租间隔
    reentrantCount int           //可重入计数器
}
// NewRedisMutex 新建一个redis锁
func NewRedisMutex(key string, duration time.Duration, rdb *redis.Client) *RedisMutex {
	return &RedisMutex{
		rdb: rdb,
		lockOptions: lockOptions{
			key:        key,
			token:      gofakeit.UUID(),
			expiration: duration,
		},
		watchDog:       make(chan struct{}),
		renewInterval:  RENEW_INTERVAL,
		reentrantCount: 0,
	}
}
```

#### 加锁
```go
// TryLock 尝试获取一次锁
func (rm *RedisMutex) TryLock(ctx context.Context) error {
	//尝试获取锁
	result, err := rm.rdb.SetNX(ctx, rm.key, rm.token, rm.expiration).Result()
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
```
```go
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
```

#### 解锁
```go
// Unlock 释放锁
func (rm *RedisMutex) Unlock(ctx context.Context) error {
	// 释放锁
	result, err := rm.rdb.Eval(ctx, UNLOCK_SCRIPT, []string{rm.key}, rm.token).Result()
	// 释放锁之后关闭watch dog
	close(rm.watchDog)
	if err != nil || result == 0 {
		return ErrDelLock
	}
	return nil
}
```

### RedLock部分

设有n个redis节点

1、每次尝试加锁时候，都会对n个节点同时申请加锁，同时记录开始时的时间戳。

2、如果过半节点加锁成功，且使用时间小于锁的有效时间时，则加锁成功，并且启动看门狗。

3、否则需要把加锁成功的节点进行解锁操作。


#### 加锁

```go

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
	fmt.Println(t2)
	fmt.Println(idxList)
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
```
```go
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
```