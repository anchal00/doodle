package parser

import (
	"encoding/json"
)

type CreateGameRequest struct {
	Player         string `json:"player,omitempty"`
	MaxPlayerCount uint8  `json:"max_players,omitempty"`
	TotalRounds    uint8  `json:"total_rounds,omitempty"`
}

func ParseCreateGameRequest(data []byte) (*CreateGameRequest, error) {
	gameRequest := &CreateGameRequest{}
	err := json.Unmarshal(data, gameRequest)
	if err != nil {
		return nil, err
	}
	return gameRequest, err
}

type CreateGameResponse struct {
	GameId string `json:"game_id,omitempty"`
}

type JoinGameRequest struct {
	Player string `json:"player,omitempty"`
}

type JoinGameResponse struct {
	Token   string `json:"token,omitempty"`
	GameUrl string `json:"game_url,omitempty"`
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
	Player string `json:"player,omitempty"`
	GameId string `json:"game_id,omitempty"`
	Xcoord uint8  `json:"x_cord,omitempty"`
	Ycoord uint8  `json:"y_cord,omitempty"`
}
