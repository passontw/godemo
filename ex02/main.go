package main

import (
	"fmt"
)

var isActive bool
var enabled, disabled = true, false

const (
	x = iota // x == 0
	y = iota // y == 1
	z = iota // z == 2
	w        // 常數宣告省略值時，預設和之前一個值的字面相同。這裡隱含的說 w = iota，因此 w == 3。其實上面 y 和 z 可同樣不用"= iota"
)

const (
	a       = iota //a=0
	b       = "B"
	c       = iota             //c=2
	d, e, f = iota, iota, iota //d=3,e=3,f=3
	g       = iota             //g = 4
)

func main() {
	var i string = "1"
	const PI = 3.1415926
	fmt.Printf(i + "\n")
	fmt.Println(PI)
	fmt.Println(isActive)
	fmt.Println(enabled)
	fmt.Println(disabled)
	s := "hello"
	c := []byte(s) // 將字串 s 轉換為 []byte 型別
	c[0] = 'c'
	s2 := string(c) // 再轉換回 string 型別
	fmt.Printf("%s\n", s2)

	fmt.Printf("===x===")
	fmt.Println(x)
	fmt.Printf("===w===")
	fmt.Println(w)
	fmt.Printf("===d===")
	fmt.Println(d)
	fmt.Printf("===e===")
	fmt.Println(e)
	fmt.Printf("===f===")
	fmt.Println(f)

	result := w == e
	fmt.Printf("===result===")
	fmt.Println(result)

	var arr [10]int
	arr[0] = 42                                     // 陣列下標是從 0 開始的
	arr[1] = 13                                     // 賦值操作
	fmt.Printf("The first element is %d\n", arr[0]) // 取得資料，回傳 42
	fmt.Printf("The last element is %d\n", arr[9])  //回傳未賦值的最後一個元素，預設回傳 0

	aa := [3]int{1, 2, 3} // 宣告了一個長度為 3 的 int 陣列

	bb := [10]int{1, 2, 3} // 宣告了一個長度為 10 的 int 陣列，其中前三個元素初始化為 1、2、3，其它預設為 0

	cc := [...]int{4, 5, 6} // 可以省略長度而採用`...`的方式，Go 會自動根據元素個數來計算長度
	fmt.Printf("===aa===")
	fmt.Println(aa)
	fmt.Printf("===bb===")
	fmt.Println(bb)
	fmt.Printf("===cc===")
	fmt.Println(cc)

	// 宣告了一個二維陣列，該陣列以兩個陣列作為元素，其中每個陣列中又有 4 個 int 型別的元素
	doubleArray := [2][4]int{[4]int{1, 2, 3, 4}, [4]int{5, 6, 7, 8}}

	// 上面的宣告可以簡化，直接忽略內部的型別
	easyArray := [2][4]int{{1, 2, 3, 4}, {5, 6, 7, 8}}
	fmt.Printf("===doubleArray===")
	fmt.Println(doubleArray)
	fmt.Printf("===easyArray===")
	fmt.Println(easyArray)
}
