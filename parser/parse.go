package parser

import (
	"encoding/json"
	"fmt"
)

type CreateGameRequest struct {
	Player         string
	MaxPlayerCount uint8
	TotalRounds    uint8
}

func ParseCreateGameRequest(data []byte) (*CreateGameRequest, error) {
	gameRequest := &CreateGameRequest{}
	err := json.Unmarshal(data, gameRequest)
	if err != nil {
		return nil, err
	}
	return gameRequest, err
}

type JoinGameRequest struct {
	GameId string
	Player string
}

func ParseJoinGameRequest(data []byte) (*JoinGameRequest, error) {
	request := &JoinGameRequest{}
	err := json.Unmarshal(data, request)
	if err != nil {
		return nil, err
	}
	return request, err
}

type GamePlayerInput struct {
	Player string
	GameId string
	Xcoord uint8
	Ycoord uint8
}
