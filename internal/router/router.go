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

	r.Get("/{id}", handler.ResponseURL)
	r.Get("/ping", handler.PingHandler) // Проверка соединения с БД

	r.Route("/api/shorten", func(r chi.Router) {
		r.Post("/", handler.ReceiveShorten)
		r.Post("/batch", handler.BatchShortenHandler)
	})

	// 🔐 Защищённый маршрут — только для авторизованных пользователей
	r.Get("/api/user/urls", handler.GetUserURLs)
	r.Delete("/api/user/urls", handler.DeleteUserURLs)

	return r
}
