package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Totarae/URLShortener/internal/auth"
	"github.com/Totarae/URLShortener/internal/model"
	"github.com/Totarae/URLShortener/internal/repositories"
	"github.com/Totarae/URLShortener/internal/storage"
	"github.com/Totarae/URLShortener/internal/util"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// Handler содержит зависимости и реализует HTTP-обработчики
// для операций с сокращёнными ссылками (создание, получение, удаление и т.д.).
type Handler struct {
	store         storage.Storage // Use the new URLStore for thread safety
	baseURL       string
	Repo          repositories.URLRepositoryInterface
	Logger        *zap.Logger
	Mode          string
	Auth          *auth.Auth
	TrustedSubnet *net.IPNet
}

// ShortenRequest представляет структуру запроса на сокращение URL.
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse представляет структуру ответа с сокращённым URL.
type ShortenResponse struct {
	Result string `json:"result"`
}

// BatchShortenRequest представляет одну запись в пакетном запросе на сокращение URL.
type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchShortenResponse представляет одну запись в пакетном ответе.
type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// UserURLResponse представляет пару оригинального и сокращённого URL пользователя.
type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

var validIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{6,22}$`)

// NewHandler создаёт новый экземпляр Handler с заданным хранилищем, базовым URL,
// реализацией репозитория, логгером, режимом работы (file/database) и сервисом аутентификации.
func NewHandler(store storage.Storage, baseURL string, repo repositories.URLRepositoryInterface, logger *zap.Logger,
	mode string, authService *auth.Auth, trustedSubnet *net.IPNet) *Handler {
	return &Handler{
		store:         store,
		baseURL:       strings.TrimSuffix(baseURL, "/"),
		Repo:          repo,
		Logger:        logger,
		Mode:          mode,
		Auth:          authService,
		TrustedSubnet: trustedSubnet,
	}
}

// ReceiveURL принимает plain-текст ссылку в теле запроса,
// генерирует сокращённый вариант и возвращает его в ответе.
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

// ResponseURL перенаправляет по сокращённому идентификатору на оригинальный URL,
// если он существует и не удалён.
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
		if urlObj.IsDeleted {
			http.Error(res, "gone", http.StatusGone)
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

// ReceiveShorten принимает JSON-запрос с оригинальным URL,
// сохраняет и возвращает сокращённый URL в формате JSON.
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

// PingHandler выполняет проверку подключения к базе данных.
// Возвращает 200 OK, если соединение активно.
func (h *Handler) PingHandler(res http.ResponseWriter, req *http.Request) {
	if err := h.Repo.Ping(req.Context()); err != nil {
		h.Logger.Error("Database ping failed", zap.Error(err))
		http.Error(res, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
	res.Write([]byte("OK"))
}

// BatchShortenHandler обрабатывает пакетный JSON-запрос со множеством ссылок,
// и возвращает массив сокращённых ссылок с их корреляционными ID.
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

	batchResponse := make([]BatchShortenResponse, 0, len(batchRequest))
	urlObjects := make([]*model.URLObject, 0, len(batchRequest))

	for _, item := range batchRequest {
		originalURL := strings.TrimSpace(item.OriginalURL)
		parsedURL, err := url.ParseRequestURI(originalURL)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			continue
		}

		shortURL := util.GenerateShortURL(originalURL)

		urlObjects = append(urlObjects, &model.URLObject{
			Origin:  originalURL,
			Shorten: shortURL,
			Created: time.Now(),
		})
		batchResponse = append(batchResponse, BatchShortenResponse{
			CorrelationID: item.CorrelationID,
			ShortURL:      h.baseURL + "/" + shortURL,
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
				h.Logger.Error("Ошибка сохранения в память: %v", zap.Error(err))
			}
		}
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	json.NewEncoder(res).Encode(batchResponse)
}

// GetUserURLs возвращает все активные сокращённые ссылки пользователя
// в формате JSON. Использует идентификатор пользователя из cookie.
func (h *Handler) GetUserURLs(res http.ResponseWriter, req *http.Request) {
	userID := h.Auth.GetOrSetUserID(res, req) // создаст куку, если её нет

	urlObjs, err := h.Repo.GetURLsByUserID(req.Context(), userID)
	if err != nil {
		h.Logger.Error("Ошибка получения URL пользователя", zap.Error(err))
		http.Error(res, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if len(urlObjs) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	response := make([]UserURLResponse, 0, len(urlObjs))
	for _, u := range urlObjs {
		response = append(response, UserURLResponse{
			ShortURL:    fmt.Sprintf("%s/%s", h.baseURL, u.Shorten),
			OriginalURL: u.Origin,
		})
	}

	res.Header().Set("Content-Type", "application/json")
	json.NewEncoder(res).Encode(response)
}

// DeleteUserURLs помечает заданные ссылки пользователя как удалённые.
// Принимает JSON-массив сокращённых ID в теле запроса.
func (h *Handler) DeleteUserURLs(res http.ResponseWriter, req *http.Request) {
	userID := h.Auth.GetOrSetUserID(res, req)

	var shortenIDs []string
	if err := json.NewDecoder(req.Body).Decode(&shortenIDs); err != nil {
		http.Error(res, "invalid request body", http.StatusBadRequest)
		return
	}

	go func(ids []string, userID string) {
		ctx := context.Background() // безопасный независимый контекст
		const batchSize = 100
		for i := 0; i < len(ids); i += batchSize {
			end := i + batchSize
			if end > len(ids) {
				end = len(ids)
			}
			batch := ids[i:end]

			if err := h.Repo.MarkURLsAsDeleted(ctx, batch, userID); err != nil {
				h.Logger.Error("Ошибка при пометке URL как удалённых", zap.Error(err))
			}
		}
	}(shortenIDs, userID)

	res.WriteHeader(http.StatusAccepted)
}

// GetStatsHandler для статистики
func (h *Handler) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	ipStr := r.Header.Get("X-Real-IP")
	if ipStr == "" || h.TrustedSubnet == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	ip := net.ParseIP(ipStr)
	if ip == nil || !h.TrustedSubnet.Contains(ip) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	urlCount, err := h.Repo.CountURLs(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	userCount, err := h.Repo.CountUsers(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := map[string]int{
		"urls":  urlCount,
		"users": userCount,
	}
	json.NewEncoder(w).Encode(resp)
}
func (h *Handler) Store() storage.Storage {
	return h.store
}
