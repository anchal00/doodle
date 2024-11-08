package server

import "github.com/gorilla/websocket"

type ConnectionStore struct {
	conns map[string]map[string]*websocket.Conn
}

func NewConnectionStore() *ConnectionStore {
	// TODO: Make it singleton
	return &ConnectionStore{
		conns: make(map[string]map[string]*websocket.Conn),
	}
}

func (c *ConnectionStore) AddConnection(player, game string, wssConn *websocket.Conn) {
	c.conns[game][player] = wssConn
}

func (c *ConnectionStore) GetConnection(player, game string) *websocket.Conn {
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

func (c *ConnectionStore) RemoveConnection(player, game string) {
	gameConns, exists := c.conns[game]
	if !exists {
		return
	}
	delete(gameConns, player)
}
