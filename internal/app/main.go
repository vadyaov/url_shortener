package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"        // Для сигналов
	"os/signal" // Для graceful shutdown
	"syscall"   // Для сигналов
	"time"

	"github.com/vadyaov/url_shortener/internal/service"
	"github.com/vadyaov/url_shortener/internal/storage"
)

const (
	defaultBaseUrl   = "localhost:8081"
	getShortUrlPath  = "/get_short_url"
	getOriginUrlPath = "/get_origin_url"
	httpRedirect     = "/redirect"
)

type Response struct {
	Url    string `json:"url"`
	Status int    `json:"status"`
	Error  string `json:"error"`
}

var urlService *service.UrlService

// POST
func handleCreateShortUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Error parsing form: %v", err))
		return
	}

	origin_url := r.Form.Get("url")
	if origin_url == "" {
		respondWithError(w, http.StatusBadRequest, "Incorrect or empry 'url' field")
		return
	}

	// need to parse origin_url to remove 'http/https' prefix
	// https://github.com --> github.com etc.

	short, err := urlService.GetShortUrl(origin_url)
	if err != nil {
		if errors.Is(err, storage.ErrDuplicateShortCode) {
			respondWithError(w, http.StatusConflict, fmt.Sprintf("Failed to create short URL due to conflict: %v", err))
		} else {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create short URL: %v", err))
		}
		return
	}

	fullShortUrl := fmt.Sprintf("http://%s/%s", r.Host, short)

	respondWithJSON(w, http.StatusCreated, &Response{Url: fullShortUrl, Status: http.StatusCreated})
}

// GET
func handleGetOriginUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	short_url := r.URL.Query().Get("url")
	if short_url == "" {
		respondWithError(w, http.StatusBadRequest, "Incorrect or empty 'url' field")
		return
	}

	origin, err := urlService.GetOriginUrl(short_url)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			respondWithError(w, http.StatusNotFound, "Short URL not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get original URL: %v", err))
		}
		return
	}

	respondWithJSON(w, http.StatusOK, &Response{Url: origin, Status: http.StatusOK})
}

func handleRedirect(w http.ResponseWriter, r *http.Request) {
	shortCode := r.URL.Query().Get("url")
	origin, err := urlService.GetOriginUrl(shortCode)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMovedPermanently)
	http.Redirect(w, r, origin, http.StatusFound)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, Response{Error: message, Status: code})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	storeType := flag.String("store", "inmemory", "Storage type: 'inmemory' or 'postgres'")
	postgresDSN := flag.String("dsn", "postgres://shortener:pswrd@localhost:5432/urlshortener_db?sslmode=disable", "PostgreSQL DSN")

	flag.Parse()

	var store storage.URLStore

	appCtx, cancelAppCtx := context.WithCancel(context.Background())
	defer cancelAppCtx()

	log.Printf("Selected storage type: %s", *storeType)

	switch *storeType {
	case "inmemory":
		store = storage.NewInMemoryStore()
		log.Println("Using in-memory store")
	case "postgres":
		pgStore, pgErr := storage.NewPostgresStore(appCtx, *postgresDSN)
		if pgErr != nil {
			log.Fatalf("Failed to initialize PostgreSQL store: %s", pgErr)
		}
		store = pgStore
		log.Println("Using postgres store")
	default:
		log.Fatalf("Unsupported store type: %s. Use 'inmemory' or 'postgres'.", *storeType)
	}

	urlService = service.NewUrlService(store)

	mux := http.NewServeMux()

	mux.HandleFunc(getShortUrlPath, handleCreateShortUrl)
	mux.HandleFunc(getOriginUrlPath, handleGetOriginUrl)
	mux.HandleFunc(httpRedirect, handleRedirect)

	server := &http.Server{
		Addr:    defaultBaseUrl,
		Handler: mux,
	}

	// Graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit // Блокируемся до получения сигнала

		log.Println("Shutting down server...")
		cancelAppCtx() // Сигнализируем компонентам (например, pgStore) о завершении

		// Контекст для graceful shutdown сервера
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdown()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("Server forced to shutdown: %v", err)
		}
		log.Println("Server gracefully stopped.")

		// Закрываем соединения с БД после остановки сервера
		if pgStore, ok := store.(*storage.PostgresStore); ok {
			pgStore.Close()
		}
	}()

	fmt.Printf("Server running on %s\n", defaultBaseUrl)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("ListenAndServe: %v", err)
	}
	log.Println("Server exiting.")
}
