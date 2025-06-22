package storage

import (
	"context"
	"errors"
	"log"
	"fmt"

	"github.com/jackc/pgx/v5"
  "github.com/jackc/pgx/v5/pgconn"
  "github.com/jackc/pgx/v5/pgxpool"
)

type 	PostgresStore struct {
	pool *pgxpool.Pool
	ctx  context.Context
}

// dsn - Data Source Name, e.g. "postgres://user:password@host:port/dbname?sslmode=disable"
func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse DSN: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return store, nil
}

func (store *PostgresStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS urls (
			short_code VARCHAR(16) PRIMARY KEY, -- Увеличим немного длину на всякий случай
			original_url TEXT NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_original_url_unique ON urls (original_url);
	`

	_, err := store.pool.Exec(store.ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema initialization: %w", err)
  }
	log.Println("Database schema initialized (or already existed).")
	return nil
}

func (store *PostgresStore) SaveURL(originUrl, shortCode string) error {
	var existingOrigin string
	err := store.pool.QueryRow(store.ctx, "SELECT origin_url FROM urls WHERE short_code = $1", shortCode).Scan(&existingOrigin)
	if err == nil {
		if originUrl != existingOrigin {
			return fmt.Errorf("%w: short code '%s' already maps to '%s'", ErrDuplicateShortCode, shortCode, existingOrigin)
		}
		return nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to check existing short code: %w", err)
	}

	query := `INSERT INTO urls (short_code, origin_url) VALUES ($1, $2)`
	_, err := store.pool.Exec(store.ctx, query, shortCode, originUrl)
	if err != nil {
		return fmt.Errorf("failed to save URL to postgres: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetOriginURL(shortCode string) (string, error) {
	var originUrl string
	query := `SELECT origin_url FROM urls WHERE short_code = $1`
	err := store.pool.QueryRow(store.ctx, query, shortCode).Scan(&originUrl)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get origin url from psql: %w", err)
	}
	return originUrl, nil
}

func (s *PostgresStore) GetShortURL(originUrl string) (string, error) {
	var shortUrl string
	query := `SELECT short_url FROM urls WHERE origin_url = $1`
	err := store.pool.QueryRow(store.ctx, query, originUrl).Scan(&shortUrl)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get short url from psql: %w", err)
	}
	return shortUrl, nil
}

func (s *PostgresStore) Close() {
	s.pool.Close()
	fmt.Println("PostgreSQL connection pool closed.")
}
