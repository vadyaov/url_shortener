package service

import (
	"crypto/sha256"
	"github.com/yihleego/base62"
)

func CreateShortUrl(origin string) (string, error) {
	hash := sha256.Sum256([]byte(origin))
	encoded := base62.StdEncoding.EncodeToString(hash[:])
	return encoded[:7], nil
}