package utils

import (
	"math/rand"
)

func GetRandomGameId(size int) string {
	r := make([]byte, size)
	for i := 0; i < size; i += 1 {
		offset := rand.Intn(26)
    r[i] = byte(97+offset)
	}
	return string(r)
}
