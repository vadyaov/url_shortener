package service

import (
	"crypto/sha256"
	"sync"
	"fmt"
	"errors"

	"github.com/yihleego/base62"
)

type SyncUrlMap struct {
	urls  map[string]string
	m          sync.Mutex
}

var shortUrlsMap = SyncUrlMap {
	urls: make(map[string]string),
}

var originUrlsMap = SyncUrlMap {
	urls: make(map[string]string),
}

func GetShortUrl(origin string) (string) {
	shortUrlsMap.m.Lock()
	defer shortUrlsMap.m.Unlock()


	if _, ok := shortUrlsMap.urls[origin]; ok {
		fmt.Println("This url is already exists in map")
		return shortUrlsMap.urls[origin]
	}

	fmt.Println("Creating new Short Url...")
	hash := sha256.Sum256([]byte(origin))
	encoded := base62.StdEncoding.EncodeToString(hash[:])[:7]
	shortUrlsMap.urls[origin] = encoded
	{
		originUrlsMap.m.Lock()
		defer originUrlsMap.m.Unlock()

		originUrlsMap.urls[encoded] = origin
	}

	return encoded
}

func GetOriginUrl(short string) (string, error) {
	originUrlsMap.m.Lock()
	defer originUrlsMap.m.Unlock()
	if _, ok := originUrlsMap.urls[short]; !ok {
		return "", errors.New("url does not exist")
	}
	return originUrlsMap.urls[short], nil
}