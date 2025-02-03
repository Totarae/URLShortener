package handlers

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReceiveURL(t *testing.T) {
	reqBody := "https://example.com"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	ReceiveURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}
}

func TestReceiveURL_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	ReceiveURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestReceiveURL_WrongMethod(t *testing.T) {
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

	ReceiveURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

// TestResponseURL проверяет редирект на оригинальный URL
func TestResponseURL(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/{id}", ResponseURL)

	// Устанавливаем baseURL (иначе generateShortURL выдаст неверный путь)
	SetBaseURL("http://localhost:8080")

	// Генерируем сокращенный URL
	shortID := generateShortURL("https://example.com")
	shortPath := strings.TrimPrefix(shortID, "http://localhost:8080/")

	// Добавляем в хранилище вручную
	mutex.Lock()
	urlStore[shortPath] = "https://example.com"
	mutex.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/"+shortPath, nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", shortPath)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Errorf("expected status %d, got %d", http.StatusTemporaryRedirect, resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "https://example.com" {
		t.Errorf("expected redirect to %s, got %s", "https://example.com", location)
	}
}

func TestResponseURL_NotFound(t *testing.T) {
	r := chi.NewRouter()

	// Add the route to the router
	r.Get("/{id}", ResponseURL)

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
	req := httptest.NewRequest(http.MethodPost, "/someid", nil)
	w := httptest.NewRecorder()

	ResponseURL(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}
