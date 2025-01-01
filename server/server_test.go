package server

import (
	"bytes"
	"doodle/db"
	dbMock "doodle/db/mocks"
	"doodle/logger"
	"doodle/parser"
	connStoreMock "doodle/server/mocks"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type GameServerTestSuite struct {
	suite.Suite
	dbMock        *dbMock.Repository
	connStoreMock *connStoreMock.ConnectionStore
	server        *httptest.Server
}

func (suite *GameServerTestSuite) SetupTest() {
	suite.dbMock = dbMock.NewRepository(suite.T())
	suite.connStoreMock = connStoreMock.NewConnectionStore(suite.T())
	gs := CreateMockGameServer(suite.T(), suite.dbMock, suite.connStoreMock)
	suite.server = httptest.NewServer(gs.Router)
}

func (suite *GameServerTestSuite) TearDownTest() {
	suite.server.Close()
}

func TestGameServerSuite(t *testing.T) {
	suite.Run(t, new(GameServerTestSuite))
}

func CreateMockGameServer(t *testing.T, db *dbMock.Repository, connStore *connStoreMock.ConnectionStore) *GameServer {
	router := mux.NewRouter().PathPrefix(HTTP_API_V1_PREFIX).Subrouter()
	gs := &GameServer{
		Db:          db,
		Logger:      logger.New("test_logger"),
		port:        "9999",
		wssUpgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		Router:      router,
		GameState:   connStore,
	}
	router.HandleFunc("/game", gs.CreateNewGame).Methods("POST")
	router.HandleFunc("/game/{gameId:[a-z]+}", gs.JoinGame).Methods("POST")
	router.HandleFunc("/connect/game/{gameId:[a-z]+}", gs.HandlePlayerInput)
	return gs
}

func ReadResponseBody(response *http.Response) ([]byte, error) {
	bodyReader := response.Body
	bytesRead, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil, err
	}
	return bytesRead, nil
}

func (suite *GameServerTestSuite) TestCreateNewGame() {
	url := suite.server.URL + HTTP_API_V1_PREFIX + "/game"
	tests := []struct {
		description        string
		player             string
		max_player         int
		total_rounds       int
		expectedStatusCode int
	}{
		{"Test with valid new game request", "rookie", 5, 4, http.StatusCreated},
		{"Test with player name containing all whitespaces", "     ", 5, 4, http.StatusBadRequest},
		{"Test with empty player name", "", 5, 4, http.StatusBadRequest},
		{"Test with invalid rounds and player count", "rookie", -5, -4, http.StatusBadRequest},
		{"Test with invalid player count", "rookie", -5, 4, http.StatusBadRequest},
		{"Test with invalid round count", "rookie", 5, -4, http.StatusBadRequest},
	}
	for _, tc := range tests {
		suite.Run(tc.description, func() {
			suite.dbMock.On("CreateNewGame", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			createGameRequestBody, err := json.Marshal(map[string]any{
				"player":       tc.player,
				"max_players":  tc.max_player,
				"total_rounds": tc.total_rounds,
			})
			suite.Nil(err, "Failed to create CreateGame request body")
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(createGameRequestBody))
			suite.Nil(err, "Failed to execute CreateGame api call")
			suite.Equal(tc.expectedStatusCode, resp.StatusCode, "Failed to create new game")
			if tc.expectedStatusCode == 400 {
				return
			}
			respBody, err := ReadResponseBody(resp)
			suite.Nil(err, "Failed to read CreateGame response body")
			createGameResponse := parser.CreateGameResponse{}
			err = json.Unmarshal(respBody, &createGameResponse)
			suite.Nil(err, "Failed to deserialize CreateGame response body")
			gameId := createGameResponse.GameId
			suite.NotNil(gameId, "Failed to extract game id from CreateGame response body")
		})
	}
}

func (suite *GameServerTestSuite) TestCreateNewGameExceedingMaxAllowedPlayersAndRounds() {
	createGameRequest := parser.CreateGameRequest{
		Player: "dummy",
		// Even though MaxPlayerCount exceeds server.MAX_ALLOWED_PLAYERS
		// and server.MAX_ALLOWED_ROUNDS, Game should be created with MAX_ALLOWED_PLAYERS and MAX_ALLOWED_ROUNDS
		MaxPlayerCount: 10,
		TotalRounds:    10,
	}
	url := suite.server.URL + HTTP_API_V1_PREFIX + "/game"
	suite.dbMock.On("CreateNewGame", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	createGameRequestBody, err := json.Marshal(createGameRequest)
	suite.Nil(err, "Failed to create CreateGame request body")
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(createGameRequestBody))
	suite.Nil(err, "Failed to execute CreateGame api call")
	suite.Equal(http.StatusCreated, resp.StatusCode, "Failed to create new game")
}

func (suite *GameServerTestSuite) TestPlayersJoin() {
	mockGameObject := db.Game{
		GameId:       "xxxxxx",
		PlayerCount:  1,
		MaxPlayers:   4,
		CurrentRound: 0,
		TotalRounds:  4,
	}
	joiningPlayerName := "player1"
	suite.dbMock.On("GetGameById", mockGameObject.GameId).Return(&mockGameObject)
	suite.dbMock.On("AddPlayerToGame", mockGameObject.GameId, joiningPlayerName).Return(nil)
	url := suite.server.URL + HTTP_API_V1_PREFIX + fmt.Sprintf("/game/%s", mockGameObject.GameId)
	join_request, _ := json.Marshal(parser.JoinGameRequest{Player: joiningPlayerName})
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(join_request))
	suite.Nil(err, "Failed to send JoinGameRequest")
	suite.Equal(http.StatusOK, resp.StatusCode, "Unable to Join the requested game")
	respBody, err := ReadResponseBody(resp)
	suite.Nil(err, "Failed to read JoinGameResponse body")
	joinGameResponse := parser.JoinGameResponse{}
	err = json.Unmarshal(respBody, &joinGameResponse)
	suite.Nil(err, "Failed to deserialize JoinGameResponse body")
	suite.NotNil(joinGameResponse.GameUrl, "Failed to extract game url from JoinGameResponse body")
	suite.NotNil(joinGameResponse.Token, "Failed to extract auth token from JoinGameResponse body")
}

func (suite *GameServerTestSuite) TestPlayersJoinCapacityFull() {
	mockGameObject := db.Game{
		GameId:       "xxxxxx",
		PlayerCount:  4,
		MaxPlayers:   4,
		CurrentRound: 0,
		TotalRounds:  4,
	}
	joiningPlayerName := "player1"
	suite.dbMock.On("GetGameById", mockGameObject.GameId).Return(&mockGameObject)
	url := suite.server.URL + HTTP_API_V1_PREFIX + fmt.Sprintf("/game/%s", mockGameObject.GameId)
	join_request, _ := json.Marshal(parser.JoinGameRequest{Player: joiningPlayerName})
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(join_request))
	suite.Nil(err, "Failed to send JoinGameRequest")
	suite.Equal(http.StatusBadRequest, resp.StatusCode, "Unable to Join the requested game")
}

func (suite *GameServerTestSuite) TestPlayerInputs() {
}

func (suite *GameServerTestSuite) TestBadPlayerInput() {

}
