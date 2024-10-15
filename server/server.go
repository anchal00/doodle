package server

import (
	"doodle/db"
	"doodle/log"
	"doodle/parser"
	"doodle/utils"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

const HTTP_API_V1_PREFIX = "/api/v1"

type GameServer struct {
	Db          db.Repository
	Logger      *slog.Logger
	port        string
	wssUpgrader websocket.Upgrader
	router      *mux.Router
}

func (s *GameServer) Run() {
	s.Logger.Info("Starting server on port 9000")
	// TODO: Handle SIGINT and gracefully shutdown the server
	if err := http.ListenAndServe(":9000", s.router); err != nil {
		s.Logger.Error("Failed to start server on port 9000", slog.String("error", err.Error()))
		return
	}
}

func (s *GameServer) CreateNewGame(writer http.ResponseWriter, request *http.Request) {
	s.Logger.Info("Player is creating a new game")
	bodyReader := request.Body
	bytesRead, err := io.ReadAll(bodyReader)
	if err != nil {
		s.Logger.Error("Failed to read request body", slog.String("error", err.Error()))
		return
	}
	gameRequest, err := parser.ParseCreateGameRequest(bytesRead)
	if err != nil {
		s.Logger.Error("Failed to parse new game request", slog.String("error", err.Error()))
		return
	}
	gameId := utils.GetRandomGameId(6)
	// gameId could possibly be duplicate, fix this
	s.Logger.Info(fmt.Sprintf("game id %s", gameId))
	var MAX_ALLOWED_PLAYERS uint8 = 4
	max_players := min(MAX_ALLOWED_PLAYERS, gameRequest.MaxPlayerCount)
  err = s.Db.CreateNewGame(gameId, gameRequest.Player, max_players, gameRequest.TotalRounds)
  if err != nil {
    writer.WriteHeader(400)
    s.Logger.Error("CreateNewGame request failed") 
    return
  }
  s.Logger.Info("CreateNewGame request processed successfully") 
}

func (s *GameServer) JoinGame(writer http.ResponseWriter, request *http.Request) {
	s.Logger.Info("Player is joining a game")
}

func (s *GameServer) HandleClientPush(writer http.ResponseWriter, request *http.Request) {
	_, err := s.wssUpgrader.Upgrade(writer, request, nil)
	if err != nil {
		s.Logger.Error("Failed to process update", slog.String("error", err.Error()))
		return
	}
	s.Logger.Info("Player is sending a update")
}

func NewGameServer(port string) (*GameServer, error) {
	repo, err := db.SetupDB(os.Getenv("DOODLE_DB"))
	if err != nil {
		return nil, err
	}
	router := mux.NewRouter().PathPrefix(HTTP_API_V1_PREFIX).Subrouter()
	gs := &GameServer{
		Db:     repo,
		Logger: logger.NewLogger("api_server"),
		port:   port,
		wssUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		router: router,
	}
	router.HandleFunc("/game", gs.CreateNewGame).Methods("POST")
	router.HandleFunc("/game/{gameId:[a-z]+}", gs.JoinGame).Methods("POST")
	router.HandleFunc("/push", gs.HandleClientPush)
	return gs, nil
}
