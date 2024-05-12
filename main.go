package main

import (
	"github.com/gin-gonic/gin"
	"github.com/sgatu/chezz-back/handlers"
)

func main() {
	router := gin.Default()
	handlers.SetupRoutes(router)
	router.Run(":8888")
}
