package infra

import (
	"crypto/rand"
	"encoding/hex"
)

func newMsgID() string {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "msg"
	}
	return hex.EncodeToString(b[:])
}
