package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

var (
	urlStore = make(map[string]string) // Хранилище сокращённых URL
	mutex    = sync.RWMutex{}          // Мьютекс для безопасного доступа к хранилищу
)

func commonHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		responseURL(w, r)
	case http.MethodPost:
		receiveURL(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusBadRequest)
	}
}

func receiveURL(res http.ResponseWriter, req *http.Request) {

	if req.Method == http.MethodPost {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, "BadRequest", http.StatusBadRequest)
			return
		}

		originalURL := string(body)
		if originalURL == "" {
			http.Error(res, "URL empty", http.StatusBadRequest)
			return
		}

		shortURL := generateShortURL(originalURL)
		res.WriteHeader(http.StatusCreated)
		res.Header().Set("Content-Type", "text/plain")
		res.Write([]byte(shortURL))
		return
	}
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

	return "http://localhost:8080/" + hashString
}

func responseURL(res http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodGet {
		http.Error(res, "Method Not Allowed", http.StatusBadRequest)
		return
	}

	// Получаем id из пути Убираем первый `/`

	// Проверяем, что путь содержит идентификатор
	id := strings.TrimPrefix(req.URL.Path, "/")
	if id == "" {
		http.Error(res, "Bad Request: Missing ID in URL", http.StatusBadRequest)
		return
	}

	fmt.Println("ID : ", id)
	// Ищем оригинальный URL в хранилище
	originalURL, exists := urlStore[id]
	if !exists {
		http.NotFound(res, req)
		return
	}

	// Устанавливаем заголовок Location и код 307
	res.Header().Set("Location", originalURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("/", commonHandler)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}

}
