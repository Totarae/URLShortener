package model

// ShortenRequest представляет структуру запроса на сокращение URL.
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse представляет структуру ответа с сокращённым URL.
type ShortenResponse struct {
	Result string `json:"result"`
}
