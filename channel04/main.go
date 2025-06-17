package main

import (
	"fmt"
	"time"
)

func main() {
	ch := make(chan int, 1)
	select {
	case result := <-ch:
		fmt.Println(result)
	case <-time.After(5 * time.Second):
		fmt.Println("操作超時")
	}

	ch <- 1
	select {
	case result := <-ch:
		fmt.Println(result)
	case <-time.After(5 * time.Second):
		fmt.Println("操作超時")
	}
}
