package util

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/Totarae/URLShortener/internal/model"
	"github.com/Totarae/URLShortener/internal/storage"
)

// URLStore provides a thread-safe URL storage
type URLStore struct {
	data  map[string]string
	mutex sync.RWMutex
	file  string
}

// NewURLStore initializes a new URLStore
func NewURLStore(file string) *URLStore {
	store := &URLStore{
		data: make(map[string]string),
		file: file,
	}

	// Загружаем данные из файла
	if err := store.LoadFromFile(); err != nil {
		log.Printf("Ошибка загрузки из файла: %v", err)
	}

	return store
}

// Save stores a shortened URL
func (s *URLStore) Save(short, original string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.data[short] = original

	// Сохраняем в файл
	entry := model.Entry{ShortURL: short, OriginalURL: original}
	if err := s.AppendToFile(entry); err != nil {
		log.Printf("Ошибка сохранения в файл: %v", err)
	}
}

// Get retrieves the original URL by its short version
func (s *URLStore) Get(short string) (string, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	original, exists := s.data[short]
	return original, exists
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
	store.Save(storeURL, originalURL) // Store in the map
	return nil
}

// LoadFromFile загружает данные из файла при старте сервера
func (s *URLStore) LoadFromFile() error {
	file, err := os.Open(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Файл ещё не создан, это не ошибка
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
		s.data[entry.ShortURL] = entry.OriginalURL
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
	for short, original := range s.data {
		entry := model.Entry{
			ShortURL:    short,
			OriginalURL: original,
		}
		if err := encoder.Encode(entry); err != nil {
			return err
		}
	}

	log.Printf("Сохранено %d URL-адресов в файл %s", len(s.data), s.file)
	return nil
}
