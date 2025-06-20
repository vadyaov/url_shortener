package main

import (
	"fmt"
	"net/http"
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
	Url 		 string `json:"url"`
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

		short := service.GetShortUrl(origin_url)

		resp := &Response {
			Url: 			short,
			Status:   http.StatusCreated,
			Error:    "",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode responce data; %v", err), http.StatusInternalServerError)
			return
		}
	}
}

// GET
func handleGetOriginalUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if err := r.ParseForm(); err != nil {
			http.Error(w, fmt.Sprintf("Error parsing form: %v", err), http.StatusInternalServerError)
			return
		}
		short_url := r.Form.Get("shortUrl")
		if len(short_url) == 0 {
			http.Error(w, "Incorrect or empty 'shortUrl' field", http.StatusInternalServerError)
			return
		}
		fmt.Printf("POST CreateShortUrlRequest. Original URL: %s\n", short_url)

		origin, err := service.GetOriginUrl(short_url)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		resp := &Response {
			Url: 			origin,
			Status:   http.StatusCreated,
			Error:    "",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode responce data; %v", err), http.StatusInternalServerError)
			return
		}
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