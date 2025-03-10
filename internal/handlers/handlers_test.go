package handlers

import (
	"context"
	"fmt"
	"github.com/Totarae/URLShortener/internal/mocks"
	"github.com/Totarae/URLShortener/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupMockHandler(t *testing.T, mockURL *mocks.MockURLRepositoryInterface, mockStore *mocks.MockStorage) *Handler {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	baseURL := "http://localhost:8080"

	return NewHandler(mockStore, baseURL, mockURL, logger)
}

func TestReceiveURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)

	// Ожидаем вызов `SaveURL`, если используется БД
	mockRepo.EXPECT().SaveURL(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	// Ожидаем вызов `Save`, ТОЛЬКО если используется in-memory store
	mockStore.EXPECT().Save(gomock.Any(), gomock.Any()).Times(0)

	h := setupMockHandler(t, mockRepo, mockStore)

	reqBody := "https://example.com"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	h.ReceiveURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.NotEmpty(t, body, "Ответ должен содержать короткий URL")
}

func TestReceiveURL_EmptyBody(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)

	h := setupMockHandler(t, mockRepo, mockStore)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	h.ReceiveURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestReceiveURL_WrongMethod(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)
	h := setupMockHandler(t, mockRepo, mockStore)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Print request details
	fmt.Printf("Request: Method=%s, URL=%s\n", req.Method, req.URL.String())
	for key, values := range req.Header {
		fmt.Printf("Header: %s=%v\n", key, values)
	}
	body, _ := io.ReadAll(req.Body)
	fmt.Printf("Body: %s\n", string(body))
	req.Body = io.NopCloser(strings.NewReader("")) // Restore body for processing

	w := httptest.NewRecorder()

	h.ReceiveURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

// TestResponseURL проверяет редирект на оригинальный URL
func TestResponseURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)

	shortID := "shortid"
	originalURL := "https://example.com"

	// База данных не находит URL, проверяем хранилище
	mockRepo.EXPECT().GetURL(gomock.Any(), shortID).Return(&model.URLObject{
		Origin:  originalURL,
		Shorten: shortID,
	}, nil).Times(1)

	h := setupMockHandler(t, mockRepo, mockStore)
	r := chi.NewRouter()
	r.Get("/{id}", h.ResponseURL)

	req := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", shortID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	assert.Equal(t, "https://example.com", resp.Header.Get("Location"))
}

func TestResponseURL_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)

	// Мокаем `GetURL`, который должен вернуть nil, nil (означает, что URL не найден)
	mockRepo.EXPECT().GetURL(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

	h := setupMockHandler(t, mockRepo, mockStore)
	r := chi.NewRouter()

	// Add the route to the router
	r.Get("/{id}", h.ResponseURL)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent") // Set the ID to "nonexistent"
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Serve the request using the Chi router
	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestResponseURL_WrongMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)

	h := setupMockHandler(t, mockRepo, mockStore)

	req := httptest.NewRequest(http.MethodPost, "/someid", nil)
	w := httptest.NewRecorder()

	h.ResponseURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestReceiveShorten(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)

	// Ожидаем вызов `SaveURL`, если используется БД
	mockRepo.EXPECT().SaveURL(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	h := setupMockHandler(t, mockRepo, mockStore)
	reqBody := `{"url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.ReceiveShorten(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}
}

func TestReceiveShorten_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)

	h := setupMockHandler(t, mockRepo, mockStore)
	reqBody := `{"invalid":"data"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.ReceiveShorten(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}
