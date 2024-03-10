package handlers

import (
	"fmt"
	"os"
	"strconv"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/sgatu/chezz-back/infrastructure/repositories"
	"github.com/sgatu/chezz-back/middleware"
	"github.com/sgatu/chezz-back/models"
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

func SetupMiddlewares(engine *gin.Engine, node *snowflake.Node, redisClient *redis.Client) {
	sessionRedisRepo := repositories.NewRedisSessionRepository(redisClient)
	sessionManager := middleware.SessionManager{SessionRepository: sessionRedisRepo, Node: node}
	engine.Use(sessionManager.ManageSession())
}

func GetContextValue[T any](c *gin.Context, key string) (T, error) {
	result, ok := c.Get(key)
	if !ok {
		return *new(T), fmt.Errorf("key '%s' not found in context", key)
	}
	casted, ok := result.(T)
	if !ok {
		return *new(T), fmt.Errorf("key '%s' in context does not have type %T", key, result)
	}
	return casted, nil
}

func RefreshSession(c *gin.Context) error {
	session, err := GetContextValue[*models.SessionStore](c, "session")
	if err != nil {
		return err
	}
	// refresh the session
	session_mgr, err := GetContextValue[models.SessionRepository](c, "session_mgr")
	if err != nil {
		return err
	}
	session_updated, err := session_mgr.GetSession(session.SessionId)
	if err != nil {
		return err
	}
	c.Set("session", session_updated)
	return nil
}

func GetCurrentSession(c *gin.Context) (*models.SessionStore, error) {
	session, err := GetContextValue[*models.SessionStore](c, "session")
	if err != nil {
		return nil, err
	}
	return session, nil
}

func SetupRoutes(engine *gin.Engine) error {
	if err := godotenv.Load(); err != nil {
		return err
	}

	node, err := snowflake.NewNode(1)
	if err != nil {
		return err
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", getEnvDefault("REDIS_HOST", "localhost"), getEnvDefaultInt("REDIS_PORT", 6379)),
	})

	SetupMiddlewares(engine, node, redisClient)

	gameRedisRepo := repositories.NewRedisGameRepository(redisClient)

	healthHandler := &HealthHandler{
		gameRepository: gameRedisRepo,
		node:           node,
	}

	gameHandler := &GameHandler{
		gameRepository: gameRedisRepo,
		node:           node,
	}
	playHandler := &PlayHandler{
		gameRepository: gameRedisRepo,
	}
	// routes
	engine.GET("/health", healthHandler.healthHandler)
	engine.GET("/test", healthHandler.testHandler)
	engine.GET("/game/:id", gameHandler.getGame)
	engine.POST("/game", gameHandler.createNewGame)
	engine.GET("/play/:id", playHandler.Play)
	return nil
}
