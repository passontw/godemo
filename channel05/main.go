package main

import (
	"fmt"
	"sync"
	"time"
)

// 定義要傳遞的資料結構
type Task struct {
	ID   int
	Data string
}

// 處理函數
func processTask(task Task) {
	fmt.Printf("處理任務 %d: %s\n", task.ID, task.Data)
	time.Sleep(1 * time.Second) // 模擬處理時間
	fmt.Printf("任務 %d 完成\n", task.ID)
}

func main() {
	ch := make(chan Task, 100)
	var wg sync.WaitGroup

	// 啟動 worker goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for task := range ch { // 持續從 channel 接收任務
			processTask(task)
		}
	}()

	// 發送任務到 channel
	for i := 1; i <= 3; i++ {
		ch <- Task{
			ID:   i,
			Data: fmt.Sprintf("任務資料 %d", i),
		}
	}

	close(ch) // 關閉 channel，告訴 worker 沒有更多任務
	wg.Wait()
}
