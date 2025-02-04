package router

import (
	"github.com/Totarae/URLShortener/internal/handlers"
	"github.com/go-chi/chi/v5"
)

// NewRouter создаёт и настраивает маршрутизатор
func NewRouter(handler *handlers.Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/", handler.ReceiveURL)
	r.Get("/{id}", handler.ResponseURL)
	return r
}
