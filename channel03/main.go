package main

import (
	"fmt"
	"sync"
)

func main() {
	ch := make(chan int)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		value := <-ch
		fmt.Println(value)
	}()

	ch <- 1
	wg.Wait()
}
