package router

import (
	"github.com/Totarae/URLShortener/internal/handlers"
	"github.com/Totarae/URLShortener/internal/middleware"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// NewRouter создаёт и настраивает маршрутизатор
func NewRouter(handler *handlers.Handler, logger *zap.Logger) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.LoggingMiddleware(logger)) // Подключаем логирование
	r.Use(middleware.GzipMiddleware)            // Gzip-сжатие

	r.Post("/", handler.ReceiveURL)
	r.Post("/api/shorten", handler.ReceiveShorten)
	r.Get("/{id}", handler.ResponseURL)
	return r
}
