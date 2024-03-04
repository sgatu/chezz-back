// services/gameManagerService.go
package services

import (
	"sync"

	"github.com/sgatu/chezz-back/models"
)

type MoveMessage struct {
	Move          string
	ErrorsChannel chan error
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

func (s *GameManagerService) GetLiveGameState(gameId int64) (*LiveGameState, error) {
	if s.liveGameStates[gameId] == nil {
		game, err := s.gameRepository.GetGame(gameId)
		if err != nil {
			return nil, err
		}
		s.gameStatesLock.Lock()
		defer s.gameStatesLock.Unlock()
		s.liveGameStates[gameId] = &LiveGameState{
			game:              game,
			chCommandsChannel: make(chan MoveMessage, 10),
			observers:         make([]chan string, 0),
		}
	}
	return s.liveGameStates[gameId], nil
}

func (s *GameManagerService) RemoveLiveGameState(gameId int64) {
	s.gameStatesLock.Lock()
	defer s.gameStatesLock.Unlock()
	delete(s.liveGameStates, gameId)
}

type LiveGameState struct {
	chCommandsChannel chan MoveMessage
	game              *models.Game
	observers         []chan string
	observersMutex    sync.Mutex
	gameManager       *GameManagerService
}

func (lgs *LiveGameState) AddObserver(observerCh chan string) {
	lgs.observersMutex.Lock()
	defer lgs.observersMutex.Unlock()
	lgs.observers = append(lgs.observers, observerCh)
}

func (lgs *LiveGameState) RemoveObserver(observerCh chan string) {
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
		lgs.gameManager.RemoveLiveGameState(lgs.game.Id())
	}
}
func (lgs *LiveGameState) ExecuteMove(move string) {
	lgs.chCommandsChannel <- move
}
func (lgs *LiveGameState) StartAwaitingMoves() {
	go func() {
		for move := range lgs.chCommandsChannel {
			err := lgs.game.GameState().UpdateGameState(move)
			if err == nil {
				lgs.NotifyMoveObservers(move)
			} else {

			}
		}
	}()
}

func (lgs *LiveGameState) NotifyMoveObservers(move string) {
	lgs.observersMutex.Lock()
	defer lgs.observersMutex.Unlock()
	for _, observer := range lgs.observers {
		observer <- move
	}
}
