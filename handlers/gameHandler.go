package handlers

import (
	"net/http"
	"strconv"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	handlers_messages "github.com/sgatu/chezz-back/handlers/messages"
	"github.com/sgatu/chezz-back/models"
)

type GameHandler struct {
	gameRepository models.GameRepository
	node           *snowflake.Node
}

func (gh *GameHandler) getGame(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		handlers_messages.PushGameNotFoundMessage(c, idParam)
		return
	}
	game, err := gh.gameRepository.GetGame(id)
	if err != nil || game == nil {
		handlers_messages.PushGameNotFoundMessage(c, idParam)
		return
	}
	c.JSON(200, game)
}

func (gh *GameHandler) createNewGame(c *gin.Context) {
	game := models.NewGame(gh.node)
	gh.gameRepository.SaveGame(game)
	c.JSON(http.StatusCreated, struct {
		Message string `json:"message"`
		GameId  int64  `json:"game_id"`
	}{Message: "Game created", GameId: game.Id()})
}
