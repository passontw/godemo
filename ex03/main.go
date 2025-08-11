package main

import (
	"fmt"
)

func computedValue() (output1 int) {
	return 5
}

func main() {
	x := 10
	if x > 10 {
		fmt.Println("x is greater than 10")
	} else {
		fmt.Println("x is less than 10")
	}

	// 計算取得值 x，然後根據 x 回傳的大小，判斷是否大於 10。
	if y := computedValue(); y > 10 {
		fmt.Println("y is greater than 10")
	} else {
		fmt.Println("y is less than 10")
	}

	sum := 0
	for index := 0; index < 10; index++ {
		sum += index
	}
	fmt.Println("sum is equal to ", sum)
}
