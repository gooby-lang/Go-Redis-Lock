package main

import (
	"Go-Redis/RedisTool"
	"context"
	"fmt"
	"sync"
	"time"
)

var testTimeout = 200 * time.Millisecond

func main() {
	test1()
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
func test1() {
	rm := RedisTool.InitMutex()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			ctx, cancel := context.WithTimeout(context.Background(), 10000*time.Millisecond)
			defer wg.Done()
			defer cancel()
			err := rm.Lock(ctx)
			if err != nil {
				fmt.Printf("router %d FUCK %v\n", id, err)
				return
			}
			fmt.Printf("Gorouter %d GET IT!!!!!!!!!!\n", id)
			time.Sleep(300 * time.Millisecond)
			fmt.Printf("Gorouter %d FINISHED IT !!!!!!!!!!!!!\n", id)
			defer rm.Unlock(ctx)
		}(i)
		//time.Sleep(100 * time.Millisecond)
	}
	wg.Wait()
}
