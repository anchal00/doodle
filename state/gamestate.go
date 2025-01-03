package state

import (
	"doodle/db"
	"doodle/logger"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-set/v3"
)

type GameState struct {
	turnQueue    []string
	gameId       string
	connections  map[string]*websocket.Conn
	db           db.Repository
	currentRound uint8
	maxRounds    uint8
	players      set.Set[string]
    mut         *sync.Mutex
    log         logger.Logger
}

func InitGameState(gameId string, database db.Repository) *GameState {
    gs := &GameState{
		turnQueue:   []string{},
		gameId:      gameId,
		connections: make(map[string]*websocket.Conn),
		db:          database,
        players:     set.Set[string]{},
        mut: &sync.Mutex{},
        log: logger.New(fmt.Sprintf("GameStateLogger %s", gameId)),
	}
    gs.Refresh()
    return gs
}

func (g *GameState) broadcast(data interface{}) {}

func (g *GameState) HandleInput(input []byte) {
    jsonMap := make(map[string]interface{})
    err := json.Unmarshal(input, &jsonMap)
    if err != nil {
        g.log.Error("Failed to deserialize input", err)
        return
    }
    // TODO: Validate and process input
    g.log.Info("Input handled successfully")
}

func (g *GameState) Refresh() {
    g.mut.Lock()
    defer g.mut.Unlock()
	game := g.db.GetGameById(g.gameId)
	g.currentRound = game.CurrentRound
	g.maxRounds = game.TotalRounds
	// Re-read all player info from DB
    players, err := g.db.GetGamePlayers(g.gameId)
    if err != nil {
        g.log.Error("Failed to refresh game state", err)
        return
    }
    for _, player := range players {
        name := player.Name
        if g.players.Contains(name) {continue}
        g.players.Insert(name)
        g.turnQueue = append(g.turnQueue, name)
    }
    g.log.Info("Refreshed GameState successfully")
}

func (g *GameState) AddConnection(player string, conn *websocket.Conn) {
	g.connections[player] = conn
    g.log.Info(fmt.Sprintf("Connection for player %s added successfully", player))
}

func (g *GameState) RemoveConnection(player string) {
    g.db.DeletePlayer(g.gameId, player)
	delete(g.connections, player)
    g.log.Info(fmt.Sprintf("Connection for player %s removed successfully", player))
}
