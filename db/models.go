package db

type Game struct {
	GameId       string `db:"game_id"`
	PlayerCount  uint8  `db:"player_count"`
	MaxPlayers   uint8  `db:"max_players"`
	CurrentRound uint8  `db:"current_round"`
	TotalRounds  uint8  `db:"total_rounds"`
}

type Player struct {
	Name      string `db:"name"`
	GameId    string `db:"game_id"`
	IsAdmin   bool   `db:"is_admin"`
	AuthToken string `db:"token"`
}
