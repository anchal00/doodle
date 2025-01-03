package state

import (
	"doodle/db"
	"github.com/gorilla/websocket"
)

type GameState struct {
	turnQueue   []string
	gameId      string
	connections map[string]*websocket.Conn
	db          *db.Repository
}

func InitGameState(gameId string, database *db.Repository) *GameState {
	return &GameState{
		turnQueue:   []string{},
		gameId:      gameId,
		connections: make(map[string]*websocket.Conn),
		db:          database,
	}
}

func HandleInput(input []byte) {
}

func (gstate *GameState) broadcast(data interface{}) {}

func (gstate *GameState) Refresh() {}

func (gstate *GameState) AddConnection(player string, conn *websocket.Conn) {
	gstate.connections[player] = conn
}
func (gstate *GameState) RemoveConnection(player string) {
	delete(gstate.connections, player)
}
