package server

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

var wssUpgrader = websocket.Upgrader{
  CheckOrigin: func(r *http.Request) bool {return true},
}

type GameAPIServer struct {
	Logger *slog.Logger
}

func (s *GameAPIServer) CreateNewGame(writer http.ResponseWriter, request *http.Request) {
    s.Logger.Info("Player is creating a new game")
}

func (s *GameAPIServer) JoinGame(writer http.ResponseWriter, request *http.Request) {
    s.Logger.Info("Player is joining a game")
}

func (s *GameAPIServer) HandleClientPush(writer http.ResponseWriter, request *http.Request) {
    _, err := wssUpgrader.Upgrade(writer, request, nil)
    if err != nil {
        s.Logger.Error("Failed to process update", slog.String("error", err.Error()))
        return
    }
    s.Logger.Info("Player is sending a update")
}

