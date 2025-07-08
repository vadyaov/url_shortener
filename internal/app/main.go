package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	shortener_v0 "github.com/vadyaov/url_shortener/internal/app/grpc/pkg/shortener_v0" // Укажите правильный путь
	grpchandlers "github.com/vadyaov/url_shortener/internal/handlers/grpc"
	httphandlers "github.com/vadyaov/url_shortener/internal/handlers/http"
	"github.com/vadyaov/url_shortener/internal/service"
	"github.com/vadyaov/url_shortener/internal/storage"

	"google.golang.org/grpc"
)

const (
	httpServerAddr = "localhost:8081"
	grpcServerAddr = "localhost:6969"

	getShortUrlPath  = "/get_short_url"
	getOriginUrlPath = "/get_origin_url"
	httpRedirect     = "/"
)

func main() {
	storeType := flag.String("store", "inmemory", "Storage type: 'inmemory' or 'postgres'")
	postgresDSN := flag.String("dsn", "postgres://shortener:pswrd@localhost:5432/urlshortener_db?sslmode=disable", "PostgreSQL DSN")
	flag.Parse()

	appCtx, cancelAppCtx := context.WithCancel(context.Background())
	defer cancelAppCtx()

	// --- Инициализация хранилища и сервиса (без изменений) ---
	var store storage.URLStore
	log.Printf("Selected storage type: %s", *storeType)
	switch *storeType {
	case "inmemory":
		store = storage.NewInMemoryStore()
	case "postgres":
		pgStore, pgErr := storage.NewPostgresStore(appCtx, *postgresDSN)
		if pgErr != nil {
			log.Fatalf("Failed to initialize PostgreSQL store: %s", pgErr)
		}
		store = pgStore
	default:
		log.Fatalf("Unsupported store type: %s. Use 'inmemory' or 'postgres'.", *storeType)
	}
	urlSvc := service.NewUrlService(store)

	go runHTTPServer(urlSvc)

	grpcServer := runGRPCServer(urlSvc)

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down servers...")
	cancelAppCtx() // Сигнал для компонентов о завершении

	grpcServer.GracefulStop() // Останавливаем gRPC сервер
	log.Println("gRPC server stopped.")

	// Даем время на завершение HTTP-сервера
	time.Sleep(5 * time.Second)

	if pgStore, ok := store.(*storage.PostgresStore); ok {
		pgStore.Close()
		log.Println("PostgreSQL connection closed.")
	}

	log.Println("Server exiting.")
}

func runHTTPServer(urlSvc service.URLShortenerService) {
	urlH := httphandlers.NewUrlHandler(urlSvc)
	mux := http.NewServeMux()
	mux.HandleFunc(getShortUrlPath, urlH.HandleCreateShortUrl)
	mux.HandleFunc(getOriginUrlPath, urlH.HandleGetOriginUrl)
	mux.HandleFunc(httpRedirect, urlH.HandleRedirect)

	server := &http.Server{
		Addr:    httpServerAddr,
		Handler: mux,
	}

	go func() {
		fmt.Printf("HTTP Server running on %s\n", httpServerAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP ListenAndServe: %v", err)
		}
	}()

	// Отдельный канал для ожидания сигнала на остановку HTTP сервера
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server Shutdown Failed:%+v", err)
	}
	log.Println("HTTP server gracefully stopped.")
}

func runGRPCServer(urlSvc service.URLShortenerService) *grpc.Server {
	lis, err := net.Listen("tcp", grpcServerAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	grpcHandler := grpchandlers.NewServer(urlSvc)
	shortener_v0.RegisterShortenerV0Server(grpcServer, grpcHandler)

	go func() {
		fmt.Printf("gRPC Server running on %s\n", grpcServerAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	return grpcServer
}
