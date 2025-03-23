package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Totarae/URLShortener/internal/auth"
	"github.com/Totarae/URLShortener/internal/model"
	"github.com/Totarae/URLShortener/internal/repositories"
	"github.com/Totarae/URLShortener/internal/storage"
	"github.com/Totarae/URLShortener/internal/util"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Handler struct {
	store   storage.Storage // Use the new URLStore for thread safety
	baseURL string
	Repo    repositories.URLRepositoryInterface
	Logger  *zap.Logger
	Mode    string
	Auth    *auth.Auth
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

var validIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{6,22}$`)

func NewHandler(store storage.Storage, baseURL string, repo repositories.URLRepositoryInterface, logger *zap.Logger, mode string, authService *auth.Auth) *Handler {
	return &Handler{
		store:   store,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		Repo:    repo,
		Logger:  logger,
		Mode:    mode,
		Auth:    authService,
	}
}

func (h *Handler) ReceiveURL(res http.ResponseWriter, req *http.Request) {

	// Получаем или создаём userID через куку
	userID := h.Auth.GetOrSetUserID(res, req)

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

	shortURL := util.GenerateShortURL(originalURL)
	log.Printf("Generated URL: %s", shortURL)
	urlObj := &model.URLObject{
		Origin:  originalURL,
		Shorten: shortURL,
		Created: time.Now(),
		UserID:  userID,
	}

	if h.Mode == "database" {
		err = h.Repo.SaveURL(req.Context(), urlObj)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// URL уже существует, получаем его сокращённый вариант
				existingShortURL, lookupErr := h.Repo.GetShortURLByOrigin(req.Context(), originalURL)
				if lookupErr != nil {
					h.Logger.Error("Ошибка получения существующего сокращённого URL", zap.Error(lookupErr))
					http.Error(res, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				// Сразу отстреливаем ответ
				res.Header().Set("Content-Type", "text/plain")
				res.WriteHeader(http.StatusConflict)
				res.Write([]byte(h.baseURL + "/" + existingShortURL))
				return
			}

			h.Logger.Error("Ошибка сохранения URL в БД", zap.Error(err))
			http.Error(res, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	} else if h.Mode == "file" {
		log.Printf("In file saving")
		entry := model.Entry{ShortURL: shortURL, OriginalURL: originalURL}
		if err := h.store.AppendToFile(entry); err != nil {
			log.Printf("Ошибка сохранения в файл: %v", err)
		}
		h.store.Save(shortURL, originalURL)
	} else {
		if err := util.SaveURL(originalURL, shortURL, h.store); err != nil {
			log.Printf("Ошибка сохранения в память: %v", err)
		}
	}

	shortURL = h.baseURL + "/" + shortURL

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

	var originalURL string

	if h.Mode == "database" {
		urlObj, err := h.Repo.GetURL(req.Context(), id)
		if err != nil {
			h.Logger.Error("Ошибка получения URL из БД", zap.Error(err))
			http.Error(res, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if urlObj == nil {
			http.NotFound(res, req)
			return
		}
		originalURL = urlObj.Origin // Теперь правильно присваиваем оригинальный URL

	} else {
		var exists bool
		// Ищем оригинальный URL в хранилище
		originalURL, exists = h.store.Get(id)
		if !exists {
			http.NotFound(res, req)
			return
		}
	}

	if originalURL == "" && h.store != nil {
		var exists bool
		originalURL, exists = h.store.Get(id)
		if exists {
			fmt.Println("Found in store:", originalURL)
		} else {
			fmt.Println("Not found in store")
		}
	}

	if originalURL == "" {
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

	shortURL := util.GenerateShortURL(originalURL)

	urlObj := &model.URLObject{
		Origin:  originalURL,
		Shorten: shortURL,
		Created: time.Now(),
	}

	if h.Mode == "database" {
		err = h.Repo.SaveURL(req.Context(), urlObj)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// URL уже существует, возвращаем существующий сокращённый URL
				existingShortURL, lookupErr := h.Repo.GetShortURLByOrigin(req.Context(), originalURL)
				if lookupErr != nil {
					h.Logger.Error("Ошибка получения существующего сокращённого URL", zap.Error(lookupErr))
					http.Error(res, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				response := ShortenResponse{Result: h.baseURL + "/" + existingShortURL}
				res.Header().Set("Content-Type", "application/json")
				res.WriteHeader(http.StatusConflict)
				json.NewEncoder(res).Encode(response)
				return
			}
			h.Logger.Error("Ошибка сохранения URL в БД", zap.Error(err))
			http.Error(res, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	} else if h.Mode == "file" {
		log.Printf("In file saving")
		entry := model.Entry{ShortURL: shortURL, OriginalURL: originalURL}
		if err := h.store.AppendToFile(entry); err != nil {
			log.Printf("Ошибка сохранения в файл: %v", err)
		}
		h.store.Save(shortURL, originalURL)
	} else {
		if err := util.SaveURL(originalURL, shortURL, h.store); err != nil {
			log.Printf("Ошибка сохранения в память: %v", err)
		}
	}

	shortURL = h.baseURL + "/" + shortURL

	response := ShortenResponse{Result: shortURL}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	json.NewEncoder(res).Encode(response)
}

func (h *Handler) PingHandler(res http.ResponseWriter, req *http.Request) {
	if err := h.Repo.Ping(req.Context()); err != nil {
		h.Logger.Error("Database ping failed", zap.Error(err))
		http.Error(res, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
	res.Write([]byte("OK"))
}

func (h *Handler) BatchShortenHandler(res http.ResponseWriter, req *http.Request) {

	if req.Body == nil {
		http.Error(res, "Empty request body", http.StatusBadRequest)
		return
	}

	var batchRequest []BatchShortenRequest
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&batchRequest); err != nil {
		http.Error(res, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(batchRequest) == 0 {
		http.Error(res, "Batch request is empty", http.StatusBadRequest)
		return
	}

	var batchResponse []BatchShortenResponse
	var urlObjects []*model.URLObject

	for _, item := range batchRequest {
		originalURL := strings.TrimSpace(item.OriginalURL)
		parsedURL, err := url.ParseRequestURI(originalURL)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			continue
		}

		shortURL := util.GenerateShortURL(originalURL)
		shortFullURL := h.baseURL + "/" + shortURL

		urlObj := &model.URLObject{
			Origin:  originalURL,
			Shorten: shortURL,
			Created: time.Now(),
		}

		urlObjects = append(urlObjects, urlObj)
		batchResponse = append(batchResponse, BatchShortenResponse{
			CorrelationID: item.CorrelationID,
			ShortURL:      shortFullURL,
		})
	}

	if h.Mode == "database" {
		if err := h.Repo.SaveBatchURLs(req.Context(), urlObjects); err != nil {
			h.Logger.Error("Ошибка сохранения batch URL в БД", zap.Error(err))
			http.Error(res, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	} else if h.Mode == "file" {
		for _, obj := range urlObjects {
			entry := model.Entry{ShortURL: obj.Shorten, OriginalURL: obj.Origin}
			if err := h.store.AppendToFile(entry); err != nil {
				log.Printf("Ошибка сохранения в файл: %v", err)
			}
			h.store.Save(obj.Shorten, obj.Origin)
		}
	} else {
		for _, obj := range urlObjects {
			if err := util.SaveURL(obj.Origin, obj.Shorten, h.store); err != nil {
				log.Printf("Ошибка сохранения в память: %v", err)
			}
		}
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	json.NewEncoder(res).Encode(batchResponse)
}

func (h *Handler) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	userID := h.Auth.GetOrSetUserID(w, r) // создаст куку, если её нет

	urlObjs, err := h.Repo.GetURLsByUserID(r.Context(), userID)
	if err != nil {
		h.Logger.Error("Ошибка получения URL пользователя", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if len(urlObjs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var response []UserURLResponse
	for _, u := range urlObjs {
		response = append(response, UserURLResponse{
			ShortURL:    fmt.Sprintf("%s/%s", h.baseURL, u.Shorten),
			OriginalURL: u.Origin,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
