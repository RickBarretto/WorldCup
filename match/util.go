package main

import (
	"crypto/rand"
	"encoding/hex"
)

func newCardID() CardID {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
