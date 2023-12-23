package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/sgatu/chezz-back/services"
)

type HealthHandler struct {
	testService *services.TestService
}

func (hh *HealthHandler) healthHandler(c *gin.Context) {
	c.String(200, "ok")
}
func (hh *HealthHandler) testHandler(c *gin.Context) {
	c.String(200, hh.testService.Value)
}
