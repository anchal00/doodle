package state

import "github.com/gorilla/websocket"

type ConnectionStore interface {
	AddSessionToken(player, gameId, token string)
	GetSessionToken(player, gameId string) string
	AddConnection(player, gameId string, conn *websocket.Conn)
	RemoveConnection(player, gameId string)
	GetConnection(player, gameId string) *websocket.Conn
}
