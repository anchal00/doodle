package main

import (
	"doodle/server/apis/v1"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

const HTTP_API_V1_PREFIX = "/api/v1"

func getLogger() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})
	logger := slog.New(handler)
	return logger
}

func main() {
	server := &server.GameAPIServer{Logger: getLogger()}
	router := mux.NewRouter().PathPrefix(HTTP_API_V1_PREFIX).Subrouter()

	router.HandleFunc("/game", server.CreateNewGame).Methods("POST")
	router.HandleFunc("/game/{gameId:[a-z]+}", server.JoinGame).Methods("POST")

	router.HandleFunc("/push", server.HandleClientPush)

	// TODO: Accept port via args
	server.Logger.Info("Starting server on port 9000")
	if err := http.ListenAndServe(":9000", router); err != nil {
		server.Logger.Error("Failed to start server on port 9000", slog.String("error", err.Error()))
		return
	}
}
