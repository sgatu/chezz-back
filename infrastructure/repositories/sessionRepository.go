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
	prefix    string
}

func (rsr *RedisSessionRepository) SaveSession(session *models.SessionStore) error {
	sessSerialized, err := json.Marshal(session)
	if err != nil {
		return err
	}
	rslt := rsr.redisConn.Set(rsr.ctx, rsr.getSessionKey(session.SessionId), sessSerialized, time.Hour*24*30)
	if rslt.Err() != nil {
		fmt.Printf("%+v\n", rslt.Err())
	}
	return nil
}

func (rsr *RedisSessionRepository) SetPrefix(prefix string) {
	rsr.prefix = prefix
}

func (rsr *RedisSessionRepository) getSessionKey(sessionId string) string {
	return rsr.prefix + "session." + sessionId
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
func (rsr *RedisSessionRepository) GetSession(sessionId string) (*models.SessionStore, error) {
	cmdResult := rsr.redisConn.Get(rsr.ctx, rsr.getSessionKey(sessionId))
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

func NewRedisSessionRepository(redisClient *redis.Client) *RedisSessionRepository {
	return &RedisSessionRepository{
		redisConn: redisClient,
		ctx:       context.Background(),
	}
}
