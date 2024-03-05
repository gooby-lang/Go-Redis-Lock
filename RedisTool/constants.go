package RedisTool

import (
	"errors"
	"github.com/redis/go-redis/v9"
	"time"
)

type redisLockClient struct {
	token string        //唯一标识符
	rdb   *redis.Client //redis客户端
}

// LockOptions 锁的设置
type LockOptions struct {
	Key        string        //资源名称
	Token      string        //唯一标识符
	Expiration time.Duration //锁的过期时间
}

// RedLock 红锁
type RedLock struct {
	watchDog      chan struct{}      //看门狗机制
	renewInterval time.Duration      //续租间隔
	redisClients  []*redisLockClient //redis节点
}

// RedisMutex 互斥锁
type RedisMutex struct {
	LockOptions
	rdb            *redis.Client //redis客户端
	watchDog       chan struct{} //看门狗
	renewInterval  time.Duration //续租间隔
	reentrantCount int           //可重入计数器
}

var (
	ErrGetLockFailed  = errors.New("GET lock failed")   //获取锁失败错误
	ErrGetLockTimeout = errors.New("get lock time out") //超时错误
	ErrDelLock        = errors.New("lock delete error") //锁删除错误
	ErrLockRent       = errors.New("rent lock error")
)

const (
	ALL_NODE_TIMEOUT    = 2000 * time.Millisecond
	LOCK_TIMEOUT        = 600 * time.Millisecond //锁的有效时间
	RENEW_INTERVAL      = 500 * time.Millisecond //续租锁的时间间隔
	SINGLE_NODE_TIMEOUT = 400 * time.Millisecond //单个节点的交互时间上限
	UNLOCK_SCRIPT       = `if redis.call("get", KEYS[1]) == ARGV[1] then
							return redis.call("del", KEYS[1]);
						end;
						return 0;`  //解锁脚本
	EXPIRE_SCRIPT = `if redis.call("get", KEYS[1]) == KEYS[2] then
						return redis.call("expire", KEYS[1], ARGV[1])
					end
					return 0`
	DEFAULT_REDIS_ADR = "localhost:6379"
	DEFAULT_REDIS_PSW = "123456"
)
