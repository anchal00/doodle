package state

import (
	"doodle/db"
	"doodle/logger"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-set/v3"
)

type state int

const (
    CREATED state = iota
    STARTED
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
    st           state
    msgQ         chan []byte
    log          logger.Logger
}

func InitGameState(gameId string, database db.Repository) *GameState {
    gs := &GameState{
		turnQueue:   []string{},
		gameId:      gameId,
		connections: make(map[string]*websocket.Conn),
		db:          database,
        players:     set.Set[string]{},
        mut:         &sync.Mutex{},
        st:          CREATED,
        msgQ:        make(chan []byte),
        log:         logger.New(fmt.Sprintf("GameStateLogger %s", gameId)),
	}
    gs.Refresh()
    return gs
}

func (g *GameState) GetState() state { return g.st }

func (g *GameState) StartGameLoop() {
    g.mut.Lock()
    g.st = STARTED
    g.mut.Unlock()
    for {
        go g.processMessages()
        for player := range g.players.Items() {
            if conx, exists := g.connections[player]; exists {
                go g.tryReadingPlayerInput(player, conx)
            } else {
                g.log.Error(fmt.Sprintf("No connection found for player %s", player), errors.New("Connection not found"));
                // TODO: should remove players ?
            }
        }
    }
}

func (g *GameState) broadcast(data interface{}) {}

func (g *GameState) processMessages() {
    for {
        message := <- g.msgQ
        jsonMap := make(map[string]interface{})
        err := json.Unmarshal(message, &jsonMap)
        if err != nil {
            g.log.Error("Failed to deserialize message", err)
            return
        }
        // TODO: Validate and process input
        g.log.Info("Message processed successfully")
    }
}

func (g *GameState) tryReadingPlayerInput(player string, c *websocket.Conn) {
	for {
		_, msg, err := c.ReadMessage()
		g.log.Info(fmt.Sprintf("Received data from player %s", player))
		if err != nil {
			g.log.Info(fmt.Sprintf("Player %s disconnected", player))
            c.Close()
			// Delete from Db
			// gs.RemoveConnection(player.Name)
			return
		}
        g.msgQ <- msg
	}
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
    g.players.Insert(player)
	g.connections[player] = conn
    g.log.Info(fmt.Sprintf("Connection for player %s added successfully", player))
}

func (g *GameState) RemoveConnection(player string) {
    g.players.Remove(player)
    g.db.DeletePlayer(g.gameId, player)
	delete(g.connections, player)
    g.log.Info(fmt.Sprintf("Connection for player %s removed successfully", player))
}
