package state

import (
	"doodle/db"
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
}

func InitGameState(gameId string, database db.Repository) *GameState {
    gs := &GameState{
		turnQueue:   []string{},
		gameId:      gameId,
		connections: make(map[string]*websocket.Conn),
		db:          database,
        players:     set.Set[string]{},
        mut: &sync.Mutex{},
	}
    gs.Refresh()
    return gs
}

func (g *GameState) broadcast(data interface{}) {}

func (g *GameState) HandleInput(input []byte) {}

func (g *GameState) Refresh() error {
    g.mut.Lock()
    defer g.mut.Unlock()
	game := g.db.GetGameById(g.gameId)
	g.currentRound = game.CurrentRound
	g.maxRounds = game.TotalRounds
	// Re-read all player info from DB
    players, err := g.db.GetGamePlayers(g.gameId)
    if err != nil { return err }
    for _, player := range players {
        name := player.Name
        if g.players.Contains(name) {continue}
        g.players.Insert(name)
        g.turnQueue = append(g.turnQueue, name)
    }
    return nil
}

func (g *GameState) AddConnection(player string, conn *websocket.Conn) {
	g.connections[player] = conn
}

func (g *GameState) RemoveConnection(player string) {
	delete(g.connections, player)
}
