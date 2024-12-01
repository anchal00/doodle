package server

import (
	"doodle/db"
	"doodle/logger"
	"doodle/parser"
	"doodle/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

const HTTP_API_V1_PREFIX = "/api/v1"
const MAX_ALLOWED_PLAYERS = 5
const MAX_ALLOWED_ROUNDS = 5

type GameServer struct {
	Db          db.Repository
	Logger      logger.Logger
	port        string
	wssUpgrader websocket.Upgrader
	Router      *mux.Router
	ConnStore   ConnectionStore
}

func (s *GameServer) UpgradeToWebsocket(writer http.ResponseWriter, request *http.Request) *websocket.Conn {
	conn, err := s.wssUpgrader.Upgrade(writer, request, nil)
	if err != nil {
		s.Logger.Error("Failed to upgrade to WS connection", err)
		return nil
	}
	return conn
}

func (s *GameServer) ReadRequestBody(request *http.Request) ([]byte, error) {
	bodyReader := request.Body
	bytesRead, err := io.ReadAll(bodyReader)
	if err != nil {
		s.Logger.Error("Failed to read request body", err)
		return nil, err
	}
	return bytesRead, nil
}

func (s *GameServer) Run() {
	s.Logger.Info(fmt.Sprintf("Starting server on port %s", s.port))
	sigtermHandler := make(chan os.Signal, 1)
	signal.Notify(sigtermHandler, os.Interrupt)
	go func() {
		<-sigtermHandler
		s.Shutdown()
		os.Exit(0)
	}()
	if err := http.ListenAndServe(fmt.Sprintf(":%s", s.port), s.Router); err != nil {
		s.Logger.Error(fmt.Sprintf("Failed to start server on port %s", s.port), err)
		return
	}
}

func (s *GameServer) Shutdown() {
	s.Logger.Info("Shutting down server....")
	s.Db.CloseConnection()
	s.Logger.Info("Goodbye !")
}

func isValidNewGameRequest(gameRequest parser.CreateGameRequest) bool {
	return len(gameRequest.Player) != 0
}

func (s *GameServer) sendResponse(writer http.ResponseWriter, responseBody []byte, status int) {
	writer.WriteHeader(status)
  if responseBody == nil {
    return
  }
	_, err := writer.Write(responseBody)
	if err != nil {
		s.Logger.Info("Failed to write response body")
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *GameServer) CreateNewGame(writer http.ResponseWriter, request *http.Request) {
	s.Logger.Info("Player is creating a new game")
	data, err := s.ReadRequestBody(request)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	gameRequest, err := parser.ParseCreateGameRequest(data)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		s.Logger.Error("Failed to parse new game request", err)
		return
	}
	gameRequest.Player = strings.TrimSpace(gameRequest.Player)

	if !isValidNewGameRequest(*gameRequest) {
		writer.WriteHeader(http.StatusBadRequest)
		s.Logger.Error("Bad game request", nil)
		return
	}
	gameId := utils.GetRandomGameId(6)
	// TODO: gameId could possibly be duplicate, fix this
	s.Logger.Info(fmt.Sprintf("game id %s", gameId))
	gameRequest.MaxPlayerCount = min(MAX_ALLOWED_PLAYERS, gameRequest.MaxPlayerCount)
	gameRequest.TotalRounds = min(MAX_ALLOWED_ROUNDS, gameRequest.TotalRounds)
	err = s.Db.CreateNewGame(gameId, gameRequest.Player, gameRequest.MaxPlayerCount, gameRequest.TotalRounds)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		s.Logger.Error("CreateNewGame request failed", err)
		return
	}
	// TODO: The player who created the game needs to connect via ws now
	// to be able to receieve updates of the others joining etc.
	respBody, err := json.Marshal(parser.CreateGameResponse{GameId: gameId})
	if err != nil {
    s.sendResponse(writer, nil, http.StatusInternalServerError)
		return
	}
  s.sendResponse(writer, respBody, http.StatusCreated)
}

func (s *GameServer) JoinGame(writer http.ResponseWriter, request *http.Request) {
	gameId := mux.Vars(request)["gameId"]
	s.Logger.Info(fmt.Sprintf("Player is joining game %s", gameId))
	data, err := s.ReadRequestBody(request)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	joinGameRequest, err := parser.ParseJoinGameRequest(data)
	if err != nil {
		s.Logger.Error("Failed to parse join game request", err)
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	game := s.Db.GetGameById(gameId)
	if game == nil {
		s.Logger.Error("Unrecognized game id", err)
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	if game.PlayerCount == game.MaxPlayers {
		s.Logger.Debug("Couldn't add player to the game, capacity full")
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := s.Db.AddPlayerToGame(gameId, joinGameRequest.Player); err != nil {
		s.Logger.Error("Failed to process join game request", err)
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	respBody, err := json.Marshal(parser.JoinGameResponse{
		// TODO: Create and handover valid auth token
		Token:   "dummy-token",
		GameUrl: fmt.Sprintf("http://localhost:%s%s%s", s.port, HTTP_API_V1_PREFIX, gameId),
	})
	if err != nil {
	  s.sendResponse(writer, nil, http.StatusInternalServerError)
		return
	}
  s.sendResponse(writer, respBody, http.StatusOK)
}

func (s *GameServer) HandlePlayerInput(writer http.ResponseWriter, request *http.Request) {
	gameId := mux.Vars(request)["gameId"]
	s.Logger.Info(fmt.Sprintf("Player is sending an update to game %s", gameId))
	// TODO: Authorize player
	wssConn := s.UpgradeToWebsocket(writer, request)
	joinGameRequest := &parser.JoinGameRequest{}
	if err := wssConn.ReadJSON(joinGameRequest); err != nil {
		s.Logger.Error("Cannot connect to game, bad payload", err)
		wssConn.Close()
		return
	}
	s.ConnStore.AddConnection(joinGameRequest.Player, gameId, wssConn)
	for {
		inputData := &parser.GamePlayerInput{}
		// TODO: Validate input
		if err := wssConn.ReadJSON(inputData); err != nil {
			s.Logger.Info("Player disconnected")
			// Delete from Db
			// Delete from ConnectionStore
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
		Logger: logger.New("api_server"),
		port:   port,
		wssUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		Router:    router,
		ConnStore: NewConnectionStore(),
	}
	router.HandleFunc("/game", gs.CreateNewGame).Methods("POST")
	router.HandleFunc("/game/{gameId:[a-z]+}", gs.JoinGame).Methods("POST")
	router.HandleFunc("/connect/game/{gameId:[a-z]+}", gs.HandlePlayerInput)
	return gs, nil
}
