package storage_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Totarae/URLShortener/internal/model"
	"github.com/Totarae/URLShortener/internal/util"
	"github.com/stretchr/testify/assert"
)

// Тест сохранения и получения URL из памяти
func TestURLStore_SaveAndGet(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "store.json")
	store := util.NewURLStore(tmpFile)

	original := "https://yandex.ru"
	short := util.GenerateShortURL(original)

	store.Save(short, original)

	got, ok := store.Get(short)
	assert.True(t, ok)
	assert.Equal(t, original, got)
}

// Тест генерации короткого URL
func TestGenerateShortURL(t *testing.T) {
	url1 := "https://yandex.ru"
	url2 := "https://google.com"

	short1 := util.GenerateShortURL(url1)
	short2 := util.GenerateShortURL(url2)

	assert.NotEmpty(t, short1)
	assert.NotEmpty(t, short2)
	assert.NotEqual(t, short1, short2)
}

// Тест загрузки данных из файла
func TestURLStore_LoadFromFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "store.json")
	entry := model.Entry{ShortURL: "s123", OriginalURL: "https://mail.ru"}

	data, err := json.Marshal(entry)
	assert.NoError(t, err)

	_ = os.WriteFile(tmpFile, append(data, '\n'), 0644)

	store := util.NewURLStore(tmpFile)

	got, ok := store.Get("s123")
	assert.True(t, ok)
	assert.Equal(t, "https://mail.ru", got)
}

// Тест добавления записи в файл
func TestURLStore_AppendToFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "append.json")
	store := util.NewURLStore(tmpFile)

	entry := model.Entry{ShortURL: "x1", OriginalURL: "https://vk.com"}
	err := store.AppendToFile(entry)
	assert.NoError(t, err)

	content, err := os.ReadFile(tmpFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "vk.com")
}

// Тест обёртки SaveURL
func TestSaveURL(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "save.json")
	store := util.NewURLStore(tmpFile)

	original := "https://avito.ru"
	short := util.GenerateShortURL(original)

	err := util.SaveURL(original, short, store)
	assert.NoError(t, err)

	got, ok := store.Get(short)
	assert.True(t, ok)
	assert.Equal(t, original, got)
}
