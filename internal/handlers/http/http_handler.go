package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	normalizeurl "github.com/vadyaov/url_shortener/internal/normalize"
	"github.com/vadyaov/url_shortener/internal/service"
	"github.com/vadyaov/url_shortener/internal/storage"
)

type Response struct {
	Url    string `json:"url"`
	Status int    `json:"status"`
	Error  string `json:"error"`
}

type UrlHandler struct {
	service service.URLShortenerService
}

func NewUrlHandler(svc service.URLShortenerService) *UrlHandler {
	return &UrlHandler{
		service: svc,
	}
}

func (h *UrlHandler) HandleCreateShortUrl(w http.ResponseWriter, r *http.Request) {
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

	orig_url, err := normalizeurl.Normalize(origin_url)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	short, err := h.service.GetShortUrl(orig_url)
	if err != nil {
		if errors.Is(err, storage.ErrDuplicateShortCode) {
			respondWithError(w, http.StatusConflict, fmt.Sprintf("Failed to create short URL due to conflict: %v", err))
		} else {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create short URL: %v", err))
		}
		return
	}

	// fullShortUrl := fmt.Sprintf("http://%s/%s", r.Host, short)

	respondWithJSON(w, http.StatusCreated, &Response{Url: short, Status: http.StatusCreated})
}

func (h *UrlHandler) HandleGetOriginUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	short_url := r.URL.Query().Get("url")
	if short_url == "" {
		respondWithError(w, http.StatusBadRequest, "Incorrect or empty 'url' field")
		return
	}

	origin, err := h.service.GetOriginUrl(short_url)
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

func (h *UrlHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	shortCode := strings.TrimPrefix(r.URL.Path, "/")
	if shortCode == "" {
		http.Error(w, "Short code is missing", http.StatusBadRequest)
		return
	}

	originUrl, err := h.service.GetOriginUrl(shortCode)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, originUrl, http.StatusMovedPermanently)
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
