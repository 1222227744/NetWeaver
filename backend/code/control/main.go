package main

import (
	"test2/router"
)

func main() {
	r := router.SetupRouter()
	r.Run(":8080") // 默认监听 0.0.0.0:8080
}
