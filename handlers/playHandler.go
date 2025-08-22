package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
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
			gameEntity.SetBlackPlayer(session.UserId)
		}
		if gameEntity.WhitePlayer() == 0 {
			requiresUpdate = true
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

	getRelation := func(g *models.Game) string {
		relation := "observer"
		if g.BlackPlayer() == session.UserId {
			relation = "black"
		} else if g.WhitePlayer() == session.UserId {
			relation = "white"
		}
		return relation
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
		relation := getRelation(gameEntity)
		initMessage, err := json.Marshal(struct {
			Type     string `json:"type"`
			Relation string `json:"relation"`
		}{Type: "init", Relation: relation})
		if err != nil {
			return
		}
		wsutil.WriteServerMessage(conn, ws.OpText, []byte(initMessage))
		timePassed := 0
		for {
			select {
			case <-ticker.C:
				conn.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
				timePassed++
				message, err := wsutil.ReadClientMessage(conn, nil)
				var lastMessage *wsutil.Message = nil
				if len(message) > 0 {
					lastMessage = &message[len(message)-1]
				}
				if err == nil && lastMessage != nil && lastMessage.OpCode == ws.OpClose {
					// client closed connection
					return
				}
				if lastMessage != nil && lastMessage.OpCode == ws.OpPong {
					continue
				}
				// ping every 5 seconds
				if timePassed > 5 {
					timePassed = 0
					conn.Write(ws.CompiledPing)
				}
				if err != nil {
					// fmt.Println(err)
					continue
				}
				fmt.Println("lastMessage.OpCode = ", lastMessage.OpCode)
				liveGameState.ExecuteMove(services.MoveMessage{Move: string(lastMessage.Payload), ErrorsChannel: errorCh, Who: playerId})
			case move := <-observeChan:
				mateStatusStr := ""
				if move.MateStatus == game.STATUS_CHECKMATE {
					mateStatusStr = "#"
				}
				if move.MateStatus == game.STATUS_STALEMATE {
					mateStatusStr = "-"
				}
				outputMessage, err := json.Marshal(struct {
					Type             string `json:"type"`
					Move             string `json:"uci"`
					MateStatus       string `json:"mateStatus"`
					EnPassantCapture string `json:"enPassantCapture"`
					CheckedPlayer    int    `json:"checkedPlayer"`
				}{Type: "move", Move: move.Move, CheckedPlayer: int(move.CheckedPlayer), MateStatus: mateStatusStr, EnPassantCapture: move.EnPassantCapture})
				if err != nil {
					fmt.Println("Could not serialize movement")
					return
				}
				err = wsutil.WriteServerMessage(conn, ws.OpText, []byte(outputMessage))
				if err == nil {
					fmt.Printf("Got movement, sent to player %+v\n", move)
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
