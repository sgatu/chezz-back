package repositories

import (
	"context"
	"fmt"
	"time"

	"encoding/base64"
	"encoding/binary"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"github.com/sgatu/chezz-back/game"
	"github.com/sgatu/chezz-back/models"
)

type gameMarshalStruct struct {
	GameState   []byte
	WhitePlayer int64
	BlackPlayer int64
	GameId      int64
}

type RedisGameRepository struct {
	redisConn *redis.Client
	ctx       context.Context
}

func NewRedisGameRepository(redisClient *redis.Client) models.GameRepository {
	return &RedisGameRepository{
		redisConn: redisClient,
		ctx:       context.Background(),
	}
}

// GetGame retrieves a game from the RedisGameRepository.
//
// It takes an integer ID as a parameter and returns a pointer to a models.Game
// and an error.
func (rgr *RedisGameRepository) GetGame(id int64) (*models.Game, error) {
	cmdResult := rgr.redisConn.Get(rgr.ctx, rgr.getGameKey(id))
	result, err := cmdResult.Bytes()
	if err != nil {
		return nil, err
	}
	return rgr.recoverGame(result)
}

func (rgr *RedisGameRepository) SaveGame(g *models.Game) error {
	gameSerialized, err := rgr.serializeGame(g)
	if err != nil {
		return err
	}
	status := rgr.redisConn.Set(rgr.ctx, rgr.getGameKey(g.Id()), gameSerialized, time.Hour*time.Duration(24))
	if status.Err() != nil {
		return status.Err()
	}
	return nil
}

func (rgr *RedisGameRepository) getGameKey(id int64) string {
	rawId := [8]byte{}
	binary.LittleEndian.PutUint64(rawId[:], uint64(id))
	return "game.{" + base64.RawStdEncoding.EncodeToString(rawId[:]) + "}"
}

// serializeGame serializes a game object into a byte array.
//
// It takes a pointer to a models.Game object as a parameter.
// It returns a byte array and an error.
func (rgr *RedisGameRepository) serializeGame(g *models.Game) ([]byte, error) {
	if g == nil {
		return nil, fmt.Errorf("game is nil")
	}
	gameStatusSerialized, err := g.GameState().Serialize()
	if err != nil {
		return []byte{}, fmt.Errorf("cannot serialize game")
	}
	return json.Marshal(&gameMarshalStruct{
		GameState:   gameStatusSerialized,
		WhitePlayer: g.WhitePlayer(),
		BlackPlayer: g.BlackPlayer(),
		GameId:      g.Id(),
	})
}

// recoverGame recovers a game from serialized data.
//
// data: The serialized data of the game.
// Returns the recovered game and an error if there was any.
func (rgr *RedisGameRepository) recoverGame(data []byte) (*models.Game, error) {
	unmarshaledData := &gameMarshalStruct{}
	err := json.Unmarshal(data, unmarshaledData)
	if err != nil {
		return nil, err
	}
	gameState, err := game.FromSerialized(unmarshaledData.GameState)
	if err != nil {
		return nil, err
	}
	return models.RecoverGameState(
			unmarshaledData.GameId,
			unmarshaledData.WhitePlayer,
			unmarshaledData.BlackPlayer,
			gameState),
		nil
}
