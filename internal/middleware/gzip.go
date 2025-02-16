package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// gzipResponseWriter оборачивает ResponseWriter и использует gzip.Writer для сжатия ответа
type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.Writer.Write(b)
}

// GzipMiddleware добавляет поддержку gzip для входящих и исходящих данных
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Распаковываем входящие gzip-запросы
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Unable to decompress request", http.StatusBadRequest)
				return
			}
			defer reader.Close()
			r.Body = reader
		}

		// Проверяем, поддерживает ли клиент gzip-ответ
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Устанавливаем заголовки, что ответ будет сжат
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")

		// Создаём gzip.Writer
		gzipWriter := gzip.NewWriter(w)
		defer gzipWriter.Close()

		// Оборачиваем ResponseWriter
		gzRespWriter := &gzipResponseWriter{ResponseWriter: w, Writer: gzipWriter}
		next.ServeHTTP(gzRespWriter, r)
	})
}
