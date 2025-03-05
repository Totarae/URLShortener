package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/Totarae/URLShortener/internal/database"
	"github.com/Totarae/URLShortener/internal/storage"
	"github.com/Totarae/URLShortener/internal/util"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Handler struct {
	store   storage.Storage // Use the new URLStore for thread safety
	baseURL string
	DB      database.DBInterface
	Logger  *zap.Logger
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

var validIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{6,22}$`)

func NewHandler(store storage.Storage, baseURL string, db database.DBInterface, logger *zap.Logger) *Handler {
	return &Handler{
		store:   store,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		DB:      db,
		Logger:  logger,
	}
}

func (h *Handler) ReceiveURL(res http.ResponseWriter, req *http.Request) {

	body, err := io.ReadAll(req.Body)
	if err != nil {
		h.Logger.Error("Ошибка чтения тела запроса", zap.Error(err))
		http.Error(res, "BadRequest", http.StatusBadRequest)
		return
	}

	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(res, "URL empty", http.StatusBadRequest)
		return
	}
	// Проверка корректности URL
	parsedURL, err := url.ParseRequestURI(originalURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		http.Error(res, "Invalid URL", http.StatusBadRequest)
		return
	}

	shortURL := util.GenerateShortURL(originalURL, h.baseURL, h.store)
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	res.Write([]byte(shortURL))
}

func (h *Handler) ResponseURL(res http.ResponseWriter, req *http.Request) {

	id := chi.URLParam(req, "id")
	if id == "" {
		http.Error(res, "Bad Request: Missing ID in URL", http.StatusBadRequest)
		return
	}

	// Проверяем ID на корректность
	if !validIDPattern.MatchString(id) {
		http.Error(res, "Bad Request: Invalid ID format", http.StatusBadRequest)
		return
	}

	fmt.Println("Incoming ID : ", id)
	// Ищем оригинальный URL в хранилище
	originalURL, exists := h.store.Get(id)
	if !exists {
		http.NotFound(res, req)
		return
	}

	// Устанавливаем заголовок Location и код 307
	res.Header().Set("Location", originalURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *Handler) ReceiveShorten(res http.ResponseWriter, req *http.Request) {
	var request ShortenRequest
	if req.Body == nil {
		http.Error(res, "Empty request body", http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&request); err != nil {
		http.Error(res, "Invalid JSON", http.StatusBadRequest)
		return
	}

	originalURL := strings.TrimSpace(request.URL)
	if originalURL == "" {
		http.Error(res, "URL empty", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.ParseRequestURI(originalURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		http.Error(res, "Invalid URL", http.StatusBadRequest)
		return
	}

	shortURL := util.GenerateShortURL(originalURL, h.baseURL, h.store)

	response := ShortenResponse{Result: shortURL}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	json.NewEncoder(res).Encode(response)
}

func (h *Handler) PingHandler(res http.ResponseWriter, req *http.Request) {
	if err := h.DB.Ping(req.Context()); err != nil {
		h.Logger.Error("Database ping failed", zap.Error(err))
		http.Error(res, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
	res.Write([]byte("OK"))
}
