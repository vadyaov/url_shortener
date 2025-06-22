package main

import (
	"encoding/json"
	"net/http"
	"errors"
	// "flag"
	"fmt"
	"log"

	"github.com/vadyaov/url_shortener/internal/service"
	"github.com/vadyaov/url_shortener/internal/storage"
)

const (
	defaultBaseUrl 		 = "localhost:8081"
	getShortUrlPath = "/get_short_url"
	getOriginUrlPath = "/get_origin_url"
	httpRedirect       = "/redirect"
)

type Response struct {
	Url 		 string `json:"url"`
	Status   int    `json:"status"`
	Error    string `json:"error"`
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
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Incorrect or empry 'url' field"))
		return
	}

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
	mux := http.NewServeMux()

	store := storage.NewInMemoryStore()
	urlService = service.NewUrlService(store)

	mux.HandleFunc(getShortUrlPath, handleCreateShortUrl)
	mux.HandleFunc(getOriginUrlPath, handleGetOriginUrl)
	mux.HandleFunc(httpRedirect, handleRedirect)

	fmt.Println(fmt.Sprintf("Server running on %s", defaultBaseUrl))
	err := http.ListenAndServe(defaultBaseUrl, mux)
	if err != nil {
		log.Fatal(err)
	}
}