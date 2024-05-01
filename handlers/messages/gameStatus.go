package handlers_messages

import (
	"fmt"

	"github.com/sgatu/chezz-back/models"
)

type GameStatusMessage struct {
	MyRelation  string `json:"relation"`
	BlackPlayer string `json:"blackPlayer"`
	WhitePlayer string `json:"whitePlayer"`
	GameId      string `json:"gameId"`
	Board       []byte `json:"board"`
}

func GameStatusFromGameModel(g *models.Game, s *models.SessionStore) (*GameStatusMessage, error) {
	gs, err := g.GameState().Serialize()
	if err != nil {
		return nil, err
	}
	relation := "observer"
	if g.BlackPlayer() == s.UserId {
		relation = "black"
	} else if g.WhitePlayer() == s.UserId {
		relation = "white"
	}
	return &GameStatusMessage{
		BlackPlayer: fmt.Sprint(g.BlackPlayer()),
		WhitePlayer: fmt.Sprint(g.WhitePlayer()),
		GameId:      fmt.Sprint(g.Id()),
		Board:       gs,
		MyRelation:  relation,
	}, nil
}
