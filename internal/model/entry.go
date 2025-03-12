package model

// Entry представляет структуру записи URL в файле
type Entry struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
