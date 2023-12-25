package handlers

import (
	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/sgatu/chezz-back/infrastructure/repositories"
)

func SetupRoutes(engine *gin.Engine) error {
	node, err := snowflake.NewNode(1)
	if err != nil {
		return err
	}
	healthHandler := &HealthHandler{
		gameRepository: repositories.NewRedisGameRepository("192.168.1.169", 6379, 5000),
		node:           node,
	}
	engine.GET("/health", healthHandler.healthHandler)
	engine.GET("/test", healthHandler.testHandler)
	return nil
	/*engine.GET("/test", func(ctx *gin.Context, tSvc *services.TestService) {

	})*/
}
