package handlers

import (
	_ "bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Totarae/URLShortener/internal/model"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/Totarae/URLShortener/internal/auth"
	"github.com/Totarae/URLShortener/internal/util"
	"go.uber.org/zap"
)

type mockRepo struct{}

func (m *mockRepo) SaveURL(ctx context.Context, u *model.URLObject) error         { return nil }
func (m *mockRepo) SaveBatchURLs(ctx context.Context, u []*model.URLObject) error { return nil }
func (m *mockRepo) GetURL(ctx context.Context, short string) (*model.URLObject, error) {
	return nil, nil
}
func (m *mockRepo) GetShortURLByOrigin(ctx context.Context, origin string) (string, error) {
	return "abc123", nil
}
func (m *mockRepo) GetURLsByUserID(ctx context.Context, userID string) ([]*model.URLObject, error) {
	return nil, nil
}
func (m *mockRepo) MarkURLsAsDeleted(ctx context.Context, ids []string, userID string) error {
	return nil
}
func (m *mockRepo) Ping(ctx context.Context) error              { return nil }
func (m *mockRepo) CountURLs(ctx context.Context) (int, error)  { return 0, nil }
func (m *mockRepo) CountUsers(ctx context.Context) (int, error) { return 0, nil }

// ExampleHandler_ReceiveShorten демонстрирует работу метода ReceiveShorten.
func ExampleHandler_ReceiveShorten() {
	store := util.NewURLStore("")
	logger, _ := zap.NewDevelopment()
	repo := &mockRepo{}
	authService := auth.New("example-secret")

	h := NewHandler(store, "http://localhost", repo, logger, "memory", authService, nil)

	body := `{"url":"https://yandex.ru"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ReceiveShorten(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()

	var result map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&result)

	fmt.Println(resp.StatusCode)
	fmt.Println(strings.HasPrefix(result["result"], "http://localhost/"))

	// Output:
	// 201
	// true
}
