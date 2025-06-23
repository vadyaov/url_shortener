package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"        // Для сигналов
	"os/signal" // Для graceful shutdown
	"syscall"   // Для сигналов
	"time"

	handlers "github.com/vadyaov/url_shortener/internal/handlers/http"
	"github.com/vadyaov/url_shortener/internal/service"
	"github.com/vadyaov/url_shortener/internal/storage"
)

const (
	defaultBaseUrl   = "localhost:8081"
	getShortUrlPath  = "/get_short_url"
	getOriginUrlPath = "/get_origin_url"
	httpRedirect     = "/"
)

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

	urlSvc := service.NewUrlService(store)
	urlH := handlers.NewUrlHandler(urlSvc)

	mux := http.NewServeMux()

	mux.HandleFunc(getShortUrlPath, urlH.HandleCreateShortUrl)
	mux.HandleFunc(getOriginUrlPath, urlH.HandleGetOriginUrl)
	mux.HandleFunc(httpRedirect, urlH.HandleRedirect)

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
