package main

import (
	"Go-Redis/RedisTool"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"sync"
	"time"
)

var testTimeout = 200 * time.Millisecond

func main() {
	test()
}
func test1() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "123456",
	})
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			ctx := context.Background()
			defer wg.Done()
			result, err := rdb.SetNX(ctx, "key", "value", 1000*time.Millisecond).Result()
			if err != nil {
				fmt.Printf("ERROR\n")
				return
			}
			if !result {
				fmt.Printf("Gorouter %d get lock failed\n", id)
				return
			}
			fmt.Printf("Gorouter %d GET IT!!!!!!!!!!\n", id)
			time.Sleep(1000 * time.Millisecond)
			fmt.Printf("Gorouter %d FINISHED IT !!!!!!!!!!!!!\n", id)
			defer rdb.Del(ctx, "key")
		}(i)
		//time.Sleep(100 * time.Millisecond)
	}
	wg.Wait()
}
func test() {
	rm := RedisTool.InitMutex()
	var wg sync.WaitGroup
	numThreads := 10
	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			defer ctx.Done()
			//ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
			//defer cancel()
			err := rm.Lock(ctx)
			if err != nil {
				fmt.Printf("Goroutine %d failed to acquire lock: %v\n", id, err)
				return
			}
			fmt.Printf("Goroutine %d get lock!!!!!!!\n", id)
			time.Sleep(testTimeout)
			defer rm.Unlock(ctx)
		}(i)
		time.Sleep(100 * time.Millisecond)
	}
	wg.Wait()
}
