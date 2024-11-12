package db

import (
	"fmt"
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
  name varchar(10) NOT NULL,
  game_id varchar(8) REFERENCES games(game_id) ON DELETE CASCADE,
  is_active boolean NOT NULL,
  PRIMARY KEY (name, game_id)

  CONSTRAINT non_empty_player CHECK (TRIM(name) <> '')
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
	s.Logger.Info(fmt.Sprintf("Database %s setup successfully", sqlite_dbfile))
	return nil
}

func (s *SqliteStore) CloseConnection() {
	s.Logger.Info("Closing database connection")
	if err := s.Conn.Close(); err != nil {
		s.Logger.Error("Failed to tear down database connection")
		return
	}
	s.Logger.Info("Database connection closed successfully")
}

func (s *SqliteStore) GetGameById(gameId string) *Game {
	sql := `SELECT * FROM games WHERE game_id = ?;`
	s.Logger.Info(fmt.Sprintf("Fetching game %s", gameId))
	game := &Game{}
	err := s.Conn.Get(game, sql, gameId)
	if err != nil {
		s.Logger.Error("Failed to fetch game", slog.String("error", err.Error()))
		return nil
	}
	return game
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
		errRoll := txn.Rollback()
		if errRoll != nil {
			s.Logger.Error("Failed to rollback CreateGame txn", slog.String("error", errRoll.Error()))
			return errRoll
		}
		return err
	}
	s.Logger.Info("Game created successfully")
	insertPlayerSQL := `INSERT INTO players VALUES(?, ?, ?);`
	_, err = txn.Exec(insertPlayerSQL, player, gameId, true)
	if err != nil {
		s.Logger.Error("Failed to save player", slog.String("error", err.Error()))
		errRoll := txn.Rollback()
		if errRoll != nil {
			s.Logger.Error("Failed to rollback CreateGame txn", slog.String("error", errRoll.Error()))
			return errRoll
		}
		return err
	}

	errCommit := txn.Commit()
	if errCommit != nil {
		s.Logger.Error("Failed to Commit CreateGame txn", slog.String("error", errCommit.Error()))
		return errCommit
	}
	return nil
}

func (s *SqliteStore) AddPlayerToGame(gameId, playerName string) error {
	txn, err := s.Conn.Beginx()
	if err != nil {
		s.Logger.Error("Failed to add player to game", slog.String("error", err.Error()))
		return err
	}
	insertPlayerSQL := `INSERT INTO players VALUES(?, ?, ?);`
	_, err = txn.Exec(insertPlayerSQL, playerName, gameId, true)
	if err != nil {
		s.Logger.Error("Failed to add player to game", slog.String("error", err.Error()))
		errRoll := txn.Rollback()
		if errRoll != nil {
			s.Logger.Error("Failed to rollback AddPlayerToGame txn", slog.String("error", errRoll.Error()))
			return errRoll
		}
		return err
	}
	s.Logger.Info("Player added to the game successfully")
	updatePlayerCountSQL := `UPDATE games SET player_count=player_count+1 WHERE game_id = ?;`
	_, err = txn.Exec(updatePlayerCountSQL, gameId)
	if err != nil {
		s.Logger.Error("Failed to update player count", slog.String("error", err.Error()))
		errRoll := txn.Rollback()
		if errRoll != nil {
			s.Logger.Error("Failed to rollback AddPlayerToGame txn", slog.String("error", errRoll.Error()))
			return errRoll
		}
		return err
	}

	errCommit := txn.Commit()
	if errCommit != nil {
		s.Logger.Error("Failed to Commit AddPlayerToGame txn", slog.String("error", errCommit.Error()))
		return errCommit
	}
	return nil
}

func (s *SqliteStore) UpdatePlayerScore(gameId, playerName string, scoreDelta uint8) error {
	return nil
}
