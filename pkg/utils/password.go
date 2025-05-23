package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

func EncryptPassword(password string) string {
	// SHA256加密
	h := sha256.New()
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil))
}
