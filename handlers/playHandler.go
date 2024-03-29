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
	"github.com/sgatu/chezz-back/services"
)

type PlayHandler struct {
	gameRepository models.GameRepository
	gameManager    *services.GameManagerService
}

func (ph *PlayHandler) Play(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		handlers_messages.PushGameNotFoundMessage(c, idParam)
		return
	}
	session, err := GetCurrentSession(c)
	if err != nil {
		handlers_messages.PushGameNotFoundMessage(c, idParam)
		return
	}
	fmt.Printf("session found %+v\n", session)
	game, err := ph.gameRepository.GetGame(id)
	if err != nil {
		handlers_messages.PushGameNotFoundMessage(c, idParam)
		return
	}
	// set secondary player
	if game.BlackPlayer() != session.UserId && game.WhitePlayer() != session.UserId {
		if game.BlackPlayer() == 0 {
			fmt.Printf("Setting black player to %+v\n", session.UserId)
			game.SetBlackPlayer(session.UserId)
		}
		if game.WhitePlayer() == 0 {
			fmt.Printf("Setting white player to %+v\n", session.UserId)
			game.SetWhitePlayer(session.UserId)
		}
		ph.gameRepository.SaveGame(game)
	}
	liveGameState, err := ph.gameManager.GetLiveGameState(id)
	if err != nil {
		handlers_messages.PushGameNotFoundMessage(c, idParam)
	}
	conn, _, _, err := ws.UpgradeHTTP(c.Request, c.Writer)
	if err != nil {
		fmt.Println(err)
		c.JSON(500, struct{ err string }{err: err.Error()})
		return
	}

	go func(playerId int64) {
		observeChan := make(chan string)
		liveGameState.AddObserver(observeChan)
		ticker := time.NewTicker(time.Second * 1)
		defer ticker.Stop()
		defer conn.Close()
		defer liveGameState.RemoveObserver(observeChan)
		aux := atomic.Uint32{}
		for {
			select {
			case <-ticker.C:
				conn.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
				message, err := wsutil.ReadClientMessage(conn, nil)
				if err != nil || len(message[len(message)-1].Payload) == 0 {
					if err == nil && message[len(message)-1].OpCode == ws.OpClose {
						fmt.Printf("client closed connection\n")
						// client closed the connection
						return
					}
					fmt.Printf("No message received...%+v - %+v\n", err, message)
					continue
				}
				fmt.Println("New message", string(message[len(message)-1].Payload))
				liveGameState.ExecuteMove(services.MoveMessage{Move: string(message[len(message)-1].Payload), ErrorsChannel: nil, Who: playerId})
				newVal := aux.Add(1)
				if newVal == 100 {
					return
				}
			case move := <-observeChan:
				err := wsutil.WriteServerMessage(conn, ws.OpText, []byte(move))
				if err == nil {
					fmt.Println("Got movement, sent to player", move)
					//					return
				}
			}
		}
	}(session.UserId)
}
