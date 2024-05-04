// services/gameManagerService.go
package services

import (
	"fmt"
	"sync"

	"github.com/sgatu/chezz-back/game"
	"github.com/sgatu/chezz-back/models"
)

type MoveMessage struct {
	ErrorsChannel chan error
	Move          string
	Who           int64
}

type Observer interface {
	UpdatesChannel() chan string
	ErrorsChannel() chan error
}

type GameManagerService struct {
	liveGameStates map[int64]*LiveGameState
	gameRepository models.GameRepository
	gameStatesLock sync.Mutex
}

func NewGameManagerService(gameRepository models.GameRepository) *GameManagerService {
	return &GameManagerService{
		liveGameStates: make(map[int64]*LiveGameState),
		gameRepository: gameRepository,
	}
}

func (s *GameManagerService) GetLiveGameState(gameId int64, requiresUpdate bool) (*LiveGameState, error) {
	if s.liveGameStates[gameId] == nil {
		gameEntity, err := s.gameRepository.GetGame(gameId)
		if err != nil {
			return nil, err
		}
		s.gameStatesLock.Lock()
		defer s.gameStatesLock.Unlock()
		s.liveGameStates[gameId] = &LiveGameState{
			game:              gameEntity,
			chCommandsChannel: make(chan MoveMessage, 10),
			observers:         make([]chan *game.MoveResult, 0),
			gameManager:       s,
		}
		s.liveGameStates[gameId].startAwaitingMoves()
	} else if requiresUpdate {
		gameEntity, err := s.gameRepository.GetGame(gameId)
		if err != nil {
			return nil, err
		}
		s.liveGameStates[gameId].game = gameEntity
	}
	return s.liveGameStates[gameId], nil
}

func (s *GameManagerService) removeLiveGameState(gameId int64) {
	s.gameStatesLock.Lock()
	defer s.gameStatesLock.Unlock()
	delete(s.liveGameStates, gameId)
}

type LiveGameState struct {
	chCommandsChannel chan MoveMessage
	game              *models.Game
	gameManager       *GameManagerService
	observers         []chan *game.MoveResult
	observersMutex    sync.Mutex
}

func (lgs *LiveGameState) AddObserver(observerCh chan *game.MoveResult) {
	lgs.observersMutex.Lock()
	defer lgs.observersMutex.Unlock()
	lgs.observers = append(lgs.observers, observerCh)
}

func (lgs *LiveGameState) RemoveObserver(observerCh chan *game.MoveResult) {
	lgs.observersMutex.Lock()
	defer lgs.observersMutex.Unlock()
	for i, observer := range lgs.observers {
		if observer == observerCh {
			lgs.observers = append(lgs.observers[:i], lgs.observers[i+1:]...)
			break
		}
	}
	if len(lgs.observers) == 0 {
		close(lgs.chCommandsChannel)
		fmt.Printf("before removing the livestate %+v, %+v\n", lgs.gameManager, lgs.game.Id())
		lgs.gameManager.removeLiveGameState(lgs.game.Id())
	}
}

func (lgs *LiveGameState) ExecuteMove(move MoveMessage) {
	lgs.chCommandsChannel <- move
}

func (lgs *LiveGameState) startAwaitingMoves() {
	go func() {
		for move := range lgs.chCommandsChannel {
			result, err := lgs.game.UpdateGame(move.Who, move.Move)
			if err == nil {
				lgs.notifyMoveObservers(result)
				lgs.gameManager.gameRepository.SaveGame(lgs.game)
			} else {
				fmt.Println("Could not execute move due to ", err)
				if move.ErrorsChannel != nil {
					move.ErrorsChannel <- err
				}
			}
		}
	}()
}

func (lgs *LiveGameState) notifyMoveObservers(moveResult *game.MoveResult) {
	lgs.observersMutex.Lock()
	defer lgs.observersMutex.Unlock()
	for _, observer := range lgs.observers {
		observer <- moveResult
	}
}
