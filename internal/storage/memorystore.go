package storage

import (
	"fmt"
	"sync"
)

type InMemoryStore struct {
	mu sync.RWMutex
	shortToOrig map[string]string
	origToShort map[string]string
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore {
		shortToOrig: make(map[string]string),
		origToShort: make(map[string]string),
	}
}

func (store *InMemoryStore) SaveURL(originalURL, shortCode string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if existingOrig, ok := store.shortToOrig[shortCode]; ok && existingOrig != originalURL {
		return fmt.Errorf("%w: short code '%s' already maps to '%s'", ErrDuplicateShortCode, shortCode, existingOrig)
	}

	if existingShort, ok := store.origToShort[originalURL]; ok && existingShort != shortCode {
		delete(store.shortToOrig, existingShort)
	}

	store.shortToOrig[shortCode] = originalURL
	store.origToShort[originalURL] = shortCode

	return nil
}

func (store *InMemoryStore) GetOriginURL(shortCode string) (string, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	originUrl, ok := store.shortToOrig[shortCode]
	if !ok {
		return "", ErrNotFound
	}

	return originUrl, nil
}

func (store *InMemoryStore) GetShortURL(originalURL string) (string, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	shortCode, ok := store.origToShort[originalURL]
	if !ok {
		return "", ErrNotFound
	}

	return shortCode, nil
}