package main

import (
	"fmt"
	"sync"
	"time"
)

var mu sync.Mutex

func worker(id int) {
	fmt.Printf("Goroutine %d 嘗試獲取鎖\n", id)

	mu.Lock()

	fmt.Printf("Goroutine %d 獲得鎖，開始工作\n", id)

	time.Sleep(2 * time.Second)

	fmt.Printf("Goroutine %d 完成工作，釋放鎖\n", id)
	mu.Unlock()
}

func main() {
	wg := sync.WaitGroup{}
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			worker(id)
		}(i)
	}

	wg.Wait()
}
