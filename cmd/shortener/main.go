package main

import (
	"fmt"
	"github.com/Totarae/URLShortener/internal/config"
	"github.com/Totarae/URLShortener/internal/handlers"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func main() {

	// Инициализация конфигурации
	cfg := config.InitConfig()

	// Проверка корректности конфигурации
	if err := cfg.Validate(); err != nil {
		fmt.Printf("Ошибка конфигурации: %v\n", err)
		return
	}

	// Передача базового URL в обработчики
	handlers.SetBaseURL(cfg.BaseURL)

	r := chi.NewRouter()
	r.Post("/", handlers.ReceiveURL)
	r.Get("/{id}", handlers.ResponseURL)

	fmt.Printf("Сервер запущен на %s\n", cfg.ServerAddress)
	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		fmt.Printf("Ошибка при запуске сервера: %v\n", err)
	}

}
