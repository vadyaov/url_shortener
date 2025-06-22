package storage

import (
	"errors"
)

var ErrNotFound = errors.New("URL not found")
var ErrDuplicateShortCode = errors.New("short code already exists for a different URL")

type URLStore interface {
	// Save the mapping between originalUrl and shortCode.
	// Need to check the case if shortCode already exists
	SaveURL(originalURL, shortCode string) error

	// Returns original URL from the short code
	// Returns ErrNotFound if the URL does not exist
	GetOriginURL(shortCode string) (string, error)

	// Returns short code from the original URL
	// Returns ErrNotFound if the URL does not exist
	GetShortURL(originURL string) (string, error)
}