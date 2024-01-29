package handlers_messages

import "github.com/sgatu/chezz-back/models"

type GameStatusMessage struct {
	BlackPlayer int64  `json:"blackPlayer"`
	WhitePlayer int64  `json:"whitePlayer"`
	GameId      int64  `json:"gameId"`
	Board       []byte `json:"board"`
}

func GameStatusFromGameModel(g *models.Game) (*GameStatusMessage, error) {
	gs, err := g.GameState().Serialize()
	if err != nil {
		return nil, err
	}
	return &GameStatusMessage{
		BlackPlayer: g.BlackPlayer(),
		WhitePlayer: g.WhitePlayer(),
		GameId:      g.Id(),
		Board:       gs,
	}, nil
}
