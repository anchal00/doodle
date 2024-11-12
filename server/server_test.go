package server

import (
	"bytes"
	"doodle/parser"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	gs := setup()
	if gs == nil {
		os.Exit(1)
	}
	go gs.Run()
	exitCode := m.Run()
	tearDown(gs)
	os.Exit(exitCode)
}

func setup() *GameServer {
	err := godotenv.Load("../test.local")
	if err != nil {
		log.Print("Failed to load environment variables")
		return nil
	}
	port := os.Getenv("DOODLE_PORT")
	gs, err := NewGameServer(port)
	if err != nil {
		log.Printf("Failed to setup GameServer on port %s", port)
		return nil
	}
	return gs
}

func tearDown(gs *GameServer) {
	gs.Shutdown()
	// remove sqlite file
	err := os.Remove(fmt.Sprintf("%s.db", os.Getenv("DOODLE_DB")))
	if err != nil {
		log.Print("Failed to remove db file")
	}
}

func ApiCall(method, path string, requestBody io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("http://localhost:%s%s%s", os.Getenv("DOODLE_PORT"), HTTP_API_V1_PREFIX, path)
	req, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		return nil, err
	}
	log.Printf("Sending %s request to endpoint %s ", req.Method, req.URL)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("%s request to endpoint %s failed with error %s ", req.Method, req.URL, err.Error())
		return nil, err
	}
	log.Printf("%s request to endpoint %s returned status %d", req.Method, req.URL, resp.StatusCode)
	return resp, nil
}

func readResponseBody(response *http.Response) ([]byte, error) {
	bodyReader := response.Body
	bytesRead, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil, err
	}
	return bytesRead, nil
}

func TestGamePlayHappyFlow(t *testing.T) {
	// Create new game
	createGameRequestBody, err := json.Marshal(map[string]any{
		"player":       "rookie",
		"max_players":  5,
		"total_rounds": 4,
	})
	assert.Nil(t, err, "Failed to create CreateGame request body")
	resp, err := ApiCall("POST", "/game", bytes.NewBuffer(createGameRequestBody))
	assert.Nil(t, err, "Failed to execute CreateGame api call")
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create new game")
	respBody, err := readResponseBody(resp)
	assert.Nil(t, err, "Failed to read CreateGame response body")
	createGameResponse := &parser.CreateGameResponse{}
	err = json.Unmarshal(respBody, &createGameResponse)
	gameId := createGameResponse.GameId
	assert.Nil(t, err, "Failed to deserialize CreateGame response body")
	assert.NotNil(t, gameId, "Failed to extract game id from CreateGame response body")
	// Add players to game
	for player := 1; player <= 4; player += 1 {
		addPlayer1RequestBody, err := json.Marshal(parser.JoinGameRequest{Player: fmt.Sprintf("player%d", player)})
		assert.Nil(t, err, "Failed to create AddPlayer request body")
		resp, err = ApiCall("POST", fmt.Sprintf("/game/%s", gameId), bytes.NewBuffer(addPlayer1RequestBody))
		assert.Nil(t, err, "Failed to execute AddPlayer api call")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "Failed to add new player to the game")
	}
}
