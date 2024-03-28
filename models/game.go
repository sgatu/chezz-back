package models

import (
	"fmt"

	"github.com/bwmarrin/snowflake"
	"github.com/sgatu/chezz-back/game"
)

type Game struct {
	id          int64
	gs          *game.GameState
	whitePlayer int64
	blackPlayer int64
}

func (g *Game) Id() int64 {
	return g.id
}

func (g *Game) GameState() *game.GameState {
	return g.gs
}

func (g *Game) WhitePlayer() int64 {
	return g.whitePlayer
}

func (g *Game) BlackPlayer() int64 {
	return g.blackPlayer
}

func (g *Game) SetWhitePlayer(whitePlayer int64) error {
	if g.whitePlayer != 0 {
		return fmt.Errorf("white player already defined")
	}
	g.whitePlayer = whitePlayer
	return nil
}

func (g *Game) SetBlackPlayer(blackPlayer int64) error {
	if g.blackPlayer != 0 {
		return fmt.Errorf("black player already defined")
	}
	g.blackPlayer = blackPlayer
	return nil
}

func (g *Game) IsPlayer(playerId int64) bool {
	return g.blackPlayer == playerId || g.whitePlayer == playerId
}

func (g *Game) UpdateGame(playerId int64, uciMove string) error {
	if (g.gs.GetPlayerTurn() == game.BLACK_PLAYER && playerId != g.blackPlayer) ||
		(g.gs.GetPlayerTurn() == game.WHITE_PLAYER && playerId != g.whitePlayer) {
		return fmt.Errorf("not your turn")
	}
	return g.gs.UpdateGameState(uciMove)
}

func NewGame(node *snowflake.Node, userId int64, isBlackPlayer bool) *Game {
	whitePlayer := int64(0)
	blackPlayer := int64(0)
	if isBlackPlayer {
		blackPlayer = userId
	} else {
		whitePlayer = userId
	}
	return &Game{
		id:          node.Generate().Int64(),
		gs:          game.NewGameState(),
		whitePlayer: whitePlayer,
		blackPlayer: blackPlayer,
	}
}

func RecoverGameState(id int64, whitePlayer int64, blackPlayer int64, gameState *game.GameState) *Game {
	return &Game{
		id:          id,
		whitePlayer: whitePlayer,
		blackPlayer: blackPlayer,
		gs:          gameState,
	}
}

type GameRepository interface {
	GetGame(id int64) (*Game, error)
	SaveGame(game *Game) error
}
