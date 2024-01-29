package handlers

import (
	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/sgatu/chezz-back/middleware"
	"github.com/sgatu/chezz-back/models"
)

type HealthHandler struct {
	gameRepository models.GameRepository
	node           *snowflake.Node
}

func (hh *HealthHandler) healthHandler(c *gin.Context) {
	session, errSession := GetCurrentSession(c)
	sessionMgr, errSessionMgr := GetContextValue[*middleware.SessionManager](c, "session_mgr")
	if errSession == nil && errSessionMgr == nil {
		sessionMgr.SetSessionData(session, "test", "test")
		c.JSON(200, "writing test")
	} else {
		c.JSON(200, "session not found")
	}

}
func (hh *HealthHandler) testHandler(c *gin.Context) {
	c.String(200, "hola")
}
