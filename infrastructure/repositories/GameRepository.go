package repositories

import (
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/sgatu/chezz-back/models"
)

/*
*

	type GameRepository interface {
		getGame(id int64)
		saveGame(game *Game)
	}
*/
type RedisGameRepository struct {
	redisConn *redis.Client
}

func NewRedisGameRepository(host string, port int) models.GameRepository {
	return &RedisGameRepository{
		redisConn: redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", host, port),
			Password: "",
		}),
	}
}

func (rgr *RedisGameRepository) GetGame(id int64) *models.Game {
	//rgr.redisConn.Get()
	return nil
}

func (rgr *RedisGameRepository) SaveGame(g *models.Game) error {
	return nil
}
