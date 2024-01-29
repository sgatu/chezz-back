package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sgatu/chezz-back/models"
)

type RedisSessionRepository struct {
	redisConn *redis.Client
	ctx       context.Context
}

func (rsr *RedisSessionRepository) SaveSession(session *models.SessionStore) error {
	sessSerialized, err := json.Marshal(session)
	if err != nil {
		return err
	}
	rslt := rsr.redisConn.Set(rsr.ctx, session.SessionId, sessSerialized, time.Hour*24*30)
	if rslt.Err() != nil {
		fmt.Printf("%+v\n", rslt.Err())
	}
	return nil
}

// GetSession retrieves a session from the RedisSessionRepository by its session ID.
//
// Parameters:
//
//	session_id - The session ID string.
//
// Returns:
//
//	*models.SessionStore - The retrieved session store object.
//	error - An error if the session retrieval fails.
func (rsr *RedisSessionRepository) GetSession(session_id string) (*models.SessionStore, error) {
	cmdResult := rsr.redisConn.Get(rsr.ctx, session_id)
	if cmdResult.Err() != nil {
		return nil, cmdResult.Err()
	}
	var session models.SessionStore
	err := json.Unmarshal([]byte(cmdResult.Val()), &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func NewRedisSessionRepository(redisClient *redis.Client) models.SessionRepository {
	return &RedisSessionRepository{
		redisConn: redisClient,
		ctx:       context.Background(),
	}
}
