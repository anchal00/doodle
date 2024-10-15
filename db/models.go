package db

type Game struct {
	GameId      string `db:"game_id"`
	PlayerCount uint8  `db:"player_count"`
}

type Player struct {
	name      string `db:"name"`
	GameId    string `db:"game_id"`
	is_active bool   `db:"is_active"`
}
