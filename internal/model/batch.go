package model

// BatchShortenRequest представляет одну запись в пакетном запросе на сокращение URL.
type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchShortenResponse представляет одну запись в пакетном ответе.
type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// BatchItem Внутренние структуры
type BatchItem struct {
	CorrelationID string
	OriginalURL   string
}

// BatchResult Внутренние структуры
type BatchResult struct {
	CorrelationID string
	ShortURL      string
	OriginalURL   string
}
