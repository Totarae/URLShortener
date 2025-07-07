package util

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/Totarae/URLShortener/internal/model"
	"github.com/Totarae/URLShortener/internal/storage"
)

// URLStore provides a thread-safe URL storage
type URLStore struct {
	data  map[string]model.Entry
	mutex sync.RWMutex
	file  string
}

// NewURLStore initializes a new URLStore
func NewURLStore(file string) *URLStore {
	store := &URLStore{
		data: make(map[string]model.Entry),
		file: file,
	}

	// Загружаем данные из файла
	if err := store.LoadFromFile(); err != nil {
		log.Printf("Ошибка загрузки из файла: %v", err)
	}

	return store
}

// Save stores a shortened URL
func (s *URLStore) Save(short, original, userID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	entry := model.Entry{
		ShortURL:    short,
		OriginalURL: original,
		UserID:      userID,
		IsDeleted:   false,
	}
	s.data[short] = entry

	if err := s.AppendToFile(entry); err != nil {
		log.Printf("Ошибка сохранения в файл: %v", err)
	}
}

// Get retrieves the original URL by its short version
func (s *URLStore) Get(short string) (string, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	entry, exists := s.data[short]
	if !exists || entry.IsDeleted {
		return "", false
	}
	return entry.OriginalURL, true
}

// GenerateShortURL creates a shortened URL
func GenerateShortURL(originalURL string) string {
	hash := sha256.Sum256([]byte(originalURL))
	hashString := base64.RawURLEncoding.EncodeToString(hash[:16])
	hashString = strings.ToLower(hashString) // Ensure lowercase for consistency

	return hashString
}

// SaveURL Сохранить URL в памяти
func SaveURL(originalURL string, storeURL string, store storage.Storage) error {
	store.Save(storeURL, originalURL, "unknown") // Store in the map
	return nil
}

// LoadFromFile загружает данные из файла при старте сервера
func (s *URLStore) LoadFromFile() error {
	file, err := os.Open(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	for {
		var entry model.Entry
		if err := decoder.Decode(&entry); err != nil {
			break
		}
		s.data[entry.ShortURL] = entry
	}
	log.Printf("Загружено %d URL-адресов из файла %s", len(s.data), s.file)
	return nil
}

// AppendToFile добавляет новую запись в файл
func (s *URLStore) AppendToFile(entry model.Entry) error {
	file, err := os.OpenFile(s.file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = file.WriteString(string(data) + "\n") // Записываем с новой строки
	return err
}

// SaveToFile перезаписывает весь файл данными из памяти
func (s *URLStore) SaveToFile() error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	file, err := os.Create(s.file)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, entry := range s.data {
		if err := encoder.Encode(entry); err != nil {
			return err
		}
	}
	log.Printf("Сохранено %d URL-адресов в файл %s", len(s.data), s.file)
	return nil
}

// ValidateURL проверяет, что строка — это корректный URL с http/https схемой и хостом.
func ValidateURL(raw string) (string, error) {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid URL")
	}
	return raw, nil
}

func (s *URLStore) GetByUser(userID string) map[string]string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make(map[string]string)
	for short, entry := range s.data {
		if entry.UserID == userID && !entry.IsDeleted {
			result[short] = entry.OriginalURL
		}
	}
	return result
}

func (s *URLStore) MarkDeleted(shortenIDs []string, userID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, id := range shortenIDs {
		entry, exists := s.data[id]
		if exists && entry.UserID == userID {
			entry.IsDeleted = true
			s.data[id] = entry
		}
	}
}
