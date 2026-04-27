package router

import (
	"test2/controller"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 把这里的路径从 "/ping" 改为 "/"
	r.GET("/", controller.Ping)

	return r
}
