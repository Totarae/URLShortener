package storage

import (
	"github.com/Totarae/URLShortener/internal/model"
)

// Storage определяет интерфейс для работы с хранилищем URL.
type Storage interface {
	// Save сохраняет сопоставление короткой и оригинальной ссылки.
	Save(short, original, userID string)
	// Get возвращает оригинальный URL по короткому идентификатору.
	Get(short string) (string, bool)
	// AppendToFile добавляет запись в файл (для file-based хранилищ).
	AppendToFile(entry model.Entry) error
	// GetByUser ищем по user id .
	GetByUser(userID string) map[string]string
	// MarkDeleted удаляем по user id .
	MarkDeleted(shortenIDs []string, userID string)
}
