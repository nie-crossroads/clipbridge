package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

func NewToken() string {
	var bytes [32]byte
	rand.Read(bytes[:])
	return base64.RawURLEncoding.EncodeToString(bytes[:])
}

func RequestID() string {
	var bytes [16]byte
	rand.Read(bytes[:])
	bytes[6] = bytes[6]&0x0f | 0x40
	bytes[8] = bytes[8]&0x3f | 0x80
	id := hex.EncodeToString(bytes[:])
	return id[:8] + "-" + id[8:12] + "-" + id[12:16] + "-" + id[16:20] + "-" + id[20:]
}

func Digest(value string) []byte {
	hash := sha256.Sum256([]byte(value))
	return hash[:]
}
