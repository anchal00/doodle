package server

import (
	"bytes"
	"doodle/db"
	dbMock "doodle/db/mocks"
	"doodle/logger"
	"doodle/parser"
	"doodle/state"
	stateStoreMock "doodle/state/mocks"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type GameServerTestSuite struct {
	suite.Suite
	dbMock    *dbMock.Repository
	stateMock *stateStoreMock.StateStore
	server    *httptest.Server
}

func (suite *GameServerTestSuite) SetupTest() {
	suite.dbMock = dbMock.NewRepository(suite.T())
	suite.stateMock = stateStoreMock.NewStateStore(suite.T())
	gs := CreateMockGameServer(suite.T(), suite.dbMock, suite.stateMock)
	suite.server = httptest.NewServer(gs.Router)
}

func (suite *GameServerTestSuite) TearDownTest() {
	suite.server.Close()
}

func TestGameServerSuite(t *testing.T) {
	suite.Run(t, new(GameServerTestSuite))
}

func CreateMockGameServer(t *testing.T, db *dbMock.Repository, stateStore *stateStoreMock.StateStore) *GameServer {
	router := mux.NewRouter().PathPrefix(HTTP_API_V1_PREFIX).Subrouter()
	gs := &GameServer{
		Db:          db,
		Logger:      logger.New("server_test_logger"),
		port:        "9999",
		wssUpgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		Router:      router,
		GameState:   stateStore,
	}
    gs.setupRoutes()
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
			suite.dbMock.On("CreateNewGame", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			suite.dbMock.On("GetGamePlayers", mock.Anything).Return([]db.Player{}, nil)
			if tc.expectedStatusCode == http.StatusCreated {
				suite.stateMock.On("SetGameState", mock.Anything, mock.Anything).Return(nil)
				mockGameObject := db.Game{
					PlayerCount: 1,
				}
				suite.dbMock.On("GetGameById", mock.Anything).Return(&mockGameObject)
			}
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
			assertCookieValid(suite, resp)
		})
	}
}

func assertCookieValid(suite *GameServerTestSuite, response *http.Response) {
	cookies := response.Cookies()
	suite.NotNil(cookies, "Cookie not found")
	suite.Equal(1, len(cookies), "Too many Cookies found")
	authCookie := cookies[0]
	suite.Equal(fmt.Sprintf("%s/connect", HTTP_API_V1_PREFIX), authCookie.Path, "Cookie path does not match")
	suite.Equal("session-token", authCookie.Name, "Cookie name mismatch")
	suite.True(authCookie.HttpOnly, "Cookie.HttpOnly is expected to be true")
	suite.NotEmpty(authCookie.Value, "Cookie Auth token is empty")
	timeToExpiry := authCookie.Expires
	expectedTime := time.Now().Add(time.Hour)
	suite.True(timeAlmostEqual(expectedTime, timeToExpiry, time.Minute*2), "Cookie expiry time invalid")
}

func timeAlmostEqual(date1, date2 time.Time, allowedDelta time.Duration) bool {
	durationDiff := date1.Sub(date2).Abs()
	return durationDiff <= allowedDelta
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
	mockGameObject := db.Game{
		PlayerCount: 1,
	}
	suite.dbMock.On("GetGameById", mock.Anything).Return(&mockGameObject)
	suite.dbMock.On("CreateNewGame", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	suite.dbMock.On("GetGamePlayers", mock.Anything).Return([]db.Player{}, nil)
	suite.stateMock.On("SetGameState", mock.Anything, mock.Anything).Return(nil)
	createGameRequestBody, err := json.Marshal(createGameRequest)
	suite.Nil(err, "Failed to create CreateGame request body")
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(createGameRequestBody))
	suite.Nil(err, "Failed to execute CreateGame api call")
	suite.Equal(http.StatusCreated, resp.StatusCode, "Failed to create new game")
	assertCookieValid(suite, resp)
}

func (suite *GameServerTestSuite) TestStartGame() {
	tests := []struct {
		description        string
		isAdmin            bool
        expectedStatusCode int
	}{
		{"Test admin can start game", true, http.StatusOK},
		{"Test non-admin can't start game", false, http.StatusForbidden},
	}
    for i, test := range tests {
        suite.Run(test.description, func() {
            mockGameObject := db.Game{
                GameId:       "xxxxxx",
            }
            mockPlayerObject := db.Player{
                Name:      "Player1",
                GameId:    mockGameObject.GameId,
                IsAdmin:   test.isAdmin,
                AuthToken: fmt.Sprintf("dummy-token-%d", i),
            }
            suite.dbMock.On("GetGamePlayerByToken", mockGameObject.GameId, mockPlayerObject.AuthToken).Return(&mockPlayerObject)
            suite.dbMock.On("GetGameById", mockGameObject.GameId).Return(&mockGameObject)
            suite.dbMock.On("GetGamePlayers", mock.Anything).Return([]db.Player{mockPlayerObject}, nil)
            fakeGameState := state.InitGameState(mockGameObject.GameId, suite.dbMock)
            suite.stateMock.On("GetGameState", mock.Anything).Return(fakeGameState, nil)
            url := suite.server.URL + HTTP_API_V1_PREFIX + fmt.Sprintf("/game/%s/start", mockGameObject.GameId)
            header := http.Header{}
            header.Add("Cookie", fmt.Sprintf("session-token=%s", mockPlayerObject.AuthToken))
            req, err := http.NewRequest("POST", url, nil)
            suite.Nil(err, "Failed to prepare StartGame request")
            req.Header = header
            suite.Equal(state.CREATED, fakeGameState.GetState())
            response, err := http.DefaultClient.Do(req)
            suite.Nil(err, "Failed to send StartGame request")
            suite.Equal(test.expectedStatusCode, response.StatusCode)
            if test.expectedStatusCode == http.StatusOK { suite.Equal(state.STARTED, fakeGameState.GetState()) }
            if test.expectedStatusCode == http.StatusForbidden { suite.Equal(state.CREATED, fakeGameState.GetState()) }
        })
    }

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
	suite.dbMock.On("GetGamePlayers", mock.Anything).Return([]db.Player{}, nil)
	suite.dbMock.On("AddPlayerToGame", mockGameObject.GameId, joiningPlayerName, mock.Anything).Return(nil)
	suite.stateMock.On("GetGameState", mock.Anything).Return(state.InitGameState(mockGameObject.GameId, suite.dbMock), nil)
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
	assertCookieValid(suite, resp)
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
	suite.Equal(0, len(resp.Cookies()))
}

func (suite *GameServerTestSuite) TestPlayerJoin() {
	mockGameObject := db.Game{
		GameId: "xxxxxx",
	}
	mockPlayerObject := db.Player{
		Name:      "Player1",
		GameId:    mockGameObject.GameId,
		IsAdmin:   true,
		AuthToken: "dummy-token",
	}
	suite.dbMock.On("GetGamePlayerByToken", mockGameObject.GameId, mockPlayerObject.AuthToken).Return(&mockPlayerObject)
	suite.dbMock.On("GetGameById", mockGameObject.GameId).Return(&mockGameObject)
	suite.dbMock.On("GetGamePlayers", mock.Anything).Return([]db.Player{}, nil)
	suite.stateMock.On("GetGameState", mock.Anything).Return(state.InitGameState(mockGameObject.GameId, suite.dbMock), nil)
	url := suite.server.URL + HTTP_API_V1_PREFIX + fmt.Sprintf("/connect/game/%s", mockGameObject.GameId)
	url = strings.ReplaceAll(url, "http:", "ws:")
	header := http.Header{}
	header.Add("Cookie", "session-token=dummy-token")
	_, _, err := websocket.DefaultDialer.Dial(url, header)
	suite.Nil(err, "Failed to establish websocket connection")
	// body := parser.GamePlayerInput{
	// 	Xcoord: 120,
	// 	Ycoord: 120,
	// }
	// err = conn.WriteJSON(body)
	// suite.Nil(err, "Failed to send payload on the websocket connection")
	// err = conn.Close()
	// suite.Nil(err, "Failed to close websocket connection")
}

func (suite *GameServerTestSuite) TestBadPlayerInput() {

}
