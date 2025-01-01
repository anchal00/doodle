package state

import (
	"fmt"

	"github.com/gorilla/websocket"
)

type PlayerState struct {
	Connection *websocket.Conn
	Token      string
}

type InMemoryConnectionStore struct {
	conns map[string]map[string]*PlayerState
}

func NewConnectionStore() ConnectionStore {
	// TODO: Make it singleton
	return InMemoryConnectionStore{
		conns: make(map[string]map[string]*PlayerState),
	}
}

func (c InMemoryConnectionStore) GetSessionToken(player, game string) string {
	playerState, exists := c.conns[game][player]
	if !exists {
		return ""
	}
	return playerState.Token
}
func (c InMemoryConnectionStore) AddSessionToken(player, game, token string) {
	playerState, exists := c.conns[game][player]
	if !exists {
		c.conns[game][player] = &PlayerState{}
	}
	playerState.Token = token
}

func (c InMemoryConnectionStore) AddConnection(player, game string, wssConn *websocket.Conn) {
	playerState, exists := c.conns[game][player]
	if !exists {
		c.conns[game][player] = &PlayerState{}
	}
	playerState.Connection = wssConn
}

func (c InMemoryConnectionStore) GetConnection(player, game string) *websocket.Conn {
	gameConns, exists := c.conns[game]
	if !exists {
		return nil
	}
	playerConn, exists := gameConns[fmt.Sprintf("%s_connection", player)]
	if !exists {
		return nil
	}
	return playerConn.Connection
}

func (c InMemoryConnectionStore) RemoveConnection(player, game string) {
	gameConns, exists := c.conns[game]
	if !exists {
		return
	}
	delete(gameConns, fmt.Sprintf("%s_connection", player))
	delete(gameConns, fmt.Sprintf("%s_token", player))
}
