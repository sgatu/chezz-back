package handlers

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	handlers_messages "github.com/sgatu/chezz-back/handlers/messages"
	"github.com/sgatu/chezz-back/models"
)

type PlayHandler struct {
	gameRepository models.GameRepository
}

func (ph *PlayHandler) Play(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		handlers_messages.PushGameNotFoundMessage(c, idParam)
		return
	}
	_, err = GetCurrentSession(c)
	if err != nil {
		handlers_messages.PushGameNotFoundMessage(c, idParam)
		return
	}
	//	userId := session.UserId
	_, err = ph.gameRepository.GetGame(id)
	if err != nil {
		handlers_messages.PushGameNotFoundMessage(c, idParam)
		return
	}

	conn, _, _, err := ws.UpgradeHTTP(c.Request, c.Writer)
	if err != nil {
		fmt.Println(err)
		c.JSON(500, struct{ err string }{err: err.Error()})
		return
	}
	go func() {
		ticker := time.NewTicker(time.Second * 1)
		defer ticker.Stop()
		defer conn.Close()
		aux := atomic.Uint32{}
		for range ticker.C {
			err := wsutil.WriteServerMessage(conn, ws.OpText, []byte("hola"))
			if err != nil {
				fmt.Println("Ticked... sending hola")
				return
			}
			conn.SetReadDeadline(time.Now().Add(time.Second * 2))
			message, err := wsutil.ReadClientMessage(conn, nil)
			if err == nil {
				fmt.Println(message)
			}
			newVal := aux.Add(1)
			if newVal == 25 {
				return
			}
		}
	}()
}
