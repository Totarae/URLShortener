package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Totarae/URLShortener/internal/auth"
	"github.com/Totarae/URLShortener/internal/mocks"
	"github.com/Totarae/URLShortener/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func setupMockHandler(t *testing.T, mockURL *mocks.MockURLRepositoryInterface, mockStore *mocks.MockStorage, mode string) *Handler {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	baseURL := "http://localhost:8080"

	authService := auth.New("test-secret") // используем простой секрет для теста

	return NewHandler(mockStore, baseURL, mockURL, logger, mode, authService)
}

func TestReceiveURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)

	// Ожидаем вызов `SaveURL`, если используется БД
	mockRepo.EXPECT().SaveURL(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	// Ожидаем вызов `Save`, ТОЛЬКО если используется in-memory store
	//mockStore.EXPECT().Save(gomock.Any(), gomock.Any()).Times(0)

	h := setupMockHandler(t, mockRepo, mockStore, "database")

	reqBody := "https://example.com"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	h.ReceiveURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Проверяем, что кука установлена
	setCookie := resp.Header.Get("Set-Cookie")
	assert.NotEmpty(t, setCookie, "Ожидалась установка Set-Cookie в ответе")
	assert.Contains(t, setCookie, "auth_token=", "Ответ должен содержать auth_token в Set-Cookie")

	body, _ := io.ReadAll(resp.Body)
	assert.NotEmpty(t, body, "Ответ должен содержать короткий URL")
}

func TestReceiveURL_EmptyBody(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)

	h := setupMockHandler(t, mockRepo, mockStore, "database")

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
	h := setupMockHandler(t, mockRepo, mockStore, "database")

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

	h := setupMockHandler(t, mockRepo, mockStore, "database")
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

	h := setupMockHandler(t, mockRepo, mockStore, "database")
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

	h := setupMockHandler(t, mockRepo, mockStore, "database")

	req := httptest.NewRequest(http.MethodPost, "/someid", nil)
	w := httptest.NewRecorder()

	h.ResponseURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestResponseURL_Deleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)
	h := setupMockHandler(t, mockRepo, mockStore, "database")

	shortID := "dead123"

	mockRepo.EXPECT().GetURL(gomock.Any(), shortID).Return(&model.URLObject{
		Shorten:   shortID,
		Origin:    "https://example.com/deleted",
		IsDeleted: true,
	}, nil).Times(1)

	r := chi.NewRouter()
	r.Get("/{id}", h.ResponseURL)

	req := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", shortID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusGone, resp.StatusCode)
}

func TestReceiveShorten(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)

	// Ожидаем вызов `SaveURL`, если используется БД
	mockRepo.EXPECT().SaveURL(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	h := setupMockHandler(t, mockRepo, mockStore, "database")
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

	h := setupMockHandler(t, mockRepo, mockStore, "database")
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

func TestGetUserURLs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)
	h := setupMockHandler(t, mockRepo, mockStore, "database")

	userID := "test-user-id"
	signedCookie := h.Auth.SignCookieValue(userID) // auth_token в формате userID:signature

	expectedURLs := []*model.URLObject{
		{Shorten: "abc123", Origin: "https://example.com"},
		{Shorten: "xyz789", Origin: "https://golang.org"},
	}

	mockRepo.EXPECT().GetURLsByUserID(gomock.Any(), userID).Return(expectedURLs, nil).Times(1)

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: signedCookie,
	})

	w := httptest.NewRecorder()
	h.GetUserURLs(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "example.com")
	assert.Contains(t, string(body), "golang.org")
}

func TestGetUserURLs_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)
	h := setupMockHandler(t, mockRepo, mockStore, "database")

	// Ожидаем, что вызов GetURLsByUserID произойдёт с новым userID
	mockRepo.EXPECT().GetURLsByUserID(gomock.Any(), gomock.Any()).Return([]*model.URLObject{}, nil).Times(1)

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	w := httptest.NewRecorder()

	h.GetUserURLs(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Проверим, что кука действительно установлена
	setCookie := resp.Header.Get("Set-Cookie")
	assert.NotEmpty(t, setCookie, "Ожидалась установка новой auth_token куки")
}

func TestGetUserURLs_InvalidCookie(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)
	h := setupMockHandler(t, mockRepo, mockStore, "database")

	// Ожидаем, что кука будет проигнорирована, и будет создан новый userID
	mockRepo.EXPECT().GetURLsByUserID(gomock.Any(), gomock.Any()).Return([]*model.URLObject{}, nil).Times(1)

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "someuserid:invalidsignature",
	})

	w := httptest.NewRecorder()
	h.GetUserURLs(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	setCookie := resp.Header.Get("Set-Cookie")
	assert.NotEmpty(t, setCookie, "Ожидалась переустановка куки с новым user_id")
}

func TestGetUserURLs_NoContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)
	h := setupMockHandler(t, mockRepo, mockStore, "database")

	userID := "test-user-id"
	signedCookie := h.Auth.SignCookieValue(userID)

	// Возвращаем пустой список
	mockRepo.EXPECT().GetURLsByUserID(gomock.Any(), userID).Return([]*model.URLObject{}, nil).Times(1)

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: signedCookie,
	})

	w := httptest.NewRecorder()
	h.GetUserURLs(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestDeleteUserURLs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)
	h := setupMockHandler(t, mockRepo, mockStore, "database")

	userID := "user-delete-test"
	signedCookie := h.Auth.SignCookieValue(userID)
	body := `["id1", "id2", "id3"]`

	mockRepo.EXPECT().
		MarkURLsAsDeleted(gomock.Any(), gomock.Any(), userID).
		Return(nil).
		Times(1)

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: signedCookie})
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.DeleteUserURLs(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	time.Sleep(50 * time.Millisecond) // задержка для ожидания горутины
}

func TestBatchShortenHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockURLRepositoryInterface(ctrl)
	mockStore := mocks.NewMockStorage(ctrl)
	handler := setupMockHandler(t, mockRepo, mockStore, "database")

	input := `[{"correlation_id":"abc","original_url":"https://yandex.ru"},{"correlation_id":"def","original_url":"https://google.com"}]`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(input))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Ожидаем вызов SaveBatchURLs
	mockRepo.EXPECT().
		SaveBatchURLs(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	handler.BatchShortenHandler(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Contains(t, string(body), "short_url")
}
