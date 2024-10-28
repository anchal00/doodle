package db

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
)

var schema = `CREATE TABLE IF NOT EXISTS games (
  game_id varchar(8) PRIMARY KEY,
  player_count int DEFAULT 1 NOT NULL,
  max_players int NOT NULL,
  current_round int DEFAULT 1 NOT NULL,
  total_rounds int NOT NULL
);

CREATE TABLE IF NOT EXISTS players (
  name varchar(10),
  game_id varchar(8) REFERENCES games(game_id) ON DELETE CASCADE,
  is_active boolean NOT NULL,
  PRIMARY KEY (name, game_id)
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

func (s *SqliteStore) CreateNewGame(gameId, player string, maxPlayers, totalRounds uint8) error {
	txn, err := s.Conn.Beginx()
	if err != nil {
		s.Logger.Error("Failed to create new game", slog.String("error", err.Error()))
		return err
	}
	createGameSQL := `INSERT INTO games(game_id, max_players, total_rounds) VALUES(?, ?, ?);`
	_, err = txn.Exec(createGameSQL, gameId, maxPlayers, totalRounds)
	if err != nil {
		s.Logger.Error("Failed to create new game", slog.String("error", err.Error()))
		txn.Rollback()
		return err
	}
	s.Logger.Info("Game created successfully")
	insertPlayerSQL := `INSERT INTO players VALUES(?, ?, ?);`
	_, err = txn.Exec(insertPlayerSQL, player, gameId, true)
	if err != nil {
		s.Logger.Error("Failed to save player", slog.String("error", err.Error()))
		errRoll := txn.Rollback()
		if errRoll != nil {
			s.Logger.Error("Failed to rollback", slog.String("error", errRoll.Error()))
			return errRoll
		}
		return err
	}
	txn.Commit()
	return nil
}

func (s *SqliteStore) AddPlayerToGame(gameId, playerName string) error {
	return nil
}

func (s *SqliteStore) UpdatePlayerScore(gameId, playerName string, scoreDelta uint8) error {
	return nil
}
