package service

import (
	"crypto/sha256"
	"fmt"
	"errors"

	"github.com/yihleego/base62"

	"github.com/vadyaov/url_shortener/internal/storage"
)

type UrlService struct {
	store storage.URLStore
}

func NewUrlService(store storage.URLStore) *UrlService {
	return &UrlService{store: store}
}

func (us *UrlService) GetShortUrl(origin string) (string, error) {
	existingShort, err := us.store.GetShortURL(origin)
	if err == nil {
		fmt.Println("This url is already exists in map, returning existing short url: ", existingShort)
		return existingShort, nil
	}

	fmt.Println("Creating new short url for ", origin)

	hash := sha256.Sum256([]byte(origin))
	// try several lengths if collisions occur
	for length := 7; length <= 10; length++ {
		encoded := base62.StdEncoding.EncodeToString(hash[:])[:length]
		
		// try to save. if shortCode is already exist
		// then SaveURL should produce an error --> go to another cycle iter
		errSave := us.store.SaveURL(origin, encoded)
		if errSave == nil {
			return encoded, nil
		}

		if errors.Is(errSave, storage.ErrDuplicateShortCode) {
			fmt.Printf("Collision detected for short code '%s' with length %d, trying longer.\n", encoded, length)

			if length == 10 {
				return "", fmt.Errorf("failed to genereate unique short code for '%s' after multiple attempts: %w", origin, errSave)
			}

			continue
		}

		return "", fmt.Errorf("failed to save URL: %w", errSave)
	}
	return "", errors.New("could not generate unique short URL")
}

func (us *UrlService) GetOriginUrl(short string) (string, error) {
	origin, err := us.store.GetOriginURL(short)

	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return "", err
		}
		return "", fmt.Errorf("failed to get original url: %w", err)
	}
	return origin, nil
}