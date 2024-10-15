package parser

import "encoding/json"

type CreateGameRequest struct {
	Player         string
	MaxPlayerCount uint8
	TotalRounds    uint8
}

func ParseCreateGameRequest(data []byte) (*CreateGameRequest, error) {
	gameRequest := &CreateGameRequest{}
	err := json.Unmarshal(data, gameRequest)
	return gameRequest, err
}
