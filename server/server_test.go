package server

import (
	"bytes"
	dbMock "doodle/db/mocks"
	"doodle/logger"
	"doodle/parser"
	connStoreMock "doodle/server/mocks"
	"encoding/json"
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
		ConnStore:   connStore,
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
			url := suite.server.URL + HTTP_API_V1_PREFIX
			createGameRequestBody, err := json.Marshal(map[string]any{
				"player":       tc.player,
				"max_players":  tc.max_player,
				"total_rounds": tc.total_rounds,
			})
			suite.Nil(err, "Failed to create CreateGame request body")
			resp, err := http.Post(url+"/game", "application/json", bytes.NewBuffer(createGameRequestBody))
			suite.Nil(err, "Failed to execute CreateGame api call")
			suite.Equal(tc.expectedStatusCode, resp.StatusCode, "Failed to create new game")
			if tc.expectedStatusCode == 400 {
				return
			}
			respBody, err := ReadResponseBody(resp)
			suite.Nil(err, "Failed to read CreateGame response body")
			createGameResponse := &parser.CreateGameResponse{}
			err = json.Unmarshal(respBody, &createGameResponse)
			gameId := createGameResponse.GameId
			suite.Nil(err, "Failed to deserialize CreateGame response body")
			suite.NotNil(gameId, "Failed to extract game id from CreateGame response body")
		})
	}
}

func (suite *GameServerTestSuite) TestPlayersJoin() {

}

func (suite *GameServerTestSuite) TestPlayersJoinCapacityFull() {

}

func (suite *GameServerTestSuite) TestPlayerInputs() {
}

func (suite *GameServerTestSuite) TestBadPlayerInput() {

}
