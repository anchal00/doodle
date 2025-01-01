package utils

import (
	crypto "crypto/rand"
	"encoding/hex"
	"math/rand"
)

func CreateSessionToken() (string, error) {
	token := make([]byte, 32)
	_, err := crypto.Read(token)
	if err != nil {
		return "", err
	}
	// Convert bytes to a hex string
	return hex.EncodeToString(token), nil
}

func GetRandomGameId(size int) string {
	r := make([]byte, size)
	for i := 0; i < size; i += 1 {
		offset := rand.Intn(26)
		r[i] = byte(97 + offset)
	}
	return string(r)
}
