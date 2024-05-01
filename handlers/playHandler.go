package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/sgatu/chezz-back/errors"
	"github.com/sgatu/chezz-back/game"
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
	gameEntity, err := ph.gameRepository.GetGame(id)
	if err != nil {
		handlers_messages.PushGameNotFoundMessage(c, idParam)
		return
	}
	requiresUpdate := false
	// set secondary player
	if gameEntity.BlackPlayer() != session.UserId && gameEntity.WhitePlayer() != session.UserId {
		if gameEntity.BlackPlayer() == 0 {
			requiresUpdate = true
			fmt.Printf("Setting black player to %+v\n", session.UserId)
			gameEntity.SetBlackPlayer(session.UserId)
		}
		if gameEntity.WhitePlayer() == 0 {
			requiresUpdate = true
			fmt.Printf("Setting white player to %+v\n", session.UserId)
			gameEntity.SetWhitePlayer(session.UserId)
		}
		ph.gameRepository.SaveGame(gameEntity)
	}
	liveGameState, err := ph.gameManager.GetLiveGameState(id, requiresUpdate)
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
		observeChan := make(chan *game.MoveResult)
		errorCh := make(chan error)
		liveGameState.AddObserver(observeChan)
		ticker := time.NewTicker(time.Second * 1)
		defer ticker.Stop()
		// this should be closed by writer... but, cross fingers
		defer close(errorCh)
		defer conn.Close()
		defer liveGameState.RemoveObserver(observeChan)
		aux := atomic.Uint32{}
		relation := "observer"
		if gameEntity.BlackPlayer() == session.UserId {
			relation = "black"
		} else if gameEntity.WhitePlayer() == session.UserId {
			relation = "white"
		}
		initMessage, err := json.Marshal(struct {
			Type     string `json:"type"`
			Relation string `json:"relation"`
		}{Type: "init", Relation: relation})
		if err != nil {
			return
		}
		wsutil.WriteServerMessage(conn, ws.OpText, []byte(initMessage))
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
					// fmt.Printf("No message received...%+v - %+v\n", err, message)
					continue
				}
				fmt.Println("New message", string(message[len(message)-1].Payload))
				liveGameState.ExecuteMove(services.MoveMessage{Move: string(message[len(message)-1].Payload), ErrorsChannel: errorCh, Who: playerId})
				newVal := aux.Add(1)
				if newVal == 100 {
					return
				}
			case move := <-observeChan:
				outputMessage, err := json.Marshal(struct {
					Type          string `json:"type"`
					Move          string `json:"uci"`
					CheckedPlayer int    `json:"checkedPlayer"`
					CheckMate     bool   `json:"isMate"`
				}{Type: "move", Move: move.Move, CheckedPlayer: int(move.CheckedPlayer), CheckMate: move.CheckMate})
				if err != nil {
					fmt.Println("Could not serialize movement")
					return
				}
				err = wsutil.WriteServerMessage(conn, ws.OpText, []byte(outputMessage))
				if err == nil {
					fmt.Println("Got movement, sent to player", move)
					//					return
				}
			case error := <-errorCh:
				if ferr, ok := error.(*errors.InvalidMoveError); ok {
					outputMessage, err := json.Marshal(struct {
						Type    string `json:"type"`
						Error   string `json:"error"`
						ErrCode string `json:"code"`
					}{Type: "error", Error: ferr.Message, ErrCode: ferr.ErrCode})
					if err != nil {
						fmt.Println("Could not serialize error")
						return
					}
					wsutil.WriteServerMessage(conn, ws.OpText, []byte(outputMessage))
				}
			}
		}
	}(session.UserId)
}
