package handlers_messages

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type GameNotFoundMessage struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func PushGameNotFoundMessage(c *gin.Context, id string) {
	c.JSON(
		404,
		&GameNotFoundMessage{
			Message: fmt.Sprintf("Game with id '%s' was not found or could not be recovered.", id),
			Code:    404,
		},
	)
}
