package main

import (
	"doodle/server"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	// TODO: Accept port via args
	gs, err := server.NewGameServer("9000")
	if err != nil {
		return
	}
	gs.Run()
}
