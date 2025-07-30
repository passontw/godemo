package main

import (
	"fmt"

	"go.uber.org/fx"
)

func main() {
	app := fx.New()
	app.Run()
	fmt.Println("使用 FX 框架")
}
