package main

import (
	"fmt"
	"sync"
)

func main() {
	ch := make(chan int, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ch <- 1
	}()
	wg.Wait()
	value := <-ch
	fmt.Println(value)
}
