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
	"os/signal"

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

func (s *GameServer) UpgradeToWebsocket(writer http.ResponseWriter, request *http.Request) *websocket.Conn {
	conn, err := s.wssUpgrader.Upgrade(writer, request, nil)
	if err != nil {
		s.Logger.Error("Failed to upgrade to WS connection", slog.String("error", err.Error()))
		return nil
	}
	return conn
}

func (s *GameServer) ReadRequestBody(request *http.Request) ([]byte, error) {
	bodyReader := request.Body
	bytesRead, err := io.ReadAll(bodyReader)
	if err != nil {
		s.Logger.Error("Failed to read request body", slog.String("error", err.Error()))
		return nil, err
	}
	return bytesRead, nil
}

func (s *GameServer) Run() {
	s.Logger.Info("Starting server on port 9000")
	// TODO: Handle SIGTERM and gracefully shutdown the server
	sigtermHandler := make(chan os.Signal, 1)
	signal.Notify(sigtermHandler, os.Interrupt)
	go func() {
		<-sigtermHandler
		s.Logger.Debug("Shutting down server....")
		os.Exit(0)
	}()
	if err := http.ListenAndServe(":9000", s.router); err != nil {
		s.Logger.Error("Failed to start server on port 9000", slog.String("error", err.Error()))
		return
	}
}

func (s *GameServer) CreateNewGame(writer http.ResponseWriter, request *http.Request) {
	s.Logger.Info("Player is creating a new game")
	data, err := s.ReadRequestBody(request)
	if err != nil {
		writer.WriteHeader(400)
		return
	}
	gameRequest, err := parser.ParseCreateGameRequest(data)
	if err != nil {
		writer.WriteHeader(400)
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
	// TODO: The player who created the game needs to connect via ws now
	// to be able to receieve updates of the others joining etc.
	s.Logger.Info("CreateNewGame request processed successfully")
	writer.WriteHeader(201)
}

func (s *GameServer) JoinGame(writer http.ResponseWriter, request *http.Request) {
	s.Logger.Info("Player is joining a game")
	data, err := s.ReadRequestBody(request)
	if err != nil {
		writer.WriteHeader(400)
		return
	}
	joinGameRequest, err := parser.ParseJoinGameRequest(data)
	if err != nil {
		s.Logger.Error("Failed to parse join game request", slog.String("error", err.Error()))
		writer.WriteHeader(400)
		return
	}
	if err := s.Db.AddPlayerToGame(joinGameRequest.GameId, joinGameRequest.Player); err != nil {
		s.Logger.Error("Failed to persist joinee's info")
		writer.WriteHeader(400)
		return
	}
	writer.WriteHeader(200)
}

func (s *GameServer) HandlePlayerInput(writer http.ResponseWriter, request *http.Request) {
	s.Logger.Info("Player is sending a update")
	// TODO: Authorize player
	wssConn := s.UpgradeToWebsocket(writer, request)
	for {
		inputData := &parser.GamePlayerInput{}
		// TODO: Validate input
		if err := wssConn.ReadJSON(inputData); err != nil {
			s.Logger.Info("Player disconnected")
			return
		}
		s.Logger.Info("Received data from a player")
	}
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
	router.HandleFunc("/push", gs.HandlePlayerInput)
	return gs, nil
}
