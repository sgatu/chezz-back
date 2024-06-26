package handlers

import (
	"fmt"
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
	if err != nil || id < 1 {
		handlers_messages.PushGameNotFoundMessage(c, idParam)
		return
	}
	session, err := GetCurrentSession(c)
	if err != nil {
		handlers_messages.PushGameNotFoundMessage(c, idParam)
	}
	// userId := session.UserId
	game, err := gh.gameRepository.GetGame(id)
	if err != nil || game == nil { //||
		// (game.BlackPlayer() != userId && game.WhitePlayer() != userId) {

		handlers_messages.PushGameNotFoundMessage(c, idParam)
		return
	}
	gameStatus, err := handlers_messages.GameStatusFromGameModel(game, session)
	if err != nil {
		handlers_messages.PushGameNotFoundMessage(c, idParam)
		return
	}
	c.JSON(200, gameStatus)
}

func (gh *GameHandler) createNewGame(c *gin.Context) {
	session, err := GetCurrentSession(c)
	// this should not happen
	if err != nil {
		c.JSON(401, handlers_messages.NewUnknownSessionError())
		return
	}
	isBlackQuery := c.Query("is_black")
	isBlackPlayer := isBlackQuery == "true" || isBlackQuery == "1" || isBlackQuery == "yes"
	fmt.Printf("Creating game as %+v \n", session)
	game := models.NewGame(gh.node, session.UserId, isBlackPlayer)
	gh.gameRepository.SaveGame(game)
	c.JSON(http.StatusCreated, struct {
		Message string `json:"message"`
		GameId  string `json:"game_id"`
	}{Message: "Game created", GameId: fmt.Sprint(game.Id())})
}
