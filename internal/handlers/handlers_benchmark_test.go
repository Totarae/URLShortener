package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Totarae/URLShortener/internal/auth"
	"github.com/Totarae/URLShortener/internal/handlers"
	"github.com/Totarae/URLShortener/internal/model"
	"github.com/Totarae/URLShortener/internal/util"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type mockRepo struct{}

func (m *mockRepo) CountURLs(ctx context.Context) (int, error) {
	return 42, nil
}

func (m *mockRepo) CountUsers(ctx context.Context) (int, error) {
	return 7, nil
}

func (m *mockRepo) SaveURL(ctx context.Context, u *model.URLObject) error {
	return nil
}
func (m *mockRepo) SaveBatchURLs(ctx context.Context, u []*model.URLObject) error {
	return nil
}
func (m *mockRepo) GetURL(ctx context.Context, short string) (*model.URLObject, error) {
	return &model.URLObject{Origin: "https://yandex.ru", Shorten: short}, nil
}
func (m *mockRepo) GetShortURLByOrigin(ctx context.Context, origin string) (string, error) {
	return "abc123", nil
}
func (m *mockRepo) GetURLsByUserID(ctx context.Context, userID string) ([]*model.URLObject, error) {
	return []*model.URLObject{
		{Origin: "https://yandex.ru/1", Shorten: "a1"},
		{Origin: "https://yandex.ru/2", Shorten: "a2"},
	}, nil
}
func (m *mockRepo) MarkURLsAsDeleted(ctx context.Context, ids []string, userID string) error {
	return nil
}
func (m *mockRepo) Ping(ctx context.Context) error {
	return nil
}

func setupTestHandler() *handlers.Handler {
	tmpFile := filepath.Join(os.TempDir(), "bench_data.json")
	store := util.NewURLStore(tmpFile)

	repo := &mockRepo{}
	logger, _ := zap.NewDevelopment()
	authService := auth.New("bench-secret")

	return handlers.NewHandler(store, "http://localhost:8080", repo, logger, "file", authService, nil)
}

func BenchmarkReceiveShorten(b *testing.B) {
	handler := setupTestHandler()
	body := `{"url": "https://yandex.ru/benchmark"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ReceiveShorten(rec, req.Clone(context.Background()))
	}
}

func BenchmarkBatchShortenHandler(b *testing.B) {
	handler := setupTestHandler()

	var batchBuilder strings.Builder
	batchBuilder.WriteString("[")
	for i := 0; i < 10; i++ {
		if i > 0 {
			batchBuilder.WriteString(",")
		}
		fmt.Fprintf(&batchBuilder,
			`{"correlation_id":"id%d","original_url":"https://yandex.ru/%d"}`, i, i)
	}
	batchBuilder.WriteString("]")

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(batchBuilder.String()))
	req.Header.Set("Content-Type", "application/json")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.BatchShortenHandler(rec, req.Clone(context.Background()))
	}
}

func BenchmarkGetUserURLs(b *testing.B) {
	handler := setupTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.GetUserURLs(rec, req.Clone(context.Background()))
	}
}

func BenchmarkDeleteUserURLs(b *testing.B) {
	handler := setupTestHandler()

	body := `["abc123", "def456", "ghi789"]`
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.DeleteUserURLs(rec, req.Clone(context.Background()))
	}
}

func BenchmarkResponseURL(b *testing.B) {
	handler := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	// Добавляем chi-параметр вручную
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("id", "abc123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ResponseURL(rec, req.Clone(context.Background()))
	}
}

func ExampleHandler_ReceiveShorten() {
	handler := setupTestHandler()
	body := `{"url": "https://yandex.ru"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ReceiveShorten(rec, req)

	fmt.Println(rec.Code == http.StatusCreated)

	// Output:
	// true
}
