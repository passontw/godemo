package main

import (
	"fmt"
)

func main() {
	ch := make(chan int, 1)
	select {
	case ch <- 1:
		fmt.Println("發送成功")
	default:
		fmt.Println("發送失敗")
	}
	select {
	case value := <-ch:
		fmt.Println(value)
	default:
		fmt.Println("接收失敗")
	}
}
