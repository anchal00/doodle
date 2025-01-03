package state

import (
	"errors"
	"fmt"
)

type StateStore interface {
    GetGameState(gameId string) (*GameState, error)
    SetGameState(gameId string, gs *GameState)
}

type InMemoryGameStateStore struct {
	store map[string]*GameState
}

func NewInMemoryGameStore() *InMemoryGameStateStore {
	return &InMemoryGameStateStore{store: make(map[string]*GameState)}
}

func (i InMemoryGameStateStore) GetGameState(gameId string) (*GameState, error) {
	state, exists := i.store[gameId]
	if !exists {
		return nil, errors.New(fmt.Sprintf("No state found for this game Id %s", gameId))
	}
	return state, nil
}

func (i InMemoryGameStateStore) SetGameState(gameId string, state *GameState) {
	i.store[gameId] = state
}

