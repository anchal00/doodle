package main

import (
	"doodle/server"
	"log/slog"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		slog.Error("Failed to load .env file")
		return
	}
	// TODO: Accept port via args
	gs, err := server.NewGameServer("9000")
	if err != nil {
		return
	}
	gs.Run()
}
