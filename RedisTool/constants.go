package RedisTool

import (
	"errors"
	"time"
)

var (
	ErrGetLockFailed  = errors.New("GET lock failed")   //获取锁失败错误
	ErrGetLockTimeout = errors.New("get lock time out") //超时错误
	ErrDelLock        = errors.New("lock delete error") //锁删除错误
)

const (
	LOCK_TIMEOUT   = 200 * time.Millisecond //锁的有效时间
	RENEW_INTERVAL = 100 * time.Millisecond //续租锁的时间间隔
	UNLOCK_SCRIPT  = `if redis.call("get", KEYS[1], ARGV[1]) == 1 then
						return redis.call("del", KEYS[1]);
					end;
					return 0;`  //解锁脚本
)
