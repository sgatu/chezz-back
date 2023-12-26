package handlers

import (
	"fmt"
	"os"
	"strconv"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sgatu/chezz-back/infrastructure/repositories"
)

func getEnvDefault(key string, defaultValue string) string {
	result := os.Getenv(key)
	if result == "" {
		return defaultValue
	}
	return result
}
func getEnvDefaultInt(key string, defaultValue int) int {
	result := getEnvDefault(key, fmt.Sprintf("%d", defaultValue))
	parsed, err := strconv.Atoi(result)
	if err != nil {
		return defaultValue
	}
	return parsed
}
func SetupRoutes(engine *gin.Engine) error {
	err := godotenv.Load()
	if err != nil {
		return err
	}
	node, err := snowflake.NewNode(1)
	if err != nil {
		return err
	}
	gameRedisRepo := repositories.NewRedisGameRepository(
		getEnvDefault("REDIS_HOST", "localhost"),
		getEnvDefaultInt("REDIS_PORT", 6379),
		5000,
	)
	healthHandler := &HealthHandler{
		gameRepository: gameRedisRepo,
		node:           node,
	}
	gameHandler := &GameHandler{gameRepository: gameRedisRepo}
	engine.GET("/health", healthHandler.healthHandler)
	engine.GET("/test", healthHandler.testHandler)
	engine.GET("/game/:id", gameHandler.getGame)
	engine.POST("/game", gameHandler.createNewGame)
	return nil
	/*engine.GET("/test", func(ctx *gin.Context, tSvc *services.TestService) {

	})*/
}
