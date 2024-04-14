package handlers_messages

import "github.com/sgatu/chezz-back/models"

type GameStatusMessage struct {
	MyRelation  string `json:"relation"`
	Board       []byte `json:"board"`
	BlackPlayer int64  `json:"blackPlayer"`
	WhitePlayer int64  `json:"whitePlayer"`
	GameId      int64  `json:"gameId"`
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
		BlackPlayer: g.BlackPlayer(),
		WhitePlayer: g.WhitePlayer(),
		GameId:      g.Id(),
		Board:       gs,
		MyRelation:  relation,
	}, nil
}
