package db

import (
	logger "doodle/log"

	_ "github.com/mattn/go-sqlite3"
)

type Repository interface {
	SetupConnection(database string) error
	CloseConnection()
	GetGameById(gameId string) Game
	GetGamePlayerByName(gameId, playerName string) Player
	CreateNewGame(gameId, player string, maxPlayers, totalRounds uint8) error
	AddPlayerToGame(gameId, playerName string) error
	UpdatePlayerScore(gameId, playerName string, scoreDelta uint8) error
}

func SetupDB(dbName string) (Repository, error) {
	var repository Repository = &SqliteStore{
		Logger: logger.NewLogger("database"),
	}
	err := repository.SetupConnection(dbName)
	return repository, err
}
