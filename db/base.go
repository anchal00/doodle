package db

import (
	"doodle/logger"
	_ "github.com/mattn/go-sqlite3"
)

type Repository interface {
	SetupConnection(database string) error
	CloseConnection()
	GetGameById(gameId string) *Game
	GetGamePlayerByName(gameId, playerName string) Player
	GetGamePlayers(gameId string) ([]Player, error)
	GetGamePlayerByToken(gameId, token string) *Player
	CreateNewGame(gameId, player, token string, maxPlayers, totalRounds uint8) error
	AddPlayerToGame(gameId, playerName, token string) error
	DeletePlayer(gameId, player string)
	UpdatePlayerScore(gameId, playerName string, scoreDelta uint8) error
}

func SetupDB(dbName string) (Repository, error) {
	var repository Repository = &SqliteStore{
		Logger: logger.New("database"),
	}
	err := repository.SetupConnection(dbName)
	return repository, err
}
