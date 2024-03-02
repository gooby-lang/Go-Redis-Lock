package main

import (
	"Go-Redis/RedisTool"
	"context"
	"fmt"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/redis/go-redis/v9"
	"strconv"
	"sync"
	"time"
)

var testTimeout = 200 * time.Millisecond

func main() {
	test4()
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
			time.Sleep(200 * time.Millisecond)
			fmt.Printf("Gorouter %d FINISHED IT !!!!!!!!!!!!!\n", id)
			defer rm.Unlock(ctx)
		}(i)
		//time.Sleep(100 * time.Millisecond)
	}
	wg.Wait()
}
func test2() {
	var ch = make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ch:
				fmt.Println("OVER!!!!")
				return
			}
		}
	}()
	time.Sleep(2000 * time.Millisecond)
	close(ch)
	wg.Wait()
}
func test3() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "123456",
	})
	t1 := time.Now()
	client.SetNX(context.Background(), "1", 1, 1000*time.Millisecond)
	t2 := time.Now()
	fmt.Printf("%v", t2.Sub(t1))
}
func test4() {
	redLock := RedisTool.NewRedLock(5)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			lop := RedisTool.LockOptions{
				Key:        "key",
				Token:      gofakeit.UUID(),
				Expiration: RedisTool.LOCK_TIMEOUT,
			}
			err := redLock.Lock(ctx, lop)
			if err != nil {
				fmt.Printf("routine %2d, %s !!!!\n", id, err)
				return
			}
			fmt.Printf("routine %2d:SUCCESS\n", id)
			time.Sleep(1000 * time.Millisecond)
			defer redLock.UnLock(ctx, lop)
			return
		}(i)
	}
	wg.Wait()
}
func test5() {
	var clients []*redis.Client
	for i := 0; i < 5; i++ {
		client := redis.NewClient(&redis.Options{
			Addr:     "localhost:" + strconv.Itoa(i+6379),
			Password: "123456",
		})
		clients = append(clients, client)
	}
	var wg sync.WaitGroup
	for _, client := range clients {
		client := client
		wg.Add(1)
		go func() {
			defer wg.Done()
			t1 := time.Now()
			client.SetNX(context.Background(), "key", gofakeit.UUID(), 100*time.Millisecond).Result()
			t2 := time.Since(t1)
			fmt.Printf("%v\n", t2)
		}()
	}
	wg.Wait()
}
