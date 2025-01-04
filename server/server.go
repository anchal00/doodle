package server

import (
	crypto "crypto/rand"
	"doodle/db"
	"doodle/logger"
	"doodle/parser"
	"doodle/state"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

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
	GameState   state.StateStore
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
		s.sendResponse(writer, nil, http.StatusInternalServerError)
		return
	}
}

func (s *GameServer) attachSessionToken(writer http.ResponseWriter) (string, error) {
	token, err := createSessionToken()
	if err != nil {
		s.sendResponse(writer, nil, http.StatusInternalServerError)
		s.Logger.Error("CreateNewGame request failed: Unable to create session token", err)
		return "", err
	}
	expirationTime := time.Now().Add(60 * time.Minute)
	cookie := &http.Cookie{
		Name:     "session-token",
		Value:    token,
		HttpOnly: true,
		Secure:   false,
		Path:     fmt.Sprintf("%s/connect", HTTP_API_V1_PREFIX),
		SameSite: http.SameSiteStrictMode,
		Expires:  expirationTime,
	}
	http.SetCookie(writer, cookie)
	return token, nil
}

func (s *GameServer) CreateNewGame(writer http.ResponseWriter, request *http.Request) {
	s.Logger.Info("Player is creating a new game")
	data, err := s.ReadRequestBody(request)
	if err != nil {
		s.sendResponse(writer, nil, http.StatusBadRequest)
		return
	}
	gameRequest, err := parser.ParseCreateGameRequest(data)
	if err != nil {
		s.sendResponse(writer, nil, http.StatusBadRequest)
		s.Logger.Error("Failed to parse new game request", err)
		return
	}
	gameRequest.Player = strings.TrimSpace(gameRequest.Player)

	if !isValidNewGameRequest(*gameRequest) {
		s.sendResponse(writer, nil, http.StatusBadRequest)
		s.Logger.Error("Bad game request", nil)
		return
	}
	gameId := getRandomGameId(6)
	// TODO: gameId could possibly be duplicate, fix this
	gameRequest.MaxPlayerCount = min(MAX_ALLOWED_PLAYERS, gameRequest.MaxPlayerCount)
	gameRequest.TotalRounds = min(MAX_ALLOWED_ROUNDS, gameRequest.TotalRounds)
	authToken, err := s.attachSessionToken(writer)
	if err != nil {
		s.Logger.Error("CreateNewGame request failed", err)
		return
	}
	err = s.Db.CreateNewGame(gameId, gameRequest.Player, authToken, gameRequest.MaxPlayerCount, gameRequest.TotalRounds)
	if err != nil {
		s.sendResponse(writer, nil, http.StatusBadRequest)
		s.Logger.Error("CreateNewGame request failed", err)
		return
	}
	s.GameState.SetGameState(gameId, state.InitGameState(gameId, s.Db))
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
		s.sendResponse(writer, nil, http.StatusBadRequest)
		return
	}
	joinGameRequest, err := parser.ParseJoinGameRequest(data)
	if err != nil {
		s.Logger.Error("Failed to parse join game request", err)
		s.sendResponse(writer, nil, http.StatusBadRequest)
		return
	}
	game := s.Db.GetGameById(gameId)
	if game == nil {
		s.Logger.Error("Unrecognized game id", err)
		s.sendResponse(writer, nil, http.StatusBadRequest)
		return
	}
	if game.PlayerCount == game.MaxPlayers {
		s.Logger.Debug("Couldn't add player to the game, capacity full")
		s.sendResponse(writer, nil, http.StatusBadRequest)
		return
	}
	authToken, err := s.attachSessionToken(writer)
	if err != nil {
		s.Logger.Error("JoinGame request failed", err)
		return
	}
	if err := s.Db.AddPlayerToGame(gameId, joinGameRequest.Player, authToken); err != nil {
		s.Logger.Error("Failed to process join game request", err)
		s.sendResponse(writer, nil, http.StatusBadRequest)
		return
	}
	gs, err := s.GameState.GetGameState(gameId)
	if err != nil {
		s.Logger.Error("GameStateError", err)
		s.sendResponse(writer, nil, http.StatusBadRequest)
		return
	}
	gs.Refresh()
	respBody, err := json.Marshal(parser.JoinGameResponse{
		GameUrl: fmt.Sprintf("http://127.0.0.1:%s%s/%s", s.port, HTTP_API_V1_PREFIX, gameId),
	})
	if err != nil {
		s.sendResponse(writer, nil, http.StatusInternalServerError)
		return
	}
	s.sendResponse(writer, respBody, http.StatusOK)
}

func (s *GameServer) authorizePlayer(gameId string, request *http.Request) (*db.Player, error) {
	cookie, err := request.Cookie("session-token")
	if err != nil {
		return nil, err
	}
	if err := cookie.Valid(); err != nil {
		return nil, err
	}
	// TODO: Store auth token expiry time in db as well and verify token's validity here
	token := cookie.Value
	player := s.Db.GetGamePlayerByToken(gameId, token)
	return player, nil
}

func (s *GameServer) StartGame(writer http.ResponseWriter, request *http.Request) {
	gameId := mux.Vars(request)["gameId"]
	player, err := s.authorizePlayer(gameId, request)
	if err != nil {
		s.Logger.Error("Malformed cookie", err)
		s.sendResponse(writer, nil, http.StatusInternalServerError)
		return
	}
	if !player.IsAdmin {
		s.Logger.Error("Attempt to start the game from a Non-Admin player", err)
		s.sendResponse(writer, nil, http.StatusForbidden)
		return
	}
	gs, err := s.GameState.GetGameState(gameId)
	if err != nil {
		s.Logger.Error("GameStateError", err)
		s.sendResponse(writer, nil, http.StatusBadRequest)
		return
	}
	go gs.StartGameLoop()
	s.sendResponse(writer, nil, http.StatusOK)
}

func (s *GameServer) Connect(writer http.ResponseWriter, request *http.Request) {
	gameId := mux.Vars(request)["gameId"]
	s.Logger.Info(fmt.Sprintf("Player is sending an update to game %s", gameId))
	// Authorize player
	player, err := s.authorizePlayer(gameId, request)
	if err != nil {
		s.Logger.Error("Malformed cookie", err)
		s.sendResponse(writer, nil, http.StatusInternalServerError)
		return
	}
	wssConn := s.UpgradeToWebsocket(writer, request)
	gs, err := s.GameState.GetGameState(gameId)
	if err != nil {
		s.Logger.Error("GameStateError", err)
		s.sendResponse(writer, nil, http.StatusBadRequest)
		return
	}
	gs.AddConnection(player.Name, wssConn)
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
		GameState: state.NewInMemoryGameStore(),
	}
    gs.setupRoutes()
	return gs, nil
}

func (s *GameServer) setupRoutes() {
	s.Router.HandleFunc("/game", s.CreateNewGame).Methods("POST")
	s.Router.HandleFunc("/game/{gameId:[a-z]+}", s.JoinGame).Methods("POST")
	s.Router.HandleFunc("/game/{gameId:[a-z]+}/start", s.StartGame).Methods("POST")
	s.Router.HandleFunc("/connect/game/{gameId:[a-z]+}", s.Connect)
}

func createSessionToken() (string, error) {
	token := make([]byte, 32)
	_, err := crypto.Read(token)
	if err != nil {
		return "", err
	}
	// Convert bytes to a hex string
	return hex.EncodeToString(token), nil
}

func getRandomGameId(size int) string {
	r := make([]byte, size)
	for i := 0; i < size; i += 1 {
		offset := rand.Intn(26)
		r[i] = byte(97 + offset)
	}
	return string(r)
}
