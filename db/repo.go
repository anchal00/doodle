package db

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
)

var schema = `CREATE TABLE IF NOT EXISTS games (
  game_id varchar(8) PRIMARY KEY,
  player_count int NOT NULL
);

CREATE TABLE IF NOT EXISTS players (
  name varchar(10) PRIMARY KEY,
  game_id varchar(8) REFERENCES games(game_id) ON DELETE CASCADE,
  is_active boolean NOT NULL,
  CONSTRAINT unq_player_id UNIQUE (name, game_id)
);

CREATE TABLE IF NOT EXISTS scores (
  game_id varchar(8) REFERENCES games(game_id) ON DELETE CASCADE,
  player varchar(10) REFERENCES players(name) ON DELETE CASCADE,
  score int NOT NULL
);`

type SqliteStore struct {
	Conn   *sqlx.DB
	Logger *slog.Logger
}

func (s *SqliteStore) SetupConnection(dbname string) error {
	sqlite_dbfile := dbname + ".db"
	db, err := sqlx.Connect("sqlite3", sqlite_dbfile)
	if err != nil {
		s.Logger.Error("Database setup failed")
		return err
	}
	s.Conn = db
	s.Conn.MustExec(schema)
	s.Logger.Info("Database setup complete")
	return nil
}

func (s *SqliteStore) CloseConnection() error {
	return s.Conn.Close()
}

func (s *SqliteStore) GetGameById(gameId string) Game {
	return Game{}
}

func (s *SqliteStore) GetGamePlayerByName(gameId, playerName string) Player {
	return Player{}
}

func (s *SqliteStore) CreateNewGame(gameId, player string) Game {
	return Game{}
}

func (s *SqliteStore) AddPlayerToGame(gameId, playerName string) (Game, Player) {
	return Game{}, Player{}
}

func (s *SqliteStore) UpdatePlayerScore(gameId, playerName string, scoreDelta uint8) {
}
