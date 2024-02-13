package RedisTool

import (
	"errors"
	"time"
)

var ErrLockFailed = errors.New("lock failed")    //获取锁失败错误
var ErrTimeout = errors.New("lock get time out") //超时错误
var ErrDelLock = errors.New("lock delete error") //锁删除错误

var timeout = 100 * time.Millisecond     //获取锁超时时间
var timeInterval = 40 * time.Millisecond //获取锁的时间间隔
var lockTimeout = 200 * time.Millisecond //锁的有效时间
