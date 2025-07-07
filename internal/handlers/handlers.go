package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Totarae/URLShortener/internal/auth"
	"github.com/Totarae/URLShortener/internal/model"
	"github.com/Totarae/URLShortener/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Handler содержит зависимости и реализует HTTP-обработчики
// для операций с сокращёнными ссылками (создание, получение, удаление и т.д.).
type Handler struct {
	Service       *service.ShortenerService
	Logger        *zap.Logger
	Auth          *auth.Auth
	TrustedSubnet *net.IPNet
}

// UserURLResponse представляет пару оригинального и сокращённого URL пользователя.
type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

var validIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{6,22}$`)

// NewHandler создаёт новый экземпляр Handler с заданным хранилищем, базовым URL,
// реализацией репозитория, логгером, режимом работы (file/database) и сервисом аутентификации.
func NewHandler(svc *service.ShortenerService, logger *zap.Logger, authService *auth.Auth, trustedSubnet *net.IPNet) *Handler {
	return &Handler{
		Service:       svc,
		Logger:        logger,
		Auth:          authService,
		TrustedSubnet: trustedSubnet,
	}
}

// ReceiveURL принимает plain-текст ссылку в теле запроса,
// генерирует сокращённый вариант и возвращает его в ответе.
func (h *Handler) ReceiveURL(res http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil || len(body) == 0 {
		http.Error(res, "Invalid body", http.StatusBadRequest)
		return
	}
	originalURL := strings.TrimSpace(string(body))
	parsedURL, err := url.ParseRequestURI(originalURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		http.Error(res, "Invalid URL", http.StatusBadRequest)
		return
	}

	userID := h.Auth.GetOrSetUserID(res, req)
	short, err := h.Service.ShortenURL(req.Context(), userID, originalURL)
	if err != nil {
		h.Logger.Error("Shorten error", zap.Error(err))
		http.Error(res, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	shortURL := fmt.Sprintf("%s/%s", h.Service.BaseURL, short)
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	res.Write([]byte(shortURL))
}

// ResponseURL перенаправляет по сокращённому идентификатору на оригинальный URL,
// если он существует и не удалён.
func (h *Handler) ResponseURL(res http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "id")
	if id == "" {
		http.Error(res, "Missing ID", http.StatusBadRequest)
		return
	}

	urlObj, err := h.Service.ResolveURL(req.Context(), id)
	if err != nil {
		h.Logger.Error("Resolve error", zap.Error(err))
		http.Error(res, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if urlObj == nil {
		http.NotFound(res, req)
		return
	}
	if urlObj.IsDeleted {
		http.Error(res, "Gone", http.StatusGone)
		return
	}

	res.Header().Set("Location", urlObj.Origin)
	res.WriteHeader(http.StatusTemporaryRedirect)
}

// ReceiveShorten принимает JSON-запрос с оригинальным URL,
// сохраняет и возвращает сокращённый URL в формате JSON.
func (h *Handler) ReceiveShorten(res http.ResponseWriter, req *http.Request) {
	var request model.ShortenRequest
	if err := json.NewDecoder(req.Body).Decode(&request); err != nil || request.URL == "" {
		http.Error(res, "Invalid JSON", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.ParseRequestURI(request.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		http.Error(res, "Invalid URL", http.StatusBadRequest)
		return
	}

	userID := h.Auth.GetOrSetUserID(res, req)
	short, err := h.Service.ShortenURL(req.Context(), userID, request.URL)
	if err != nil {
		h.Logger.Error("Shorten error", zap.Error(err))
		http.Error(res, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	result := model.ShortenResponse{Result: fmt.Sprintf("%s/%s", h.Service.BaseURL, short)}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	json.NewEncoder(res).Encode(result)
}

// PingHandler выполняет проверку подключения к базе данных.
// Возвращает 200 OK, если соединение активно.
func (h *Handler) PingHandler(res http.ResponseWriter, req *http.Request) {
	if err := h.Service.Ping(req.Context()); err != nil {
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

	var batchReq []model.BatchShortenRequest
	if err := json.NewDecoder(req.Body).Decode(&batchReq); err != nil {
		http.Error(res, "Invalid JSON", http.StatusBadRequest)
		return
	}

	userID := h.Auth.GetOrSetUserID(res, req)
	items := make([]model.BatchItem, 0, len(batchReq))
	for _, r := range batchReq {
		items = append(items, model.BatchItem{
			CorrelationID: r.CorrelationID,
			OriginalURL:   r.OriginalURL,
		})
	}

	results, err := h.Service.CreateBatchShortURLs(req.Context(), userID, items)
	if err != nil {
		h.Logger.Error("Batch shorten error", zap.Error(err))
		http.Error(res, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	batchResp := make([]model.BatchShortenResponse, 0, len(results))
	for _, r := range results {
		batchResp = append(batchResp, model.BatchShortenResponse{
			CorrelationID: r.CorrelationID,
			ShortURL:      fmt.Sprintf("%s/%s", h.Service.BaseURL, r.ShortURL),
		})
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	json.NewEncoder(res).Encode(batchResp)
}

// DeleteUserURLs помечает заданные ссылки пользователя как удалённые.
// Принимает JSON-массив сокращённых ID в теле запроса.
func (h *Handler) DeleteUserURLs(res http.ResponseWriter, req *http.Request) {
	userID := h.Auth.GetOrSetUserID(res, req)
	var ids []string
	if err := json.NewDecoder(req.Body).Decode(&ids); err != nil {
		http.Error(res, "Invalid JSON", http.StatusBadRequest)
		return
	}
	go h.Service.DeleteURLs(context.Background(), userID, ids)
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

	urlCount, userCount, err := h.Service.GetStats(r.Context())
	if err != nil {
		h.Logger.Error("Failed to get stats", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := map[string]int{
		"urls":  urlCount,
		"users": userCount,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetUserURLs(res http.ResponseWriter, req *http.Request) {
	userID := h.Auth.GetOrSetUserID(res, req)
	results, err := h.Service.GetUserURLs(req.Context(), userID)
	if err != nil {
		h.Logger.Error("GetUserURLs error", zap.Error(err))
		http.Error(res, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if len(results) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}
	resp := make([]UserURLResponse, 0, len(results))
	for _, r := range results {
		resp = append(resp, UserURLResponse{
			ShortURL:    fmt.Sprintf("%s/%s", h.Service.BaseURL, r.ShortURL),
			OriginalURL: r.OriginalURL,
		})
	}
	res.Header().Set("Content-Type", "application/json")
	json.NewEncoder(res).Encode(resp)
}
