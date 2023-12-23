package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/sgatu/chezz-back/services"
)

func SetupRoutes(engine *gin.Engine) {
	healthHandler := &HealthHandler{
		testService: &services.TestService{
			Value: "Test hola",
		},
	}
	engine.GET("/health", healthHandler.healthHandler)
	engine.GET("/test", healthHandler.testHandler)
	/*engine.GET("/test", func(ctx *gin.Context, tSvc *services.TestService) {

	})*/
}
