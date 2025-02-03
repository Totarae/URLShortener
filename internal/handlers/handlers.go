package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/go-chi/chi/v5"
	_ "github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"strings"
	"sync"
)

var (
	urlStore = make(map[string]string) // Хранилище сокращённых URL
	mutex    = sync.RWMutex{}          // Мьютекс для безопасного доступа к хранилищу
	baseURL  string
)

// SetBaseURL устанавливает базовый URL для сокращённых ссылок
func SetBaseURL(url string) {
	baseURL = strings.TrimSuffix(url, "/")
	fmt.Printf("Base URL: %s\n", baseURL)
}

func ReceiveURL(res http.ResponseWriter, req *http.Request) {

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "BadRequest", http.StatusBadRequest)
		return
	}

	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(res, "URL empty", http.StatusBadRequest)
		return
	}

	shortURL := generateShortURL(originalURL)
	res.WriteHeader(http.StatusCreated)
	res.Header().Set("Content-Type", "text/plain")
	res.Write([]byte(shortURL))
}

func generateShortURL(originalURL string) string {

	hash := sha256.Sum256([]byte(originalURL))
	fmt.Printf("SHA256 hash: %x\n", hash)
	hashString := base64.RawURLEncoding.EncodeToString(hash[:16])
	hashString = strings.ToLower(hashString)
	// первые 16 байт, без паддинга, url-safe
	fmt.Println("SHA256 (Base64):", hashString)

	// Сохраняем в хранилище
	mutex.Lock()
	urlStore[hashString] = originalURL
	mutex.Unlock()

	return baseURL + "/" + hashString
}

func ResponseURL(res http.ResponseWriter, req *http.Request) {

	id := chi.URLParam(req, "id")
	if id == "" {
		http.Error(res, "Bad Request: Missing ID in URL", http.StatusBadRequest)
		return
	}

	/*if req.Method != http.MethodGet {
		http.Error(res, "Method Not Allowed", http.StatusBadRequest)
		return
	}*/

	// Получаем id из пути Убираем первый `/`

	// Проверяем, что путь содержит идентификатор
	/*id := strings.TrimPrefix(req.URL.Path, "/")
	if id == "" {
		http.Error(res, "Bad Request: Missing ID in URL", http.StatusBadRequest)
		return
	}*/

	fmt.Println("Incoming ID : ", id)
	// Ищем оригинальный URL в хранилище
	//originalURL, exists := urlStore[id]
	mutex.RLock()
	originalURL, exists := urlStore[id]
	mutex.RUnlock()
	if !exists {
		http.NotFound(res, req)
		return
	}

	// Устанавливаем заголовок Location и код 307
	res.Header().Set("Location", originalURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}
