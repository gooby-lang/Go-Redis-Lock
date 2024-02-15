package RedisTool

import (
	"github.com/redis/go-redis/v9"
)

var rm *RedisMutex

func InitMutex() *RedisMutex {
	//初始化锁的设置，包括key-value，过期时间
	rm = NewRedisMutex("key", LOCK_TIMEOUT, redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "123456",
	}))
	return rm
}
