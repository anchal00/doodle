//go:generate mockery --with-expecter=true --name=Repository --dir=db --output=db/mocks
//go:generate mockery --with-expecter=true --name=Logger --dir=logger --output=logger/mocks
//go:generate mockery  --with-expecter=true --name=ConnectionStore --dir=state --output=state/mocks
package main

import (
	"doodle/server"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		slog.Error("Failed to load .env file")
		return
	}
	// TODO: Accept port via args
	port := os.Getenv("DOODLE_PORT")
	if len(port) == 0 {
		slog.Error("Env DOODLE_PORT not set")
		return
	}
	gs, err := server.NewGameServer(port)
	if err != nil {
		return
	}
	gs.Run()
}
