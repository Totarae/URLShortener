package util

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"sync"
)

// URLStore provides a thread-safe URL storage
type URLStore struct {
	data  map[string]string
	mutex sync.RWMutex
}

type Storage interface {
	Save(short, original string)
	Get(short string) (string, bool)
}

// NewURLStore initializes a new URLStore
func NewURLStore() *URLStore {
	return &URLStore{
		data: make(map[string]string),
	}
}

// Save stores a shortened URL
func (s *URLStore) Save(short, original string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.data[short] = original
}

// Get retrieves the original URL by its short version
func (s *URLStore) Get(short string) (string, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	original, exists := s.data[short]
	return original, exists
}

// GenerateShortURL creates a shortened URL
func GenerateShortURL(originalURL string, baseURL string, store Storage) string {
	hash := sha256.Sum256([]byte(originalURL))
	hashString := base64.RawURLEncoding.EncodeToString(hash[:16])
	hashString = strings.ToLower(hashString) // Ensure lowercase for consistency

	store.Save(hashString, originalURL) // Store in the map

	return baseURL + "/" + hashString
}
