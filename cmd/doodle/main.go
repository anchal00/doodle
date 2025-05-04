package main

import (
	"github.com/anchal00/doodle/internal/server"
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
