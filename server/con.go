package server

import "github.com/gorilla/websocket"

type ConnectionStore interface {
	AddConnection(player, gameId string, conn *websocket.Conn)
	RemoveConnection(player, gameId string)
	GetConnection(player, gameId string) *websocket.Conn
}

type InMemoryConnectionStore struct {
	conns map[string]map[string]*websocket.Conn
}

func NewConnectionStore() ConnectionStore {
	// TODO: Make it singleton
	return InMemoryConnectionStore{
		conns: make(map[string]map[string]*websocket.Conn),
	}
}

func (c InMemoryConnectionStore) AddConnection(player, game string, wssConn *websocket.Conn) {
	c.conns[game][player] = wssConn
}

func (c InMemoryConnectionStore) GetConnection(player, game string) *websocket.Conn {
	gameConns, exists := c.conns[game]
	if !exists {
		return nil
	}
	playerConn, exists := gameConns[player]
	if !exists {
		return nil
	}
	return playerConn
}

func (c InMemoryConnectionStore) RemoveConnection(player, game string) {
	gameConns, exists := c.conns[game]
	if !exists {
		return
	}
	delete(gameConns, player)
}
