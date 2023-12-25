package handlers

import (
	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/sgatu/chezz-back/models"
)

type HealthHandler struct {
	gameRepository models.GameRepository
	node           *snowflake.Node
}

func (hh *HealthHandler) healthHandler(c *gin.Context) {
	c.String(200, "ok")
}
func (hh *HealthHandler) testHandler(c *gin.Context) {
	game := models.NewGame(hh.node)
	hh.gameRepository.SaveGame(game)
	c.String(200, "hola")
}
