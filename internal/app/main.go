package main

import (
	"fmt"
	"net/http"
	"io"
	"log"
	"encoding/json"

	"github.com/vadyaov/url_shortener/internal/service"
)

const (
	baseUrl = "localhost:8081"
	createShortUrl = "/create_short_url"
	getOriginalUrl = "/get_original_url"
)

type Response struct {
	ShortUrl string `json:"short_url"`
	Status   int    `json:"status"`
	Error    string
}

// POST
func handleCreateShortUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, fmt.Sprintf("Error parsing form: %v", err), http.StatusInternalServerError)
			return
		}
		origin_url := r.Form.Get("originUrl")
		if len(origin_url) == 0 {
			http.Error(w, "Incorrect or empty 'originUrl' field", http.StatusInternalServerError)
			return
		}
		fmt.Printf("POST CreateShortUrlRequest. Original URL: %s\n", origin_url)

		short, err := service.CreateShortUrl(origin_url)
		if err != nil {
			http.Error(w, "Error creating short url", http.StatusInternalServerError)
			return
		}

		resp := &Response {
			ShortUrl: short,
			Status:   http.StatusCreated,
			Error:    "",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode responce data; %v", err), http.StatusInternalServerError)
			return
		}

		// map long url to short
		// save
	}
}

// GET
func handleGetOriginalUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading responce body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		fmt.Printf("POST CreateShortUrlRequest. Body: %s\n", string(body))
	}
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc(createShortUrl, handleCreateShortUrl)
	mux.HandleFunc(getOriginalUrl, handleGetOriginalUrl)

	fmt.Println(fmt.Sprintf("Server running on %s", baseUrl))
	err := http.ListenAndServe(baseUrl, mux)
	if err != nil {
		log.Fatal(err)
	}
}